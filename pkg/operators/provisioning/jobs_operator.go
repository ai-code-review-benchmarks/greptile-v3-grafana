package provisioning

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/urfave/cli/v2"
	"k8s.io/client-go/tools/cache"

	"github.com/grafana/grafana/pkg/registry/apis/provisioning/jobs"
	"github.com/grafana/grafana/pkg/registry/apis/provisioning/jobs/export"
	"github.com/grafana/grafana/pkg/registry/apis/provisioning/jobs/migrate"
	"github.com/grafana/grafana/pkg/registry/apis/provisioning/jobs/move"
	"github.com/grafana/grafana/pkg/registry/apis/provisioning/jobs/sync"
	"github.com/grafana/grafana/pkg/registry/apis/provisioning/resources"
	"github.com/grafana/grafana/pkg/services/apiserver/standalone"
	"github.com/grafana/grafana/pkg/setting"
	"github.com/grafana/grafana/pkg/storage/unified/resource"

	"github.com/grafana/grafana/apps/provisioning/pkg/controller"
	informer "github.com/grafana/grafana/apps/provisioning/pkg/generated/informers/externalversions"
	"github.com/grafana/grafana/apps/provisioning/pkg/repository"
	deletepkg "github.com/grafana/grafana/pkg/registry/apis/provisioning/jobs/delete"
)

func RunJobController(opts standalone.BuildInfo, c *cli.Context, cfg *setting.Cfg) error {
	logger := logging.NewSLogLogger(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})).With("logger", "provisioning-job-controller")
	logger.Info("Starting provisioning job controller")

	controllerCfg, err := getJobsControllerConfig(cfg)
	if err != nil {
		return fmt.Errorf("failed to setup operator: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("Received shutdown signal, stopping controllers")
		cancel()
	}()

	// Jobs informer and controller (resync ~60s like in register.go)
	jobInformerFactory := informer.NewSharedInformerFactoryWithOptions(
		controllerCfg.provisioningClient,
		controllerCfg.resyncInterval,
	)
	jobInformer := jobInformerFactory.Provisioning().V0alpha1().Jobs()
	jobController, err := controller.NewJobController(jobInformer)
	if err != nil {
		return fmt.Errorf("failed to create job controller: %w", err)
	}

	logger.Info("jobs controller started")

	var startHistoryInformers func()
	if controllerCfg.historyExpiration > 0 {
		// History jobs informer and controller (separate factory with resync == expiration)
		historyInformerFactory := informer.NewSharedInformerFactoryWithOptions(
			controllerCfg.provisioningClient,
			controllerCfg.historyExpiration,
		)
		historyJobInformer := historyInformerFactory.Provisioning().V0alpha1().HistoricJobs()
		_, err = controller.NewHistoryJobController(
			controllerCfg.provisioningClient.ProvisioningV0alpha1(),
			historyJobInformer,
			controllerCfg.historyExpiration,
		)
		if err != nil {
			return fmt.Errorf("failed to create history job controller: %w", err)
		}
		logger.Info("history cleanup enabled", "expiration", controllerCfg.historyExpiration.String())
		startHistoryInformers = func() { historyInformerFactory.Start(ctx.Done()) }
	} else {
		startHistoryInformers = func() {}
	}
	// HistoryWriter can be either Loki or the API server
	// TODO: Loki configuration and setup in the same way we do for the API server
	// https://github.com/grafana/git-ui-sync-project/issues/508
	// var jobHistoryWriter jobs.HistoryWriter
	// if b.jobHistoryLoki != nil {
	// 	jobHistoryWriter = b.jobHistoryLoki
	// } else {
	// 	jobHistoryWriter = jobs.NewAPIClientHistoryWriter(provisioningClient.ProvisioningV0alpha1())
	// }

	jobHistoryWriter := jobs.NewAPIClientHistoryWriter(controllerCfg.provisioningClient.ProvisioningV0alpha1())
	jobStore, err := jobs.NewJobStore(controllerCfg.provisioningClient.ProvisioningV0alpha1(), 30*time.Second)
	if err != nil {
		return fmt.Errorf("create API client job store: %w", err)
	}

	configProvider := NewDirectConfigProvider(controllerCfg.restCfg)
	clients := resources.NewClientFactory(configProvider)
	parsers := resources.NewParserFactory(clients)

	// HACK: This logic directly connects to unified storage. We are doing this for now as there is no global
	// search endpoint. But controllers, in general, should not connect directly to unified storage and instead
	// go through the api server. Once there is a global search endpoint, we will switch to that here as well.
	unified, err := NewUnifiedStorageClientFactory(controllerCfg.unifiedCfg, controllerCfg.tracer)
	if err != nil {
		return fmt.Errorf("create unified storage client: %w", err)
	}

	resourceLister := resources.NewResourceLister(unified)
	repositoryResources := resources.NewRepositoryResourcesFactory(parsers, clients, resourceLister)

	stageIfPossible := repository.WrapWithStageAndPushIfPossible
	exportWorker := export.NewExportWorker(
		clients,
		repositoryResources,
		export.ExportAll,
		stageIfPossible,
	)

	statusPatcher := controller.NewRepositoryStatusPatcher(controllerCfg.provisioningClient.ProvisioningV0alpha1())
	syncer := sync.NewSyncer(sync.Compare, sync.FullSync, sync.IncrementalSync)
	syncWorker := sync.NewSyncWorker(
		clients,
		repositoryResources,
		nil, // HACK: we have updated the worker to check for nil
		statusPatcher.Patch,
		syncer,
	)

	cleaner := migrate.NewNamespaceCleaner(clients)
	unifiedStorageMigrator := migrate.NewUnifiedStorageMigrator(
		cleaner,
		exportWorker,
		syncWorker,
	)

	migrationWorker := migrate.NewMigrationWorkerFromUnified(unifiedStorageMigrator)
	deleteWorker := deletepkg.NewWorker(syncWorker, stageIfPossible, repositoryResources)
	moveWorker := move.NewWorker(syncWorker, stageIfPossible, repositoryResources)

	workers := []jobs.Worker{
		deleteWorker,
		exportWorker,
		migrationWorker,
		moveWorker,
		syncWorker,
	}

	// This is basically our own JobQueue system
	driver, err := jobs.NewConcurrentJobDriver(
		3,              // 3 drivers for now
		20*time.Minute, // Max time for each job
		time.Minute,    // Cleanup jobs
		30*time.Second, // Periodically look for new jobs
		30*time.Second, // Lease renewal interval
		jobStore,
		controllerCfg.repoGetter,
		jobHistoryWriter,
		jobController.InsertNotifications(),
		workers...,
	)
	if err != nil {
		return fmt.Errorf("create concurrent job driver: %w", err)
	}

	go func() {
		logger.Info("jobs controller started")
		if err := driver.Run(ctx); err != nil {
			logger.Error("job driver failed", "error", err)
		}
	}()

	// Start informers
	go jobInformerFactory.Start(ctx.Done())
	go startHistoryInformers()

	// Optionally wait for job cache sync; history cleanup can rely on resync events
	if !cache.WaitForCacheSync(ctx.Done(), jobInformer.Informer().HasSynced) {
		return fmt.Errorf("failed to sync job informer cache")
	}

	<-ctx.Done()
	return nil
}

type jobsControllerConfig struct {
	provisioningControllerConfig
	historyExpiration time.Duration
	unifiedCfg        unifiedStorageConfig
}

func getJobsControllerConfig(cfg *setting.Cfg) (*jobsControllerConfig, error) {
	controllerCfg, err := setupFromConfig(cfg)
	if err != nil {
		return nil, err
	}

	// Unified Storage
	// TODO: This is duplicate
	gRPCAuth := cfg.SectionWithEnvOverrides("grpc_client_authentication")
	token := gRPCAuth.Key("token").String()
	if token == "" {
		return nil, fmt.Errorf("token is required in [grpc_client_authentication] section")
	}
	tokenExchangeURL := gRPCAuth.Key("token_exchange_url").String()
	if tokenExchangeURL == "" {
		return nil, fmt.Errorf("token_exchange_url is required in [grpc_client_authentication] section")
	}

	tokenNamespace := gRPCAuth.Key("token_namespace").String()
	allowInsecure := gRPCAuth.Key("allow_insecure").MustBool(false)
	// TODO: enforce this
	unifiedStorageSec := cfg.SectionWithEnvOverrides("unified_storage")
	unifiedCfg := unifiedStorageConfig{
		GrpcAddress:      unifiedStorageSec.Key("grpc_address").String(),
		GrpcIndexAddress: unifiedStorageSec.Key("grpc_index_address").String(),
		ClientConfig: resource.RemoteResourceClientConfig{
			Token:            token,
			TokenExchangeURL: tokenExchangeURL,
			// TODO: why do we get this?
			// Audiences:     unifiedStorageSec.Key("grpc_client_authentication_audiences").Strings(","),
			Namespace:     tokenNamespace,
			AllowInsecure: allowInsecure,
		},
	}

	return &jobsControllerConfig{
		provisioningControllerConfig: *controllerCfg,
		historyExpiration:            cfg.SectionWithEnvOverrides("operator").Key("history_expiration").MustDuration(0),
		unifiedCfg:                   unifiedCfg,
	}, nil
}
