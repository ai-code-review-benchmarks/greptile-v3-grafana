package secret

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"

	secretv0alpha1 "github.com/grafana/grafana/pkg/apis/secret/v0alpha1"
	"github.com/grafana/grafana/pkg/services/featuremgmt"
	"github.com/grafana/grafana/pkg/tests/apis"
	"github.com/grafana/grafana/pkg/tests/testinfra"
	"github.com/grafana/grafana/pkg/tests/testsuite"
)

var gvrSecureValues = schema.GroupVersionResource{
	Group:    "secret.grafana.app",
	Version:  "v0alpha1",
	Resource: "securevalues",
}

func TestMain(m *testing.M) {
	testsuite.Run(m)
}

func TestIntegrationSecureValue(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	helper := apis.NewK8sTestHelper(t, testinfra.GrafanaOpts{
		AppModeProduction: false, // required for experimental APIs
		EnableFeatureToggles: []string{
			// Required to start the example service
			featuremgmt.FlagGrafanaAPIServerWithExperimentalAPIs,
			featuremgmt.FlagSecretsManagementAppPlatform,
		},
	})

	t.Run("check discovery client", func(t *testing.T) {
		disco := helper.NewDiscoveryClient()

		resources, err := disco.ServerResourcesForGroupVersion("secret.grafana.app/v0alpha1")
		require.NoError(t, err)

		v1Disco, err := json.MarshalIndent(resources, "", "  ")
		require.NoError(t, err)

		var apiResourceList map[string]any
		require.NoError(t, json.Unmarshal(v1Disco, &apiResourceList))

		groupVersion, ok := apiResourceList["groupVersion"].(string)
		require.True(t, ok)
		require.Equal(t, "secret.grafana.app/v0alpha1", groupVersion)

		apiResources, ok := apiResourceList["resources"].([]any)
		require.True(t, ok)
		require.Len(t, apiResources, 2) // securevalue + keeper + (subresources...)
	})

	t.Run("creating a secure value without a name generates one", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		client := helper.GetResourceClient(apis.ResourceClientArgs{
			// #TODO: figure out permissions topic
			User: helper.Org1.Admin,
			GVR:  gvrSecureValues,
		})

		testDataSecureValue := helper.LoadYAMLOrJSONFile("testdata/secure-value-generate.yaml")

		raw, err := client.Resource.Create(ctx, testDataSecureValue, metav1.CreateOptions{})
		require.NoError(t, err)
		require.NotNil(t, raw)

		t.Cleanup(func() {
			require.NoError(t, client.Resource.Delete(ctx, raw.GetName(), metav1.DeleteOptions{}))
		})

		secureValue := new(secretv0alpha1.SecureValue)
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(raw.Object, secureValue)
		require.NoError(t, err)
		require.NotNil(t, secureValue)

		require.NotEmpty(t, secureValue.Name)
	})

	t.Run("creating a secure value returns it without any of the value or ref", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		client := helper.GetResourceClient(apis.ResourceClientArgs{
			// #TODO: figure out permissions topic
			User: helper.Org1.Admin,
			GVR:  gvrSecureValues,
		})

		testDataSecureValueXyz := helper.LoadYAMLOrJSONFile("testdata/secure-value-xyz.yaml")

		raw, err := client.Resource.Create(ctx, testDataSecureValueXyz, metav1.CreateOptions{})
		require.NoError(t, err)
		require.NotNil(t, raw)

		t.Cleanup(func() {
			require.NoError(t, client.Resource.Delete(ctx, raw.GetName(), metav1.DeleteOptions{}))
		})

		secureValue := new(secretv0alpha1.SecureValue)
		err = runtime.DefaultUnstructuredConverter.FromUnstructured(raw.Object, secureValue)
		require.NoError(t, err)
		require.NotNil(t, secureValue)

		require.Empty(t, secureValue.Spec.Value)
		require.Empty(t, secureValue.Spec.Ref)
		require.NotEmpty(t, secureValue.Spec.Title)
		require.NotEmpty(t, secureValue.Spec.Keeper)
		require.NotEmpty(t, secureValue.Spec.Audiences)

		t.Run("and creating another secure value with the same name in the same namespace returns an error", func(t *testing.T) {
			raw, err := client.Resource.Create(ctx, testDataSecureValueXyz, metav1.CreateOptions{})
			require.Error(t, err)
			require.Nil(t, raw)
		})

		t.Run("and reading the secure value returns it same as if when it was created", func(t *testing.T) {
			raw, err := client.Resource.Get(ctx, secureValue.Name, metav1.GetOptions{})
			require.NoError(t, err)
			require.NotNil(t, raw)

			anotherSecureValue := new(secretv0alpha1.SecureValue)
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(raw.Object, anotherSecureValue)
			require.NoError(t, err)
			require.NotNil(t, anotherSecureValue)

			require.EqualValues(t, secureValue, anotherSecureValue)
		})

		t.Run("and listing securevalues returns the created secure value", func(t *testing.T) {
			rawList, err := client.Resource.List(ctx, metav1.ListOptions{})
			require.NoError(t, err)
			require.NotNil(t, rawList)
			require.GreaterOrEqual(t, len(rawList.Items), 1)
			require.Equal(t, secureValue.Name, rawList.Items[0].GetName())
		})

		t.Run("and updating the secure value replaces the spec fields and returns them", func(t *testing.T) {
			newRaw := testDataSecureValueXyz.DeepCopy()
			newRaw.Object["spec"].(map[string]any)["title"] = "New title"
			newRaw.Object["spec"].(map[string]any)["keeper"] = "New keeper"
			newRaw.Object["spec"].(map[string]any)["value"] = "New secure value"
			newRaw.Object["spec"].(map[string]any)["audiences"] = []string{"audience1/name1", "audience2/*"}
			newRaw.Object["metadata"].(map[string]any)["annotations"] = map[string]any{"newAnnotation": "newValue"}

			updatedRaw, err := client.Resource.Update(ctx, newRaw, metav1.UpdateOptions{})
			require.NoError(t, err)
			require.NotNil(t, updatedRaw)

			updatedSecureValue := new(secretv0alpha1.SecureValue)
			err = runtime.DefaultUnstructuredConverter.FromUnstructured(updatedRaw.Object, updatedSecureValue)
			require.NoError(t, err)
			require.NotNil(t, updatedSecureValue)

			require.NotEqualValues(t, updatedSecureValue.Spec, secureValue.Spec)
		})
	})

	t.Run("reading a secure value that does not exist returns a 404", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		client := helper.GetResourceClient(apis.ResourceClientArgs{
			// #TODO: figure out permissions topic
			User: helper.Org1.Admin,
			GVR:  gvrSecureValues,
		})

		raw, err := client.Resource.Get(ctx, "some-secure-value-that-does-not-exist", metav1.GetOptions{})
		require.Error(t, err)
		require.Nil(t, raw)

		var statusErr *apierrors.StatusError
		require.True(t, errors.As(err, &statusErr))
		require.Equal(t, http.StatusNotFound, int(statusErr.Status().Code))
	})

	t.Run("deleting a secure value that does not exist does not return an error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		client := helper.GetResourceClient(apis.ResourceClientArgs{
			// #TODO: figure out permissions topic
			User: helper.Org1.Admin,
			GVR:  gvrSecureValues,
		})

		err := client.Resource.Delete(ctx, "some-secure-value-that-does-not-exist", metav1.DeleteOptions{})
		require.NoError(t, err)
	})

	t.Run("deleting a secure value that exists does not return an error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		t.Cleanup(cancel)

		client := helper.GetResourceClient(apis.ResourceClientArgs{
			// #TODO: figure out permissions topic
			User: helper.Org1.Admin,
			GVR:  gvrSecureValues,
		})

		generatePrefix := "generated-"

		testDataSecureValueXyz := helper.LoadYAMLOrJSONFile("testdata/secure-value-xyz.yaml")
		testDataSecureValueXyz.SetName("")
		testDataSecureValueXyz.SetGenerateName("generated-")

		raw, err := client.Resource.Create(ctx, testDataSecureValueXyz, metav1.CreateOptions{})
		require.NoError(t, err)
		require.NotNil(t, raw)

		name := raw.GetName()
		require.True(t, strings.HasPrefix(name, generatePrefix))

		err = client.Resource.Delete(ctx, name, metav1.DeleteOptions{})
		require.NoError(t, err)

		t.Run("and then trying to read it returns a 404 error", func(t *testing.T) {
			raw, err := client.Resource.Get(ctx, name, metav1.GetOptions{})
			require.Error(t, err)
			require.Nil(t, raw)

			var statusErr *apierrors.StatusError
			require.True(t, errors.As(err, &statusErr))
			require.Equal(t, http.StatusNotFound, int(statusErr.Status().Code))
		})

		t.Run("and listing secure values returns an empty list", func(t *testing.T) {
			rawList, err := client.Resource.List(ctx, metav1.ListOptions{})
			require.NoError(t, err)
			require.NotNil(t, rawList)
			require.Empty(t, rawList.Items)
		})
	})
}
