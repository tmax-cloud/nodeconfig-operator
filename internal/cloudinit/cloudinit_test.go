/*
Copyright 2019 The Kubernetes Authors.

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

package cloudinit

import (
	"testing"

	. "github.com/onsi/gomega"

	infrav1 "github.com/tmax-cloud/nodeconfig-operator/api/v1alpha1"
)

func TestNewNodeAdditionalFileEncodings(t *testing.T) {
	g := NewWithT(t)

	nodeinput := &NodeInput{
		BaseUserData: BaseUserData{
			Header:            "test",
			CloudInitCommands: nil,
			AdditionalFiles: []infrav1.File{
				{
					Path:     "/tmp/my-path",
					Encoding: infrav1.Base64,
					Content:  "aGk=",
				},
				{
					Path:    "/tmp/my-other-path",
					Content: "hi",
				},
			},
			WriteFiles: nil,
			Users:      nil,
			NTP:        nil,
		},
	}

	out, err := NewNode(nodeinput)
	g.Expect(err).NotTo(HaveOccurred())

	expectedFiles := []string{
		`-   path: /tmp/my-path
    encoding: "base64"
    content: |
      aGk=`,
		`-   path: /tmp/my-other-path
    content: |
      hi`,
	}
	for _, f := range expectedFiles {
		g.Expect(out).To(ContainSubstring(f))
	}
}

func TestNewNodeCommands(t *testing.T) {
	g := NewWithT(t)

	nodeinput := &NodeInput{
		BaseUserData: BaseUserData{
			Header:            "test",
			CloudInitCommands: []string{`"echo $(date) ': hello world!'"`},
			AdditionalFiles:   nil,
			WriteFiles:        nil,
			Users:             nil,
			NTP:               nil,
		},
	}

	out, err := NewNode(nodeinput)
	g.Expect(err).NotTo(HaveOccurred())

	expectedCommands := []string{
		`"\"echo $(date) ': hello world!'\""`,
	}
	for _, f := range expectedCommands {
		g.Expect(out).To(ContainSubstring(f))
	}
}
