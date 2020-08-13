package test

import (
	"encoding/json"
	"errors"
	"github.com/ghodss/yaml"
	"github.com/gruntwork-io/terratest/modules/helm"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"strings"

	"path/filepath"
	"testing"
)

// Tests that the chaining of the two ingresses and oauth2 service is correct
// By chaining I mean configuration has the oauth2 flow in place for our k8s components
func TestChaining(t *testing.T) {
	t.Parallel()

	// Path to the helm chart we will test
	helmChartPath, err := filepath.Abs(chartPath)
	require.NoError(t, err)

	// Path to the helm values file we will use
	valuesPath, err := filepath.Abs("valid_setup.yaml")
	require.NoError(t, err)

	options := &helm.Options{
		ValuesFiles: []string{valuesPath},
	}

	// Render the template and verify the result
	output := helm.RenderTemplate(t, options, helmChartPath, "render-test", []string{})
	toChain(t, output).verify(t)

}

type chainsetup struct {
	proxyIngress  *v1beta1.Ingress
	publicIngress *v1beta1.Ingress
	oauthService  *corev1.Service
}

// isValid checks that the structure has been populated. Returns an error when not ready to use.
func (c chainsetup) isValid() error {
	if c.proxyIngress == nil {
		return errors.New("proxy ingress not set")
	}

	if c.publicIngress == nil {
		return errors.New("public ingress not set")
	}

	if c.oauthService == nil {
		return errors.New("oauth service not set")
	}

	return nil
}

// verify ensures that the relationship between the k8s entities is set up correctly for the oauth2 flow to work
func (c chainsetup) verify(t *testing.T) {

	c.verifyProxyToOauthService(t)
	c.verifyPublicAnnotations(t)
}

// verifyProxyToOauthService checks that the configuration between the proxy ingress and oauth2 service is correct
func (c chainsetup) verifyProxyToOauthService(t *testing.T) {

	// Should just be one path that uses the oauth2 service
	rules := c.proxyIngress.Spec.Rules
	require.Equal(t, 1, len(rules), "Should just be one rule that uses the oauth2 service")

	paths := rules[0].HTTP.Paths
	require.Equal(t, 1, len(paths), "Should just be one HTTP Path that uses the oauth2 service")

	path := paths[0]
	require.Equal(t, c.oauthService.Name, path.Backend.ServiceName, "Ingress path should use the oauth2 service name")

	ports := c.oauthService.Spec.Ports
	require.Equal(t, 1, len(ports), "Should just be one port exposed by the oauth2 service")

	port := ports[0]
	require.Equal(t, port.Port, path.Backend.ServicePort.IntVal, "Ingress path should use the oauth2 port")

	require.Equal(t, "/oauth2", path.Path)

}

// verifyPublicAnnotations checks that the annotations on the public proxy are set up so that the proxy ingress will be used for oauth2 flow.
// See https://kubernetes.github.io/ingress-nginx/examples/auth/oauth-external-auth/ for details
func (c chainsetup) verifyPublicAnnotations(t *testing.T) {

	const authUrlAnnotation = "nginx.ingress.kubernetes.io/auth-url"

	require.Contains(t, c.publicIngress.Annotations, authUrlAnnotation, "auth-url annotation is not set on the public facing ingress - see https://kubernetes.github.io/ingress-nginx/examples/auth/oauth-external-auth/")
	authUrl := c.publicIngress.Annotations[authUrlAnnotation]
	require.Equal(t, "https://$host/oauth2/auth", authUrl, "Value of the auth-url annotation should be set as per https://kubernetes.github.io/ingress-nginx/examples/auth/oauth-external-auth/")

	const authSigninAnnotation = "nginx.ingress.kubernetes.io/auth-signin"

	require.Contains(t, c.publicIngress.Annotations, authSigninAnnotation, "auth-signin annotation is not set on the public facing ingress - see https://kubernetes.github.io/ingress-nginx/examples/auth/oauth-external-auth/")
	authSignin := c.publicIngress.Annotations[authSigninAnnotation]
	require.Equal(t, "https://$host/oauth2/start?rd=$escaped_request_uri", authSignin, "Value of the auth-signin annotation should be set as per https://kubernetes.github.io/ingress-nginx/examples/auth/oauth-external-auth/")

}

// toChain takes the output from the helm render stage and tries to extract the bits we care about for testing
func toChain(t *testing.T, helmoutput string) *chainsetup {

	docs := strings.Split(helmoutput, "---")

	var c chainsetup

	for _, d := range docs {
		var m metadata

		require.NoError(t, unmarshalYamlE(d, &m), "unable to extract k8s metadata")

		switch m.Kind {

		case "Service":
			var service corev1.Service
			helm.UnmarshalK8SYaml(t, d, &service)
			c.oauthService = &service

		case "Ingress":

			var ingress v1beta1.Ingress
			helm.UnmarshalK8SYaml(t, d, &ingress)

			switch {

			case strings.HasPrefix(ingress.Name, "oauth2-proxy"):
				c.proxyIngress = &ingress

			case strings.HasPrefix(ingress.Name, "external-auth"):
				c.publicIngress = &ingress
			}

		}

	}

	require.NoError(t, c.isValid(), "unable to create chain of k8s entities")

	return &c

}

type metadata struct {
	Kind     string `json:"kind"`
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
}

func unmarshalYamlE(yamlData string, destinationObj interface{}) error {
	// NOTE: the client-go library can only decode json, so we will first convert the yaml to json before unmarshaling
	jsonData, err := yaml.YAMLToJSON([]byte(yamlData))
	if err != nil {
		return err
	}

	err = json.Unmarshal(jsonData, destinationObj)
	if err != nil {
		return err
	}
	return nil
}
