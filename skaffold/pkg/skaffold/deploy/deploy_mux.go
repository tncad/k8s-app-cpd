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
	"io"
	"sort"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/build"
)

// DeployerMux forwards all method calls to the deployers it contains.
// When encountering an error, it aborts and returns the error. Otherwise,
// it collects the results and returns it in bulk.
type DeployerMux []Deployer

type unit struct{}

// stringSet helps to de-duplicate a set of strings.
type stringSet map[string]unit

func newStringSet() stringSet {
	return make(map[string]unit)
}

// insert adds strings to the set.
func (s stringSet) insert(strings ...string) {
	for _, item := range strings {
		s[item] = unit{}
	}
}

// toList returns the sorted list of inserted strings.
func (s stringSet) toList() []string {
	var res []string
	for item := range s {
		res = append(res, item)
	}
	sort.Strings(res)
	return res
}

func (m DeployerMux) Labels() map[string]string {
	labels := make(map[string]string)
	for _, deployer := range m {
		copyMap(labels, deployer.Labels())
	}
	return labels
}

func (m DeployerMux) Deploy(ctx context.Context, w io.Writer, as []build.Artifact, ls []Labeller) *Result {
	seenNamespaces := newStringSet()
	for _, deployer := range m {
		result := deployer.Deploy(ctx, w, as, ls)
		if result.err != nil {
			return result
		}
		seenNamespaces.insert(result.Namespaces()...)
	}
	return NewDeploySuccessResult(seenNamespaces.toList())
}

func (m DeployerMux) Dependencies() ([]string, error) {
	deps := newStringSet()
	for _, deployer := range m {
		result, err := deployer.Dependencies()
		if err != nil {
			return nil, err
		}
		deps.insert(result...)
	}
	return deps.toList(), nil
}

func (m DeployerMux) Cleanup(ctx context.Context, w io.Writer) error {
	for _, deployer := range m {
		if err := deployer.Cleanup(ctx, w); err != nil {
			return err
		}
	}
	return nil
}

func (m DeployerMux) Render(ctx context.Context, w io.Writer, as []build.Artifact, ls []Labeller, filepath string) error {
	resources, buf := []string{}, &bytes.Buffer{}
	for _, deployer := range m {
		buf.Reset()
		if err := deployer.Render(ctx, buf, as, ls, "" /* never write to files */); err != nil {
			return err
		}
		resources = append(resources, buf.String())
	}

	allResources := strings.Join(resources, "\n---\n")
	return outputRenderedManifests(allResources, filepath, w)
}
