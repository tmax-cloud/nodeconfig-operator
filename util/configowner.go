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

package util

import (
	"context"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConfigOwner provides a data interface for different config owner types.
type ConfigOwner struct {
	*unstructured.Unstructured
}

// IsInfrastructureReady extracts infrastructure status from the config owner.
// func (co ConfigOwner) IsInfrastructureReady() bool {
// 	infrastructureReady, _, err := unstructured.NestedBool(co.Object, "status", "infrastructureReady")
// 	if err != nil {
// 		return false
// 	}
// 	return infrastructureReady
// }

// ClusterName extracts spec.clusterName from the config owner.
// func (co ConfigOwner) ClusterName() string {
// 	clusterName, _, err := unstructured.NestedString(co.Object, "spec", "clusterName")
// 	if err != nil {
// 		return ""
// 	}
// 	return clusterName
// }

// DataSecretName extracts spec.bootstrap.dataSecretName from the config owner.
func (co ConfigOwner) DataSecretName() *string {
	dataSecretName, exist, err := unstructured.NestedString(co.Object, "spec", "bootstrap", "dataSecretName")
	if err != nil || !exist {
		return nil
	}
	return &dataSecretName
}

// IsControlPlaneMachine checks if an unstructured object is Machine with the control plane role.
// func (co ConfigOwner) IsControlPlaneMachine() bool {
// 	if co.GetKind() != "Machine" {
// 		return false
// 	}
// 	labels := co.GetLabels()
// 	if labels == nil {
// 		return false
// 	}
// 	_, ok := labels[clusterv1.MachineControlPlaneLabelName]
// 	return ok
// }

// GetConfigOwner returns the Unstructured object owning the current resource.
func GetConfigOwner(ctx context.Context, c client.Client, obj metav1.Object) (*ConfigOwner, error) {
	for _, ref := range obj.GetOwnerReferences() {
		refGV, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse GroupVersion from %q", ref.APIVersion)
		}
		refGVK := refGV.WithKind(ref.Kind)

		allowedGKs := []schema.GroupKind{
			{
				Group: "metal3.io",
				Kind:  "BareMetalHost",
			},
		}

		for _, gk := range allowedGKs {
			if refGVK.Group == gk.Group && refGVK.Kind == gk.Kind {
				return GetOwnerByRef(ctx, c, &corev1.ObjectReference{
					APIVersion: ref.APIVersion,
					Kind:       ref.Kind,
					Name:       ref.Name,
					Namespace:  obj.GetNamespace(),
				})
			}
		}
	}
	return nil, nil
}

// GetOwnerByRef finds and returns the owner by looking at the object reference.
func GetOwnerByRef(ctx context.Context, c client.Client, ref *corev1.ObjectReference) (*ConfigOwner, error) {
	obj, err := Get(ctx, c, ref, ref.Namespace)
	if err != nil {
		return nil, err
	}
	return &ConfigOwner{obj}, nil
}

// Get uses the client and reference to get an external, unstructured object.
func Get(ctx context.Context, c client.Client, ref *corev1.ObjectReference, namespace string) (*unstructured.Unstructured, error) {
	obj := new(unstructured.Unstructured)
	obj.SetAPIVersion(ref.APIVersion)
	obj.SetKind(ref.Kind)
	obj.SetName(ref.Name)
	key := client.ObjectKey{Name: obj.GetName(), Namespace: namespace}
	if err := c.Get(ctx, key, obj); err != nil {
		return nil, errors.Wrapf(err, "failed to retrieve %s external object %q/%q", obj.GetKind(), key.Namespace, key.Name)
	}
	return obj, nil
}
