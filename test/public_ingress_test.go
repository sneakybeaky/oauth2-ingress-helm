package test

import (
	"fmt"
	"k8s.io/api/networking/v1beta1"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/require"
)

const chartPath = "../chart/oauth2-ingress"
const publicIngressPath = "templates/public-ingress.yaml"

// Test the rendering of the public ingress template
func TestRenderedPublicIngress(t *testing.T) {
	t.Parallel()

	// Path to the helm chart we will test
	helmChartPath, err := filepath.Abs(chartPath)
	require.NoError(t, err)

	testCases := []*renderTest{
		ingressRenderTest(),
		tlsRenderTest(),
		annotationsRenderTest(),
	}

	for _, testCase := range testCases {
		// Here, we capture the range variable and force it into the scope of this block. If we don't do this, when the
		// subtest switches contexts (because of t.Parallel), the testCase value will have been updated by the for loop
		// and will be the next testCase!
		testCase := testCase

		// The actual sub test spawning. We name the sub test using the human friendly name. Note that we name the sub
		// test T struct to subT to make it clear which T struct corresponds to which test. However, in most cases you
		// will not reference the main test T so you can name it the same.
		t.Run(testCase.name, func(subT *testing.T) {
			subT.Parallel()

			// Render the template and verify the result
			output := helm.RenderTemplate(subT, testCase.options, helmChartPath, "render-public-ingress", []string{publicIngressPath})
			var ingress v1beta1.Ingress
			helm.UnmarshalK8SYaml(subT, output, &ingress)
			testCase.verify(subT, ingress)
		})
	}

}

type renderTest struct {
	name    string
	options *helm.Options
	verify  func(t *testing.T, got v1beta1.Ingress)
}

func ingressRenderTest() *renderTest {

	expectedHost := "www.erewhon.io"
	expectedPath := "/some/path"
	expectedServiceName := "protected-service"
	expectedServicePort := "9999"

	options := &helm.Options{
		SetValues: map[string]string{
			"ingress.host":                         expectedHost,
			"ingress.paths[0].backend.serviceName": expectedServiceName,
			"ingress.paths[0].backend.servicePort": expectedServicePort,
			"ingress.paths[0].path":                expectedPath,
		},
	}

	return &renderTest{
		name:    "Ingress defined",
		options: options,
		verify: func(t *testing.T, got v1beta1.Ingress) {

			rules := got.Spec.Rules
			require.Equal(t, 1, len(rules))
			require.Equal(t, expectedHost, rules[0].Host)

			paths := got.Spec.Rules[0].HTTP.Paths
			require.Equal(t, 1, len(paths))
			require.Equal(t, expectedPath, paths[0].Path)
			require.Equal(t, expectedServiceName, paths[0].Backend.ServiceName)
			require.Equal(t, expectedServicePort, paths[0].Backend.ServicePort.String())

		},
	}

}

func tlsRenderTest() *renderTest {

	expectedHost := "www.erewhon.io"
	expectedSecretName := "some-secret"

	tlsOptions := &helm.Options{
		SetValues: map[string]string{
			"ingress.tls[0].hosts[0]":   expectedHost,
			"ingress.tls[0].secretName": expectedSecretName,
		},
	}

	return &renderTest{
		name:    "TLS defined",
		options: tlsOptions,
		verify: func(t *testing.T, got v1beta1.Ingress) {

			tls := got.Spec.TLS
			require.Equal(t, 1, len(tls))
			require.Equal(t, expectedSecretName, tls[0].SecretName)

			hosts := tls[0].Hosts
			require.Equal(t, 1, len(hosts))
			require.Equal(t, expectedHost, hosts[0])

		},
	}

}

func annotationsRenderTest() *renderTest {

	expected := map[string]string{
		"sog\\.ingress\\.kubernetes\\.io/foo": "bar",
		"sogsimple":                           "baz",
	}

	annotationOptions := &helm.Options{
		SetValues: asAnnotationValues(expected),
	}

	return &renderTest{
		name:    "Annotations defined",
		options: annotationOptions,
		verify: func(t *testing.T, got v1beta1.Ingress) {

			found := got.Annotations

			for ek, ev := range expected {

				v, ok := found[strings.Replace(ek, "\\", "", -1)]

				if !ok {
					require.FailNow(t, fmt.Sprintf("Annotation for key %s not set", ek))
				}

				require.Equal(t, ev, v)

			}

		},
	}

}

func asAnnotationValues(from map[string]string) map[string]string {

	annotations := make(map[string]string)

	for k, v := range from {
		annotations[fmt.Sprintf("ingress.annotations.%s", k)] = v
	}

	return annotations
}
