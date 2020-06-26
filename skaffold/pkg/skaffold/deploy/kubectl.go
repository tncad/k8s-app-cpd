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

package deploy

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/segmentio/textio"
	"github.com/sirupsen/logrus"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/color"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/constants"
	deploy "github.com/GoogleContainerTools/skaffold/pkg/skaffold/deploy/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/event"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubectl"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/kubernetes"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/runner/runcontext"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/util"
)

// KubectlDeployer deploys workflows using kubectl CLI.
type KubectlDeployer struct {
	*latest.KubectlDeploy

	originalImages     []build.Artifact
	workingDir         string
	kubectl            deploy.CLI
	insecureRegistries map[string]bool
}

// NewKubectlDeployer returns a new KubectlDeployer for a DeployConfig filled
// with the needed configuration for `kubectl apply`
func NewKubectlDeployer(runCtx *runcontext.RunContext) *KubectlDeployer {
	return &KubectlDeployer{
		KubectlDeploy: runCtx.Cfg.Deploy.KubectlDeploy,
		workingDir:    runCtx.WorkingDir,
		kubectl: deploy.CLI{
			CLI:         kubectl.NewFromRunContext(runCtx),
			Flags:       runCtx.Cfg.Deploy.KubectlDeploy.Flags,
			ForceDeploy: runCtx.Opts.Force,
		},
		insecureRegistries: runCtx.InsecureRegistries,
	}
}

func (k *KubectlDeployer) Labels() map[string]string {
	return map[string]string{
		constants.Labels.Deployer: "kubectl",
	}
}

// Deploy templates the provided manifests with a simple `find and replace` and
// runs `kubectl apply` on those manifests
func (k *KubectlDeployer) Deploy(ctx context.Context, out io.Writer, builds []build.Artifact, labellers []Labeller) *Result {
	event.DeployInProgress()

	manifests, err := k.renderManifests(ctx, out, builds, labellers)
	if err != nil {
		event.DeployFailed(err)
		return NewDeployErrorResult(err)
	}

	if len(manifests) == 0 {
		event.DeployComplete()
		return NewDeploySuccessResult(nil)
	}

	namespaces, err := manifests.CollectNamespaces()
	if err != nil {
		event.DeployInfoEvent(fmt.Errorf("could not fetch deployed resource namespace. "+
			"This might cause port-forward and deploy health-check to fail: %w", err))
	}

	if err := k.kubectl.Apply(ctx, textio.NewPrefixWriter(out, " - "), manifests); err != nil {
		event.DeployFailed(err)
		return NewDeployErrorResult(fmt.Errorf("kubectl error: %w", err))
	}

	event.DeployComplete()
	return NewDeploySuccessResult(namespaces)
}

func (k *KubectlDeployer) manifestFiles(manifests []string) ([]string, error) {
	var nonURLManifests []string
	for _, manifest := range manifests {
		if !util.IsURL(manifest) {
			nonURLManifests = append(nonURLManifests, manifest)
		}
	}

	list, err := util.ExpandPathsGlob(k.workingDir, nonURLManifests)
	if err != nil {
		return nil, fmt.Errorf("expanding kubectl manifest paths: %w", err)
	}

	var filteredManifests []string
	for _, f := range list {
		if !kubernetes.HasKubernetesFileExtension(f) {
			if !util.StrSliceContains(manifests, f) {
				logrus.Infof("refusing to deploy/delete non {json, yaml} file %s", f)
				logrus.Info("If you still wish to deploy this file, please specify it directly, outside a glob pattern.")
				continue
			}
		}
		filteredManifests = append(filteredManifests, f)
	}

	return filteredManifests, nil
}

// readManifests reads the manifests to deploy/delete.
func (k *KubectlDeployer) readManifests(ctx context.Context) (deploy.ManifestList, error) {
	// Get file manifests
	manifests, err := k.Dependencies()
	if err != nil {
		return nil, fmt.Errorf("listing manifests: %w", err)
	}

	// Append URL manifests
	for _, manifest := range k.KubectlDeploy.Manifests {
		if util.IsURL(manifest) {
			manifests = append(manifests, manifest)
		}
	}

	if len(manifests) == 0 {
		return deploy.ManifestList{}, nil
	}

	return k.kubectl.ReadManifests(ctx, manifests)
}

// readRemoteManifests will try to read manifests from the given kubernetes
// context in the specified namespace and for the specified type
func (k *KubectlDeployer) readRemoteManifest(ctx context.Context, name string) ([]byte, error) {
	var args []string
	ns := ""
	if parts := strings.Split(name, ":"); len(parts) > 1 {
		ns = parts[0]
		name = parts[1]
	}
	args = append(args, name, "-o", "yaml")

	var manifest bytes.Buffer
	err := k.kubectl.RunInNamespace(ctx, nil, &manifest, "get", ns, args...)
	if err != nil {
		return nil, fmt.Errorf("getting manifest: %w", err)
	}

	return manifest.Bytes(), nil
}

func (k *KubectlDeployer) Render(ctx context.Context, out io.Writer, builds []build.Artifact, labellers []Labeller, filepath string) error {
	manifests, err := k.renderManifests(ctx, out, builds, labellers)
	if err != nil {
		return err
	}

	return outputRenderedManifests(manifests.String(), filepath, out)
}

func (k *KubectlDeployer) renderManifests(ctx context.Context, out io.Writer, builds []build.Artifact, labellers []Labeller) (deploy.ManifestList, error) {
	if err := k.kubectl.CheckVersion(ctx); err != nil {
		color.Default.Fprintln(out, "kubectl client version:", k.kubectl.Version(ctx))
		color.Default.Fprintln(out, err)
	}

	manifests, err := k.readManifests(ctx)
	if err != nil {
		return nil, fmt.Errorf("reading manifests: %w", err)
	}

	for _, m := range k.RemoteManifests {
		manifest, err := k.readRemoteManifest(ctx, m)
		if err != nil {
			return nil, fmt.Errorf("get remote manifests: %w", err)
		}

		manifests = append(manifests, manifest)
	}

	if len(k.originalImages) == 0 {
		k.originalImages, err = manifests.GetImages()
		if err != nil {
			return nil, fmt.Errorf("get images from manifests: %w", err)
		}
	}

	if len(manifests) == 0 {
		return nil, nil
	}

	manifests, err = manifests.ReplaceImages(builds)
	if err != nil {
		return nil, fmt.Errorf("replacing images in manifests: %w", err)
	}

	for _, transform := range manifestTransforms {
		manifests, err = transform(manifests, builds, k.insecureRegistries)
		if err != nil {
			return nil, fmt.Errorf("unable to transform manifests: %w", err)
		}
	}

	manifests, err = manifests.SetLabels(merge(k, labellers...))
	if err != nil {
		return nil, fmt.Errorf("setting labels in manifests: %w", err)
	}

	return manifests, nil
}

// Cleanup deletes what was deployed by calling Deploy.
func (k *KubectlDeployer) Cleanup(ctx context.Context, out io.Writer) error {
	manifests, err := k.readManifests(ctx)
	if err != nil {
		return fmt.Errorf("reading manifests: %w", err)
	}

	// revert remote manifests
	// TODO(dgageot): That seems super dangerous and I don't understand
	// why we need to update resources just before we delete them.
	if len(k.RemoteManifests) > 0 {
		var rm deploy.ManifestList
		for _, m := range k.RemoteManifests {
			manifest, err := k.readRemoteManifest(ctx, m)
			if err != nil {
				return fmt.Errorf("get remote manifests: %w", err)
			}
			rm = append(rm, manifest)
		}

		upd, err := rm.ReplaceImages(k.originalImages)
		if err != nil {
			return fmt.Errorf("replacing with originals: %w", err)
		}

		if err := k.kubectl.Apply(ctx, out, upd); err != nil {
			return fmt.Errorf("apply original: %w", err)
		}
	}

	if err := k.kubectl.Delete(ctx, textio.NewPrefixWriter(out, " - "), manifests); err != nil {
		return fmt.Errorf("delete: %w", err)
	}

	return nil
}

// Dependencies lists all the files that describe what needs to be deployed.
func (k *KubectlDeployer) Dependencies() ([]string, error) {
	return k.manifestFiles(k.KubectlDeploy.Manifests)
}
