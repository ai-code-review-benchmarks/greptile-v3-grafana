package provisioning

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/util/workqueue"

	"github.com/grafana/grafana/pkg/apimachinery/identity"
	provisioning "github.com/grafana/grafana/pkg/apis/provisioning/v0alpha1"
	client "github.com/grafana/grafana/pkg/generated/clientset/versioned/typed/provisioning/v0alpha1"
	informer "github.com/grafana/grafana/pkg/generated/informers/externalversions/provisioning/v0alpha1"
	listers "github.com/grafana/grafana/pkg/generated/listers/provisioning/v0alpha1"
	"github.com/grafana/grafana/pkg/registry/apis/provisioning/auth"
	"github.com/grafana/grafana/pkg/registry/apis/provisioning/jobs"
)

// RepositoryController controls how and when CRD is established.
type RepositoryController struct {
	client     client.ProvisioningV0alpha1Interface
	repoLister listers.RepositoryLister
	repoSynced cache.InformerSynced

	jobs jobs.JobQueue

	// Converts config to instance
	repoGetter RepoGetter
	identities auth.BackgroundIdentityService
	tester     *RepositoryTester

	// To allow injection for testing.
	syncFn            func(key string) error
	enqueueRepository func(obj any)
	keyFunc           func(obj any) (string, error)

	queue  workqueue.TypedRateLimitingInterface[string]
	logger *slog.Logger
}

// NewRepositoryController creates new RepositoryController.
func NewRepositoryController(
	provisioningClient client.ProvisioningV0alpha1Interface,
	repoInformer informer.RepositoryInformer,
	repoGetter RepoGetter,
	identities auth.BackgroundIdentityService,
	tester *RepositoryTester,
	jobs jobs.JobQueue,
) (*RepositoryController, error) {
	rc := &RepositoryController{
		client:     provisioningClient,
		repoLister: repoInformer.Lister(),
		repoSynced: repoInformer.Informer().HasSynced,
		queue: workqueue.NewTypedRateLimitingQueueWithConfig(
			workqueue.DefaultTypedControllerRateLimiter[string](),
			workqueue.TypedRateLimitingQueueConfig[string]{
				Name: "provisioningRepositoryController",
			},
		),
		repoGetter: repoGetter,
		identities: identities,
		tester:     tester,
		logger:     slog.Default().With("logger", "provisioning-repository-controller"),
		jobs:       jobs,
	}

	_, err := repoInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    rc.addRepository,
		UpdateFunc: rc.updateRepository,
		DeleteFunc: rc.deleteRepository,
	})
	if err != nil {
		return nil, err
	}

	rc.syncFn = rc.sync
	rc.enqueueRepository = rc.enqueue
	rc.keyFunc = repoKeyFunc

	return rc, nil
}

func repoKeyFunc(obj any) (string, error) {
	repo, ok := obj.(*provisioning.Repository)
	if !ok {
		return "", fmt.Errorf("expected a Repository but got %T", obj)
	}
	return cache.DeletionHandlingMetaNamespaceKeyFunc(repo)
}

// Run starts the RepositoryController.
func (rc *RepositoryController) Run(ctx context.Context, workerCount int) {
	defer utilruntime.HandleCrash()
	defer rc.queue.ShutDown()

	rc.logger.Info("Starting RepositoryController")
	defer rc.logger.Info("Shutting down RepositoryController")

	if !cache.WaitForCacheSync(ctx.Done(), rc.repoSynced) {
		return
	}

	rc.logger.Info("Starting workers", "count", workerCount)
	for i := 0; i < workerCount; i++ {
		go wait.UntilWithContext(ctx, rc.runWorker, time.Second)
	}

	rc.logger.Info("Started workers")
	<-ctx.Done()
	rc.logger.Info("Shutting down workers")
}

func (rc *RepositoryController) runWorker(ctx context.Context) {
	for rc.processNextWorkItem(ctx) {
	}
}

func (rc *RepositoryController) enqueue(obj interface{}) {
	key, err := rc.keyFunc(obj)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("couldn't get key for object: %v", err))
		return
	}
	rc.queue.Add(key)
}

func (rc *RepositoryController) addRepository(obj interface{}) {
	rc.enqueueRepository(obj)
}

func (rc *RepositoryController) updateRepository(oldObj, newObj interface{}) {
	rc.enqueueRepository(newObj)
}

func (rc *RepositoryController) deleteRepository(obj interface{}) {
	rc.enqueueRepository(obj)
}

// processNextWorkItem deals with one key off the queue.
// It returns false when it's time to quit.
func (rc *RepositoryController) processNextWorkItem(_ context.Context) bool {
	key, quit := rc.queue.Get()
	if quit {
		return false
	}
	defer rc.queue.Done(key)

	rc.logger.Info("RepositoryController processing key", "key", key)

	err := rc.syncFn(key)
	if err == nil {
		rc.queue.Forget(key)
		return true
	}

	utilruntime.HandleError(fmt.Errorf("%v failed with: %v", key, err))
	rc.queue.AddRateLimited(key)

	return true
}

// sync is the business logic of the controller.
func (rc *RepositoryController) sync(key string) error {
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return err
	}

	cachedRepo, err := rc.repoLister.Repositories(namespace).Get(name)
	if apierrors.IsNotFound(err) {
		return nil
	}
	if err != nil {
		return err
	}

	ctx := context.Background()
	id, err := rc.identities.WorkerIdentity(ctx, cachedRepo.Namespace)
	if err != nil {
		return err
	}
	ctx = identity.WithRequester(ctx, id)
	logger := rc.logger.With("repository", cachedRepo.Name, "namespace", cachedRepo.Namespace)

	repo, err := rc.repoGetter.AsRepository(ctx, cachedRepo)
	if err != nil {
		return fmt.Errorf("unable to create repository from configuration: %w", err)
	}

	// The repository is deleted
	if cachedRepo.DeletionTimestamp != nil {
		return repo.OnDelete(ctx, logger)
	}

	// Did the spec change
	now := time.Now().UnixMilli()
	generationChanged := cachedRepo.Generation != cachedRepo.Status.Health.Generation
	elapsed := time.Duration(now-cachedRepo.Status.Health.Checked) * time.Millisecond
	if elapsed < time.Millisecond*200 {
		// avoids possible inf loop!!!
		return nil
	}
	if elapsed < time.Second*30 && !generationChanged {
		// We checked status recently! and the generation has not changed
		return nil
	}

	// HACK: Use health status to check if it's update or create
	// probably this will prevent some updates to be processed.
	// QUESTION: should we add these errors to the health status?
	var status *provisioning.RepositoryStatus

	isUpdate := cachedRepo.Status.Health.Checked > 0
	if isUpdate {
		status, err = repo.OnUpdate(ctx, logger)
		if err != nil {
			return fmt.Errorf("on create repository: %w", err)
		}
	} else {
		status, err = repo.OnCreate(ctx, logger)
		if err != nil {
			return fmt.Errorf("on create repository: %w", err)
		}
	}

	if status == nil {
		status = cachedRepo.Status.DeepCopy()
	}

	status.Health.Checked = time.Now().UnixMilli()
	status.Health.Generation = cachedRepo.Generation

	res, err := rc.tester.TestRepository(ctx, repo)
	if err != nil {
		res = &provisioning.TestResults{
			Success: false,
			Errors: []string{
				"error running test repository",
				err.Error(),
			},
		}
	}

	status.Health = provisioning.HealthStatus{
		Healthy: res.Success,
		Message: res.Errors,
	}

	cfg := cachedRepo.DeepCopy()
	cfg.Status = *status
	if _, err := rc.client.Repositories(cachedRepo.GetNamespace()).
		UpdateStatus(ctx, cfg, metav1.UpdateOptions{}); err != nil {
		return fmt.Errorf("update status: %w", err)
	}

	job, err := rc.jobs.Add(ctx, &provisioning.Job{
		ObjectMeta: v1.ObjectMeta{
			Namespace: cfg.Namespace,
			Labels: map[string]string{
				"repository": cfg.Name,
			},
		},
		Spec: provisioning.JobSpec{
			Action: provisioning.JobActionSync,
		},
	})

	logger.Info("sync job created", "job", job.Name)

	return nil
}
