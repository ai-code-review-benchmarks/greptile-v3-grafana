package reststorage

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	claims "github.com/grafana/authlib/types"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"k8s.io/apimachinery/pkg/apis/meta/internalversion"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/apiserver/pkg/admission"
	"k8s.io/apiserver/pkg/endpoints/request"
	"k8s.io/apiserver/pkg/registry/rest"

	"github.com/grafana/grafana/pkg/apimachinery/utils"

	secretv0alpha1 "github.com/grafana/grafana/pkg/apis/secret/v0alpha1"
	"github.com/grafana/grafana/pkg/registry/apis/secret/contracts"
	"github.com/grafana/grafana/pkg/registry/apis/secret/service"
	"github.com/grafana/grafana/pkg/registry/apis/secret/xkube"
)

var (
	_ rest.Scoper               = (*SecureValueRest)(nil)
	_ rest.SingularNameProvider = (*SecureValueRest)(nil)
	_ rest.Getter               = (*SecureValueRest)(nil)
	_ rest.Lister               = (*SecureValueRest)(nil)
	_ rest.Storage              = (*SecureValueRest)(nil)
	_ rest.Creater              = (*SecureValueRest)(nil)
	_ rest.Updater              = (*SecureValueRest)(nil)
	_ rest.GracefulDeleter      = (*SecureValueRest)(nil)
)

// SecureValueRest is an implementation of CRUDL operations on a `securevalue` backed by a persistence layer `store`.
type SecureValueRest struct {
	secureValueService *service.SecureValueService
	resource           utils.ResourceInfo
	tableConverter     rest.TableConvertor
	tracer             trace.Tracer
}

// NewSecureValueRest is a returns a constructed `*SecureValueRest`.
func NewSecureValueRest(tracer trace.Tracer, secureValueService *service.SecureValueService, resource utils.ResourceInfo) *SecureValueRest {
	return &SecureValueRest{
		secureValueService: secureValueService,
		resource:           resource,
		tableConverter:     resource.TableConverter(),
		tracer:             tracer,
	}
}

// New returns an empty `*SecureValue` that is used by the `Create` method.
func (s *SecureValueRest) New() runtime.Object {
	return s.resource.NewFunc()
}

// Destroy is called when? [TODO]
func (s *SecureValueRest) Destroy() {}

// NamespaceScoped returns `true` because the storage is namespaced (== org).
func (s *SecureValueRest) NamespaceScoped() bool {
	return true
}

// GetSingularName is used by `kubectl` discovery to have singular name representation of resources.
func (s *SecureValueRest) GetSingularName() string {
	return s.resource.GetSingularName()
}

// NewList returns an empty `*SecureValueList` that is used by the `List` method.
func (s *SecureValueRest) NewList() runtime.Object {
	return s.resource.NewListFunc()
}

// ConvertToTable is used by Kubernetes and converts objects to `metav1.Table`.
func (s *SecureValueRest) ConvertToTable(ctx context.Context, object runtime.Object, tableOptions runtime.Object) (*metav1.Table, error) {
	return s.tableConverter.ConvertToTable(ctx, object, tableOptions)
}

// List calls the inner `store` (persistence) and returns a list of `securevalues` within a `namespace` filtered by the `options`.
func (s *SecureValueRest) List(ctx context.Context, options *internalversion.ListOptions) (runtime.Object, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, fmt.Errorf("missing namespace")
	}

	ctx, span := s.tracer.Start(ctx, "SecureValueRest.List", trace.WithAttributes(attribute.String("namespace", namespace)))
	defer span.End()

	secureValueList, err := s.secureValueService.List(ctx, xkube.Namespace(namespace))
	if err != nil {
		return nil, fmt.Errorf("failed to list secure values: %w", err)
	}

	labelSelector := options.LabelSelector

	if labelSelector == nil {
		labelSelector = labels.Everything()
	}

	fieldSelector := options.FieldSelector
	if fieldSelector == nil {
		fieldSelector = fields.Everything()
	}

	allowedSecureValues := make([]secretv0alpha1.SecureValue, 0, len(secureValueList.Items))

	for _, secureValue := range secureValueList.Items {
		// Filter by label
		if labelSelector.Matches(labels.Set(secureValue.Labels)) {
			// Filter by status.phase
			if fieldSelector.Matches(fields.Set{"status.phase": string(secureValue.Status.Phase)}) {
				allowedSecureValues = append(allowedSecureValues, secureValue)
			}
		}
	}

	return &secretv0alpha1.SecureValueList{Items: allowedSecureValues}, nil
}

// Get calls the inner `store` (persistence) and returns a `securevalue` by `name`. It will NOT return the decrypted `value`.
func (s *SecureValueRest) Get(ctx context.Context, name string, _ *metav1.GetOptions) (runtime.Object, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, fmt.Errorf("missing namespace")
	}

	ctx, span := s.tracer.Start(ctx, "SecureValueRest.Get", trace.WithAttributes(
		attribute.String("name", name),
		attribute.String("namespace", namespace),
	))
	defer span.End()

	sv, err := s.secureValueService.Read(ctx, xkube.Namespace(namespace), name)
	if err != nil {
		if errors.Is(err, contracts.ErrSecureValueNotFound) {
			return nil, s.resource.NewNotFound(name)
		}

		return nil, fmt.Errorf("failed to read secure value: %w", err)
	}

	return sv, nil
}

// Create a new `securevalue`. Does some validation and allows empty `name` (generated).
func (s *SecureValueRest) Create(
	ctx context.Context,
	obj runtime.Object,
	createValidation rest.ValidateObjectFunc,
	_ *metav1.CreateOptions,
) (runtime.Object, error) {
	sv, ok := obj.(*secretv0alpha1.SecureValue)
	if !ok {
		return nil, fmt.Errorf("expected SecureValue for create")
	}

	ctx, span := s.tracer.Start(ctx, "SecureValueRest.Create", trace.WithAttributes(
		attribute.String("name", sv.GetName()),
		attribute.String("namespace", sv.GetNamespace()),
	))
	defer span.End()

	if err := createValidation(ctx, obj); err != nil {
		return nil, err
	}

	user, ok := claims.AuthInfoFrom(ctx)
	if !ok {
		return nil, fmt.Errorf("missing auth info in context")
	}

	createdSecureValueMetadata, err := s.secureValueService.Create(ctx, sv, user.GetUID())
	if err != nil {
		return nil, fmt.Errorf("creating secure value %+w", err)
	}

	return createdSecureValueMetadata, nil
}

// Update a `securevalue`'s `value`. The second return parameter indicates whether the resource was newly created.
// Currently does not support "create on update" functionality. If the securevalue does not yet exist, it returns an error.
func (s *SecureValueRest) Update(
	ctx context.Context,
	name string,
	objInfo rest.UpdatedObjectInfo,
	_ rest.ValidateObjectFunc,
	updateValidation rest.ValidateObjectUpdateFunc,
	_forceAllowCreate bool,
	_ *metav1.UpdateOptions,
) (runtime.Object, bool, error) {
	ctx, span := s.tracer.Start(ctx, "SecureValueRest.Update", trace.WithAttributes(
		attribute.String("name", name),
		attribute.String("namespace", request.NamespaceValue(ctx)),
	))
	defer span.End()

	oldObj, err := s.Get(ctx, name, &metav1.GetOptions{})
	if err != nil {
		return nil, false, err
	}

	// Makes sure the UID and ResourceVersion are OK.
	// TODO: this also makes it so the labels and annotations are additive, unless we check and remove manually.
	newObj, err := objInfo.UpdatedObject(ctx, oldObj)
	if err != nil {
		return nil, false, fmt.Errorf("k8s updated object: %w", err)
	}

	if err := updateValidation(ctx, newObj, oldObj); err != nil {
		return nil, false, err
	}

	newSecureValue, ok := newObj.(*secretv0alpha1.SecureValue)
	if !ok {
		return nil, false, fmt.Errorf("expected SecureValue for update")
	}

	// TODO: do we need to do this here again? Probably not, but double-check!
	newSecureValue.Annotations = xkube.CleanAnnotations(newSecureValue.Annotations)

	user, ok := claims.AuthInfoFrom(ctx)
	if !ok {
		return nil, false, fmt.Errorf("missing auth info in context")
	}

	updatedSecureValueMetadata, _, err := s.secureValueService.Update(ctx, newSecureValue, user.GetUID())
	if err != nil {
		return updatedSecureValueMetadata, false, fmt.Errorf("updating secure value metadata: %+w", err)
	}

	return updatedSecureValueMetadata, false, nil
}

// The second return parameter `bool` indicates whether the delete was instant or not. It always is for `securevalues`.
func (s *SecureValueRest) Delete(ctx context.Context, name string, _ rest.ValidateObjectFunc, _ *metav1.DeleteOptions) (runtime.Object, bool, error) {
	namespace, ok := request.NamespaceFrom(ctx)
	if !ok {
		return nil, false, fmt.Errorf("missing namespace")
	}

	ctx, span := s.tracer.Start(ctx, "SecureValueRest.Delete", trace.WithAttributes(
		attribute.String("name", name),
		attribute.String("namespace", namespace),
	))
	defer span.End()

	updatedSv, err := s.secureValueService.Delete(ctx, xkube.Namespace(namespace), name)
	if err != nil {
		if errors.Is(err, contracts.ErrSecureValueNotFound) {
			return nil, false, s.resource.NewNotFound(name)
		}
		return nil, false, fmt.Errorf("deleting secure value: %+w", err)
	}

	return updatedSv, false, nil
}

// ValidateSecureValue does basic spec validation of a securevalue.
func ValidateSecureValue(sv, oldSv *secretv0alpha1.SecureValue, operation admission.Operation, decryptersAllowList map[string]struct{}) field.ErrorList {
	errs := make(field.ErrorList, 0)

	// Operation-specific field validation.
	switch operation {
	case admission.Create:
		errs = validateSecureValueCreate(sv)

	// If we plan to support PATCH-style updates, we shouldn't be requiring fields to be set.
	case admission.Update:
		errs = validateSecureValueUpdate(sv, oldSv)

	case admission.Delete:
	case admission.Connect:
	}

	// General validations.
	if len(sv.Spec.Value) > contracts.SECURE_VALUE_RAW_INPUT_MAX_SIZE_BYTES {
		errs = append(
			errs,
			field.TooLong(field.NewPath("spec", "value"), len(sv.Spec.Value), contracts.SECURE_VALUE_RAW_INPUT_MAX_SIZE_BYTES),
		)
	}

	if errs := validateDecrypters(sv.Spec.Decrypters, decryptersAllowList); len(errs) > 0 {
		return errs
	}

	return errs
}

// validateSecureValueCreate does basic spec validation of a securevalue for the Create operation.
func validateSecureValueCreate(sv *secretv0alpha1.SecureValue) field.ErrorList {
	errs := make(field.ErrorList, 0)

	if sv.Spec.Description == "" {
		errs = append(errs, field.Required(field.NewPath("spec", "description"), "a `description` is required"))
	}

	if sv.Spec.Value == "" && (sv.Spec.Ref == nil || (sv.Spec.Ref != nil && *sv.Spec.Ref == "")) {
		errs = append(errs, field.Required(field.NewPath("spec"), "either a `value` or `ref` is required"))
	}

	if sv.Spec.Value != "" && (sv.Spec.Ref != nil && *sv.Spec.Ref != "") {
		errs = append(errs, field.Forbidden(field.NewPath("spec"), "only one of `value` or `ref` can be set"))
	}

	return errs
}

// validateSecureValueUpdate does basic spec validation of a securevalue for the Update operation.
func validateSecureValueUpdate(sv, oldSv *secretv0alpha1.SecureValue) field.ErrorList {
	errs := make(field.ErrorList, 0)

	// For updates, an `old` object is required.
	if oldSv == nil {
		errs = append(errs, field.InternalError(field.NewPath("spec"), errors.New("old object is nil")))

		return errs
	}

	// Only validate if one of the fields is being changed/set.
	if sv.Spec.Value != "" || (sv.Spec.Ref != nil && *sv.Spec.Ref != "") {
		if (oldSv.Spec.Ref != nil && *oldSv.Spec.Ref != "") && sv.Spec.Value != "" {
			errs = append(errs, field.Forbidden(field.NewPath("spec"), "cannot set `value` when `ref` was already previously set"))
		}

		if (oldSv.Spec.Ref == nil || (oldSv.Spec.Ref != nil && *oldSv.Spec.Ref == "")) && (sv.Spec.Ref != nil && *sv.Spec.Ref != "") {
			errs = append(errs, field.Forbidden(field.NewPath("spec"), "cannot set `ref` when `value` was already previously set"))
		}
	}

	// Keeper cannot be changed.
	if sv.Spec.Keeper != oldSv.Spec.Keeper {
		errs = append(errs, field.Forbidden(field.NewPath("spec"), "the `keeper` cannot be changed"))
	}

	return errs
}

// validateDecrypters validates that (if populated) the `decrypters` must be unique.
func validateDecrypters(decrypters []string, decryptersAllowList map[string]struct{}) field.ErrorList {
	errs := make(field.ErrorList, 0)

	// Limit the number of decrypters to 64 to not have it unbounded.
	// The number was chosen arbitrarily and should be enough.
	if len(decrypters) > 64 {
		errs = append(
			errs,
			field.TooMany(field.NewPath("spec", "decrypters"), len(decrypters), 64),
		)

		return errs
	}

	decrypterNames := make(map[string]struct{}, 0)

	for i, decrypter := range decrypters {
		decrypter = strings.TrimSpace(decrypter)
		if decrypter == "" {
			errs = append(
				errs,
				field.Invalid(field.NewPath("spec", "decrypters", "["+strconv.Itoa(i)+"]"), decrypter, "decrypters cannot be empty if specified"),
			)

			continue
		}

		// Allow List: decrypters must match exactly and be in the allowed list to be able to decrypt.
		if len(decryptersAllowList) > 0 {
			if _, exists := decryptersAllowList[decrypter]; !exists {
				errs = append(
					errs,
					field.Invalid(field.NewPath("spec", "decrypters", "["+strconv.Itoa(i)+"]"), decrypter, fmt.Sprintf("allowed values: %v", decryptersAllowList)),
				)

				return errs
			}

			continue
		}

		// Use the same validation as labels for the decrypters.
		if verrs := validation.IsValidLabelValue(decrypter); len(verrs) > 0 {
			for _, verr := range verrs {
				errs = append(
					errs,
					field.Invalid(field.NewPath("spec", "decrypters", "["+strconv.Itoa(i)+"]"), decrypter, verr),
				)
			}

			continue
		}

		if _, exists := decrypterNames[decrypter]; exists {
			errs = append(
				errs,
				field.Invalid(field.NewPath("spec", "decrypters", "["+strconv.Itoa(i)+"]"), decrypter, "decrypters must be unique"),
			)

			continue
		}

		decrypterNames[decrypter] = struct{}{}
	}

	return errs
}
