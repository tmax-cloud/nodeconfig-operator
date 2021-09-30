/*
Copyright 2021.

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

package v1alpha1

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func errorContains(out error, want string) bool {
	if out == nil {
		return want == ""
	}
	if want == "" {
		return false
	}
	return strings.Contains(out.Error(), want)
}

func TestNodeConfigCreate(t *testing.T) {
	tests := []struct {
		name      string
		nc        *NodeConfig
		wantedErr string
	}{
		{
			name: "valid",
			nc: &NodeConfig{TypeMeta: metav1.TypeMeta{
				Kind:       "NodeConfig",
				APIVersion: "bootstrap.tmax.io/v1alpha1",
			}, ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: "test-namespace",
			}, Spec: NodeConfigSpec{
				BMC: &BMC{
					Address:  "192.168.111.204",
					Username: "USERID",
					Password: "PASSW0RD",
				},
				Image: &Image{
					URL:      "http://192.168.111.1:6180/images/CENTOS_8.2_NODE_IMAGE_K8S_v1.20.2.qcow2",
					Checksum: "http://192.168.111.1:6180/images/CENTOS_8.2_NODE_IMAGE_K8S_v1.20.2.qcow2.md5sum",
				},
			}},
			wantedErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// t.Log("tt.nc", tt.nc)
			if err := tt.nc.ValidateCreate(); !errorContains(err, tt.wantedErr) {
				t.Errorf("NodeConfig.ValidateCreate() error = %v, wantErr %v", err, tt.wantedErr)
			}
		})
	}
}

// func newSecret(name string, data map[string]string) *corev1.Secret {
// 	secretData := make(map[string][]byte)
// 	for k, v := range data {
// 		secretData[k] = []byte(base64.StdEncoding.EncodeToString([]byte(v)))
// 	}

// 	secret := &corev1.Secret{
// 		TypeMeta: metav1.TypeMeta{
// 			Kind:       "Secret",
// 			APIVersion: "v1",
// 		},
// 		ObjectMeta: metav1.ObjectMeta{
// 			Name:      name,
// 			Namespace: namespace,
// 		},
// 		Data: secretData,
// 	}

// 	return secret
// }
