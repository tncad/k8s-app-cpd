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

package validation

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/docker"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/schema/latest"
	"github.com/GoogleContainerTools/skaffold/pkg/skaffold/yamltags"
)

var (
	// for testing
	validateYamltags = yamltags.ValidateStruct
)

// Process checks if the Skaffold pipeline is valid and returns all encountered errors as a concatenated string
func Process(config *latest.SkaffoldConfig) error {
	errs := visitStructs(config, validateYamltags)
	errs = append(errs, validateImageNames(config.Build.Artifacts)...)
	errs = append(errs, validateDockerNetworkMode(config.Build.Artifacts)...)
	errs = append(errs, validateCustomDependencies(config.Build.Artifacts)...)
	errs = append(errs, validateSyncRules(config.Build.Artifacts)...)
	errs = append(errs, validatePortForwardResources(config.PortForward)...)
	errs = append(errs, validateJibPluginTypes(config.Build.Artifacts)...)

	if len(errs) == 0 {
		return nil
	}

	var messages []string
	for _, err := range errs {
		messages = append(messages, err.Error())
	}
	return fmt.Errorf(strings.Join(messages, " | "))
}

// validateImageNames makes sure the artifact image names are valid base names,
// without tags nor digests.
func validateImageNames(artifacts []*latest.Artifact) (errs []error) {
	for _, a := range artifacts {
		parsed, err := docker.ParseReference(a.ImageName)
		if err != nil {
			errs = append(errs, fmt.Errorf("invalid imageName '%s': %v", a.ImageName, err))
			continue
		}

		if parsed.Tag != "" {
			errs = append(errs, fmt.Errorf("invalid imageName '%s': no tag should be specified. Use taggers instead: https://skaffold.dev/docs/how-tos/taggers/", a.ImageName))
		}

		if parsed.Digest != "" {
			errs = append(errs, fmt.Errorf("invalid imageName '%s': no digest should be specified. Use taggers instead: https://skaffold.dev/docs/how-tos/taggers/", a.ImageName))
		}
	}
	return
}

// validateDockerNetworkMode makes sure that networkMode is one of `bridge`, `none`, or `host` if set.
func validateDockerNetworkMode(artifacts []*latest.Artifact) (errs []error) {
	for _, a := range artifacts {
		if a.DockerArtifact == nil || a.DockerArtifact.NetworkMode == "" {
			continue
		}
		mode := strings.ToLower(a.DockerArtifact.NetworkMode)
		if mode == "none" || mode == "bridge" || mode == "host" {
			continue
		}
		errs = append(errs, fmt.Errorf("artifact %s has invalid networkMode '%s'", a.ImageName, mode))
	}
	return
}

// validateCustomDependencies makes sure that dependencies.ignore is only used in conjunction with dependencies.paths
func validateCustomDependencies(artifacts []*latest.Artifact) (errs []error) {
	for _, a := range artifacts {
		if a.CustomArtifact == nil || a.CustomArtifact.Dependencies == nil || a.CustomArtifact.Dependencies.Ignore == nil {
			continue
		}

		if a.CustomArtifact.Dependencies.Dockerfile != nil || a.CustomArtifact.Dependencies.Command != "" {
			errs = append(errs, fmt.Errorf("artifact %s has invalid dependencies; dependencies.ignore can only be used in conjunction with dependencies.paths", a.ImageName))
		}
	}
	return
}

// visitStructs recursively visits all fields in the config and collects errors found by the visitor
func visitStructs(s interface{}, visitor func(interface{}) error) []error {
	v := reflect.ValueOf(s)
	t := reflect.TypeOf(s)

	switch v.Kind() {
	case reflect.Struct:
		var errs []error
		if err := visitor(v.Interface()); err != nil {
			errs = append(errs, err)
		}

		// also check all fields of the current struct
		for i := 0; i < t.NumField(); i++ {
			if !v.Field(i).CanInterface() {
				continue
			}
			if fieldErrs := visitStructs(v.Field(i).Interface(), visitor); fieldErrs != nil {
				errs = append(errs, fieldErrs...)
			}
		}

		return errs

	case reflect.Slice:
		// for slices check each element
		var errs []error
		for i := 0; i < v.Len(); i++ {
			if elemErrs := visitStructs(v.Index(i).Interface(), visitor); elemErrs != nil {
				errs = append(errs, elemErrs...)
			}
		}
		return errs

	case reflect.Ptr:
		// for pointers check the referenced value
		if v.IsNil() {
			return nil
		}
		return visitStructs(v.Elem().Interface(), visitor)

	default:
		// other values are fine
		return nil
	}
}

// validateSyncRules checks that all manual sync rules have a valid strip prefix
func validateSyncRules(artifacts []*latest.Artifact) []error {
	var errs []error
	for _, a := range artifacts {
		if a.Sync != nil {
			for _, r := range a.Sync.Manual {
				if !strings.HasPrefix(r.Src, r.Strip) {
					err := fmt.Errorf("sync rule pattern '%s' does not have prefix '%s'", r.Src, r.Strip)
					errs = append(errs, err)
				}
			}
		}
	}
	return errs
}

// validatePortForwardResources checks that all user defined port forward resources
// have a valid resourceType
func validatePortForwardResources(pfrs []*latest.PortForwardResource) []error {
	var errs []error
	validResourceTypes := map[string]struct{}{
		"pod":                   {},
		"deployment":            {},
		"service":               {},
		"replicaset":            {},
		"replicationcontroller": {},
		"statefulset":           {},
		"daemonset":             {},
		"cronjob":               {},
		"job":                   {},
	}
	for _, pfr := range pfrs {
		resourceType := strings.ToLower(string(pfr.Type))
		if _, ok := validResourceTypes[resourceType]; !ok {
			errs = append(errs, fmt.Errorf("%s is not a valid resource type for port forwarding", pfr.Type))
		}
	}
	return errs
}

// validateJibPluginTypes makes sure that jib type is one of `maven`, or `gradle` if set.
func validateJibPluginTypes(artifacts []*latest.Artifact) (errs []error) {
	for _, a := range artifacts {
		if a.JibArtifact == nil || a.JibArtifact.Type == "" {
			continue
		}
		t := strings.ToLower(a.JibArtifact.Type)
		if t == "maven" || t == "gradle" {
			continue
		}
		errs = append(errs, fmt.Errorf("artifact %s has invalid Jib plugin type '%s'", a.ImageName, t))
	}
	return
}
