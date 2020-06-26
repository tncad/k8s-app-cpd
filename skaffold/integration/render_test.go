/*
Copyright 2019 The Skaffold Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package integration

import (
	"bytes"
	"context"
	"testing"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestKubectlRender(t *testing.T) {
	MarkIntegrationTest(t, CanRunWithoutGcp)

	tests := []struct {
		description string
		builds      []build.Artifact
		labels      []deploy.Labeller
		input       string
		expectedOut string
	}{
		{
			description: "normal render",
			builds: []build.Artifact{
				{
					ImageName: "gcr.io/k8s-skaffold/skaffold",
					Tag:       "gcr.io/k8s-skaffold/skaffold:test",
				},
			},
			labels: []deploy.Labeller{},
			input: `apiVersion: v1
kind: Pod
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold
    name: skaffold
`,
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  labels:
    skaffold.dev/deployer: kubectl
  namespace: default
spec:
  containers:
  - image: gcr.io/k8s-skaffold/skaffold:test
    name: skaffold
`,
		},
		{
			description: "two artifacts",
			builds: []build.Artifact{
				{
					ImageName: "gcr.io/project/image1",
					Tag:       "gcr.io/project/image1:tag1",
				},
				{
					ImageName: "gcr.io/project/image2",
					Tag:       "gcr.io/project/image2:tag2",
				},
			},
			labels: []deploy.Labeller{},
			input: `apiVersion: v1
kind: Pod
spec:
  containers:
  - image: gcr.io/project/image1
    name: image1
  - image: gcr.io/project/image2
    name: image2
`,
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  labels:
    skaffold.dev/deployer: kubectl
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
  - image: gcr.io/project/image2:tag2
    name: image2
`,
		},
		{
			description: "two artifacts, combined manifests",
			builds: []build.Artifact{
				{
					ImageName: "gcr.io/project/image1",
					Tag:       "gcr.io/project/image1:tag1",
				},
				{
					ImageName: "gcr.io/project/image2",
					Tag:       "gcr.io/project/image2:tag2",
				},
			},
			input: `apiVersion: v1
kind: Pod
spec:
  containers:
  - image: gcr.io/project/image1
    name: image1
---
apiVersion: v1
kind: Pod
spec:
  containers:
  - image: gcr.io/project/image2
    name: image2
`,
			expectedOut: `apiVersion: v1
kind: Pod
metadata:
  labels:
    skaffold.dev/deployer: kubectl
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image1:tag1
    name: image1
---
apiVersion: v1
kind: Pod
metadata:
  labels:
    skaffold.dev/deployer: kubectl
  namespace: default
spec:
  containers:
  - image: gcr.io/project/image2:tag2
    name: image2
`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			t.NewTempDir().
				Write("deployment.yaml", test.input).
				Chdir()

			deployer := deploy.NewKubectlDeployer(&runcontext.RunContext{
				WorkingDir: ".",
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							KubectlDeploy: &latest.KubectlDeploy{
								Manifests: []string{"deployment.yaml"},
							},
						},
					},
				},
			})
			var b bytes.Buffer
			err := deployer.Render(context.Background(), &b, test.builds, test.labels, "")

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedOut, b.String())
		})
	}
}

func TestHelmRender(t *testing.T) {
	if testing.Short() || RunOnGCP() {
		t.Skip("skipping kind integration test")
	}

	tests := []struct {
		description  string
		builds       []build.Artifact
		labels       []deploy.Labeller
		helmReleases []latest.HelmRelease
		expectedOut  string
	}{
		{
			description: "Bare bones render",
			builds: []build.Artifact{
				{
					ImageName: "gke-loadbalancer",
					Tag:       "gke-loadbalancer:test",
				},
			},
			labels: []deploy.Labeller{},
			helmReleases: []latest.HelmRelease{{
				Name:      "gke_loadbalancer",
				ChartPath: "testdata/gke_loadbalancer/loadbalancer-helm",
				ArtifactOverrides: map[string]string{
					"image": "gke-loadbalancer",
				},
			}},
			expectedOut: `---
# Source: loadbalancer-helm/templates/k8s.yaml
apiVersion: v1
kind: Service
metadata:
  name: gke-loadbalancer
  labels:
    app: gke-loadbalancer
spec:
  type: LoadBalancer
  ports:
    - port: 80
      targetPort: 3000
      protocol: TCP
      name: http
  selector:
    app: "gke-loadbalancer"
---
# Source: loadbalancer-helm/templates/k8s.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: gke-loadbalancer
  labels:
    app: gke-loadbalancer
spec:
  replicas: 1
  selector:
    matchLabels:
      app: gke-loadbalancer
  template:
    metadata:
      labels:
        app: gke-loadbalancer
    spec:
      containers:
        - name: gke-container
          image: gke-loadbalancer:test
          ports:
            - containerPort: 3000

`,
		},
		{
			description: "A more complex template",
			builds: []build.Artifact{
				{
					ImageName: "gcr.io/k8s-skaffold/skaffold-helm",
					Tag:       "gcr.io/k8s-skaffold/skaffold-helm:sha256-nonsenslettersandnumbers",
				},
			},
			labels: []deploy.Labeller{},
			helmReleases: []latest.HelmRelease{{
				Name:      "skaffold-helm",
				ChartPath: "testdata/helm/skaffold-helm",
				ArtifactOverrides: map[string]string{
					"image": "gcr.io/k8s-skaffold/skaffold-helm",
				},
			}},
			expectedOut: `---
# Source: skaffold-helm/templates/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: skaffold-helm-skaffold-helm
  labels:
    app: skaffold-helm
    chart: skaffold-helm-0.1.0
    release: skaffold-helm
    heritage: Helm
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: 80
      protocol: TCP
      name: nginx
  selector:
    app: skaffold-helm
    release: skaffold-helm
---
# Source: skaffold-helm/templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: skaffold-helm
  labels:
    app: skaffold-helm
    chart: skaffold-helm-0.1.0
    release: skaffold-helm
    heritage: Helm
spec:
  replicas: 1
  selector:
    matchLabels:
      app: skaffold-helm
      release: skaffold-helm
  template:
    metadata:
      labels:
        app: skaffold-helm
        release: skaffold-helm
    spec:
      containers:
        - name: skaffold-helm
          image: gcr.io/k8s-skaffold/skaffold-helm:sha256-nonsenslettersandnumbers
          imagePullPolicy: 
          ports:
            - containerPort: 80
          resources:
            {}
---
# Source: skaffold-helm/templates/ingress.yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: skaffold-helm-skaffold-helm
  labels:
    app: skaffold-helm
    chart: skaffold-helm-0.1.0
    release: skaffold-helm
    heritage: Helm
  annotations:
spec:
  rules:
    - http:
        paths:
          - path: /
            backend:
              serviceName: skaffold-helm-skaffold-helm
              servicePort: 80

`,
		},
	}
	for _, test := range tests {
		testutil.Run(t, test.description, func(t *testutil.T) {
			deployer := deploy.NewHelmDeployer(&runcontext.RunContext{
				Cfg: latest.Pipeline{
					Deploy: latest.DeployConfig{
						DeployType: latest.DeployType{
							HelmDeploy: &latest.HelmDeploy{
								Releases: test.helmReleases,
							},
						},
					},
				},
			})
			var b bytes.Buffer
			err := deployer.Render(context.Background(), &b, test.builds, test.labels, "")

			t.CheckNoError(err)
			t.CheckDeepEqual(test.expectedOut, b.String())
		})
	}
}
