# oauth2 proxy

This chart assembles a chain of ingress controllers and an ouath2 proxy service to protect access to a service via an oauth2 flow. See https://kubernetes.github.io/ingress-nginx/examples/auth/oauth-external-auth/ and https://akomljen.com/protect-kubernetes-external-endpoints-with-oauth2-proxy/ for more details.

As usual with helm charts the default `values.yaml` in the chart gives you an idea of how to set up.

## Layout
The structure of this directory is as follows

- `chart` The helm chart 
- `test` Tests to check the rendering of the helm chart using `terratest`

## Local development

It's a good idea to use the [terratest](https://github.com/gruntwork-io/terratest) helm library as a TDD enabler while making changes.

### Lint

Simply run `helm lint` in the chart directory to see if any errors are surfaced 

### Terratest

For this you will need go version 11 or later and helm installed

    $ cd test
    $ go test -count=1 -v  ./...
    

