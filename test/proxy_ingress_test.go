package test

import (
	"k8s.io/api/networking/v1beta1"
	"path/filepath"
	"testing"

	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/require"
)

const proxyIngressPath = "templates/public-ingress.yaml"

// Test the rendering of the proxy ingress template
func TestRenderedroxyIngress(t *testing.T) {
	t.Parallel()

	// Path to the helm chart we will test
	helmChartPath, err := filepath.Abs(chartPath)
	require.NoError(t, err)

	testCases := []*renderTest{
		tlsRenderTest(),
		ingressRenderTest(),
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
			output := helm.RenderTemplate(subT, testCase.options, helmChartPath, "render-proxy-ingress", []string{proxyIngressPath})
			var ingress v1beta1.Ingress
			helm.UnmarshalK8SYaml(subT, output, &ingress)
			testCase.verify(subT, ingress)
		})
	}

}
