package resources

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"

	"gopkg.in/yaml.v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"

	"github.com/grafana/grafana-app-sdk/logging"
	"github.com/grafana/grafana/pkg/apimachinery/apis/common/v0alpha1"
	"github.com/grafana/grafana/pkg/apimachinery/utils"
	provisioning "github.com/grafana/grafana/pkg/apis/provisioning/v0alpha1"
	"github.com/grafana/grafana/pkg/registry/apis/provisioning/lint"
	"github.com/grafana/grafana/pkg/registry/apis/provisioning/repository"
)

var ErrNamespaceMismatch = errors.New("the file namespace does not match target namespace")

type ParserFactory struct {
	Client *ClientFactory
}

func (f *ParserFactory) GetParser(ctx context.Context, repo repository.Repository) (*Parser, error) {
	config := repo.Config()
	client, kinds, err := f.Client.New(config.Namespace) // As system user
	if err != nil {
		return nil, err
	}

	parser := &Parser{
		repo:   config,
		client: client,
		kinds:  kinds,
	}
	// TODO: Figure out how we want to determine this in practice.
	linting, ok := os.LookupEnv("GRAFANA_LINTING")
	if ok && linting == "true" {
		linterFactory := lint.NewDashboardLinterFactory()
		cfg, err := repo.Read(ctx, linterFactory.ConfigPath(), "")

		logger := logging.FromContext(ctx)
		var linter lint.Linter
		switch {
		case err == nil:
			logger.Info("linter config found", "config", string(cfg.Data))
			linter, err = linterFactory.NewFromConfig(cfg.Data)
			if err != nil {
				return nil, fmt.Errorf("failed to create linter: %w", err)
			}
		case apierrors.IsNotFound(err):
			logger.Info("no linter config found, using default")
			linter = linterFactory.New()
		default:
			return nil, fmt.Errorf("failed to read linter config: %w", err)
		}

		parser.linter = linter
	}

	return parser, nil
}

type Parser struct {
	// The target repository
	repo *provisioning.Repository

	// client helper (for this namespace?)
	client *DynamicClient
	kinds  KindsLookup
	linter lint.Linter
}

type ParsedResource struct {
	// Original file Info
	Info *repository.FileInfo

	// Check for classic file types (dashboard.json, etc)
	Classic provisioning.ClassicFileType

	// Parsed contents
	Obj *unstructured.Unstructured
	// Metadata accessor for the file object
	Meta utils.GrafanaMetaAccessor

	// The Kind is defined in the file
	GVK *schema.GroupVersionKind
	// The Resource is found by mapping Kind to the right apiserver
	GVR *schema.GroupVersionResource
	// Client that can talk to this resource
	Client dynamic.ResourceInterface

	// The Existing object (same name)
	// ?? do we need/want the whole thing??
	Existing *unstructured.Unstructured

	// Create or Update
	Action provisioning.ResourceAction

	// The results from dry run
	DryRunResponse *unstructured.Unstructured

	// Optional lint issues
	Lint []provisioning.LintIssue

	// If we got some Errors
	Errors []error
}

func (r *Parser) Client() *DynamicClient {
	return r.client
}

func (r *Parser) Parse(ctx context.Context, info *repository.FileInfo, validate bool) (parsed *ParsedResource, err error) {
	logger := logging.FromContext(ctx).With("path", info.Path, "validate", validate)
	parsed = &ParsedResource{
		Info: info,
	}

	if ShouldIgnorePath(info.Path) {
		return parsed, ErrUnableToReadResourceBytes
	}

	parsed.Obj, parsed.GVK, err = DecodeYAMLObject(bytes.NewBuffer(info.Data))
	if err != nil {
		logger.Debug("failed to find GVK of the input data", "error", err)
		parsed.Obj, parsed.GVK, parsed.Classic, err = ReadClassicResource(ctx, info)
		if err != nil {
			logger.Debug("also failed to get GVK from fallback loader?", "error", err)
			return parsed, err
		}
	}

	parsed.Meta, err = utils.MetaAccessor(parsed.Obj)
	if err != nil {
		return nil, err
	}
	obj := parsed.Obj
	cfg := r.repo

	// Validate the namespace
	if obj.GetNamespace() != "" && obj.GetNamespace() != cfg.GetNamespace() {
		parsed.Errors = append(parsed.Errors, ErrNamespaceMismatch)
	}

	obj.SetNamespace(cfg.GetNamespace())
	parsed.Meta.SetRepositoryInfo(&utils.ResourceRepositoryInfo{
		Name:      cfg.Name,
		Path:      info.Path, // joinPathWithRef(info.Path, info.Ref),
		Hash:      info.Hash,
		Timestamp: nil, // ???&info.Modified.Time,
	})

	objName, folderName := NamesFromHashedRepoPath(cfg.Name, info.Path)
	obj.SetName(objName)
	obj.SetUID("")             // clear identifiers
	obj.SetResourceVersion("") // clear identifiers
	parsed.Meta.SetFolder(folderName)

	// We can not do anything more if no kind is defined
	if parsed.GVK == nil {
		return parsed, nil
	}

	gvr, ok := r.kinds.Resource(*parsed.GVK)
	if !ok {
		return parsed, nil
	}

	parsed.GVR = &gvr
	parsed.Client = r.client.Resource(gvr)
	if !validate {
		return parsed, nil
	}

	if r.linter != nil { // lint
		raw := info.Data
		if parsed.Classic == provisioning.ClassicDashboard {
			raw, err = json.MarshalIndent(parsed.Obj, "", "  ") // indent so it is not all on one line
			if err != nil {
				return parsed, err
			}
		}
		parsed.Lint, err = r.linter.Lint(ctx, raw)
		if err != nil {
			parsed.Errors = append(parsed.Errors, err)
		}
	}

	if parsed.Client == nil {
		parsed.Errors = append(parsed.Errors, fmt.Errorf("unable to find client"))
		return parsed, nil
	}

	// Dry run CREATE or UPDATE
	parsed.Existing, _ = parsed.Client.Get(ctx, obj.GetName(), metav1.GetOptions{})
	if parsed.Existing == nil {
		parsed.Action = provisioning.ResourceActionCreate
		parsed.DryRunResponse, err = parsed.Client.Create(ctx, obj, metav1.CreateOptions{
			DryRun: []string{"All"},
		})
	} else {
		parsed.Action = provisioning.ResourceActionUpdate
		parsed.DryRunResponse, err = parsed.Client.Update(ctx, obj, metav1.UpdateOptions{
			DryRun: []string{"All"},
		})
	}
	if err != nil {
		parsed.Errors = append(parsed.Errors, err)
	}
	return parsed, nil
}

func (f *ParsedResource) ToSaveBytes() ([]byte, error) {
	// TODO? should we use the dryRun (validated) version?
	obj := make(map[string]any)
	for k, v := range f.Obj.Object {
		if k != "metadata" {
			obj[k] = v
		}
	}

	switch path.Ext(f.Info.Path) {
	// JSON pretty print
	case ".json":
		return json.MarshalIndent(obj, "", "  ")

	// Write the value as yaml
	case ".yaml", ".yml":
		return yaml.Marshal(obj)

	default:
		return nil, fmt.Errorf("unexpected format")
	}
}

func (f *ParsedResource) AsResourceWrapper() *provisioning.ResourceWrapper {
	info := f.Info
	res := provisioning.ResourceObjects{
		Type: provisioning.ResourceType{
			Classic: f.Classic,
		},
		Action: f.Action,
	}

	if f.GVK != nil {
		res.Type.Group = f.GVK.Group
		res.Type.Version = f.GVK.Version
		res.Type.Kind = f.GVK.Kind
	}

	// The resource (GVR) is derived from the kind (GVK)
	if f.GVR != nil {
		res.Type.Resource = f.GVR.Resource
	}

	if f.Obj != nil {
		res.File = v0alpha1.Unstructured{Object: f.Obj.Object}
	}
	if f.Existing != nil {
		res.Existing = v0alpha1.Unstructured{Object: f.Existing.Object}
	}
	if f.DryRunResponse != nil {
		res.DryRun = v0alpha1.Unstructured{Object: f.DryRunResponse.Object}
	}
	wrap := &provisioning.ResourceWrapper{
		Path:      info.Path,
		Ref:       info.Ref,
		Hash:      info.Hash,
		Timestamp: info.Modified,
		Resource:  res,
		Lint:      f.Lint,
	}
	for _, err := range f.Errors {
		wrap.Errors = append(wrap.Errors, err.Error())
	}
	return wrap
}

// ShouldIgnorePath determines if the path given is worth looking at.
// If this returns true, skip processing it.
// JSON and YAML files are valid paths. Anything else isn't.
func ShouldIgnorePath(p string) bool {
	ext := path.Ext(p)
	// .yaml is the official extension per the spec, but .yml is widespread, too. (even this repo uses the unofficial one a lot!)
	return ext != ".yml" && ext != ".yaml" &&
		// We only support YAML, but JSON is sometimes generated from other tools, and is a valid subset of YAML.
		ext != ".json"
}
