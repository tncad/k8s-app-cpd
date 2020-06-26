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

package tag

import (
	"testing"

	"github.com/GoogleContainerTools/skaffold/testutil"
)

func TestGenerateFullyQualifiedImageName(t *testing.T) {
	c := &ChecksumTagger{}

	tag, err := c.GenerateFullyQualifiedImageName(".", "img:tag")
	testutil.CheckErrorAndDeepEqual(t, false, err, "img:tag", tag)

	tag, err = c.GenerateFullyQualifiedImageName(".", "img")
	testutil.CheckErrorAndDeepEqual(t, false, err, "img:latest", tag)

	tag, err = c.GenerateFullyQualifiedImageName(".", "registry.example.com:8080/img:tag")
	testutil.CheckErrorAndDeepEqual(t, false, err, "registry.example.com:8080/img:tag", tag)

	tag, err = c.GenerateFullyQualifiedImageName(".", "registry.example.com:8080/img")
	testutil.CheckErrorAndDeepEqual(t, false, err, "registry.example.com:8080/img:latest", tag)

	tag, err = c.GenerateFullyQualifiedImageName(".", "registry.example.com/img")
	testutil.CheckErrorAndDeepEqual(t, false, err, "registry.example.com/img:latest", tag)

	tag, err = c.GenerateFullyQualifiedImageName(".", "registry.example.com:8080:garbage")
	testutil.CheckErrorAndDeepEqual(t, true, err, "", tag)
}
