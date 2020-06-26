/*
Copyright 2020 The Skaffold Authors

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

package debug

import (
	"encoding/json"
	"testing"

	cnb "github.com/buildpacks/lifecycle"
	"github.com/buildpacks/lifecycle/launch"
	v1 "k8s.io/api/core/v1"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestUpdateForCNBImage(t *testing.T) {
	// metadata with default process type
	md := cnb.BuildMetadata{Processes: []launch.Process{
		{Type: "web", Command: "webProcess", Args: []string{"webArg1", "webArg2"}},
		{Type: "diag", Command: "diagProcess"},
		{Type: "direct", Command: "command", Args: []string{"cmdArg1"}, Direct: true},
	}}
	mdMarshalled, _ := json.Marshal(&md)
	mdJSON := string(mdMarshalled)
	// metadata with no default process type
	mdnd := cnb.BuildMetadata{Processes: []launch.Process{
		{Type: "diag", Command: "diagProcess"},
		{Type: "direct", Command: "command", Args: []string{"cmdArg1"}, Direct: true},
	}}
	mdndMarshalled, _ := json.Marshal(&mdnd)
	mdndJSON := string(mdndMarshalled)

	tests := []struct {
		description string
		input       imageConfiguration
		shouldErr   bool
		expected    v1.Container
	}{
		{
			description: "error when missing build.metadata",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}},
			shouldErr:   true,
		},
		{
			description: "error when build.metadata missing processes",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, labels: map[string]string{"io.buildpacks.build.metadata": "{}"}},
			shouldErr:   true,
		},
		{
			description: "direct command-lines are rewritten as direct command-lines",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, arguments: []string{"--", "web", "arg1", "arg2"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"--", "web", "arg1", "arg2"}},
		},
		{
			description: "defaults to web process when no process type",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"webProcess", "webArg1", "webArg2"}},
		},
		{
			description: "resolves to default 'web' process",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"webProcess", "webArg1", "webArg2"}},
		},
		{
			description: "CNB_PROCESS_TYPE=web",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, env: map[string]string{"CNB_PROCESS_TYPE": "web"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"webProcess", "webArg1", "webArg2"}},
		},
		{
			description: "CNB_PROCESS_TYPE=diag",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, env: map[string]string{"CNB_PROCESS_TYPE": "diag"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"diagProcess"}},
		},
		{
			description: "CNB_PROCESS_TYPE=direct",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, env: map[string]string{"CNB_PROCESS_TYPE": "direct"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"--", "command", "cmdArg1"}},
		},
		{
			description: "script command-line",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, arguments: []string{"python main.py"}, labels: map[string]string{"io.buildpacks.build.metadata": mdJSON}},
			shouldErr:   false,
			expected:    v1.Container{Args: []string{"python main.py"}},
		},
		{
			description: "no process and no args",
			input:       imageConfiguration{entrypoint: []string{"/cnb/lifecycle/launcher"}, labels: map[string]string{"io.buildpacks.build.metadata": mdndJSON}},
			shouldErr:   false,
			expected:    v1.Container{},
		},
	}
	for _, test := range tests {
		// Test that when a transform modifies the command-line arguments, then
		// the changes are reflected to the launcher command-line
		testutil.Run(t, test.description+" (args changed)", func(t *testutil.T) {
			argsChangedTransform := func(c *v1.Container, ic imageConfiguration) (ContainerDebugConfiguration, string, error) {
				c.Args = ic.arguments
				return ContainerDebugConfiguration{}, "", nil
			}
			copy := v1.Container{}
			_, _, err := updateForCNBImage(&copy, test.input, argsChangedTransform)
			t.CheckErrorAndDeepEqual(test.shouldErr, err, test.expected, copy)
		})

		// Test that when the arguments are left unchanged, that the container is unchanged
		testutil.Run(t, test.description+" (args unchanged)", func(t *testutil.T) {
			argsUnchangedTransform := func(c *v1.Container, ic imageConfiguration) (ContainerDebugConfiguration, string, error) {
				return ContainerDebugConfiguration{}, "", nil
			}

			copy := v1.Container{}
			_, _, err := updateForCNBImage(&copy, test.input, argsUnchangedTransform)
			t.CheckError(test.shouldErr, err)
			if copy.Args != nil {
				t.Errorf("args not nil: %v", copy.Args)
			}
		})
	}
}
