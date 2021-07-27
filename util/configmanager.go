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

	"github.com/go-logr/logr"
	bmh "github.com/metal3-io/baremetal-operator/apis/metal3.io/v1alpha1"
	nco "github.com/tmax-cloud/nodeconfig-operator/api/v1alpha1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ConfigManager struct {
	client client.Client

	NodeConfig *nco.NodeConfig
	Log        logr.Logger
}

const (
	// ProviderName is exported.
	ProviderName = "metal3"
	// HostAnnotation is the key for an annotation that should go on a Metal3Machine to
	// reference what BareMetalHost it corresponds to.
	HostAnnotation = "metal3.io/BareMetalHost"
)

// NewConfigManager returns a new helper for managing a config
func (c *ConfigManager) NewConfigManager(client client.Client,
	nodeconfig *nco.NodeConfig,
	configLog logr.Logger) (*ConfigManager, error) {

	return &ConfigManager{
		client: client,

		NodeConfig: nodeconfig,
		Log:        configLog,
	}, nil
}

// Associate associates a machine and is invoked by the Config Controller
func (c *ConfigManager) Associate(ctx context.Context) error {
	c.Log.Info("Associating nodeconfig", "nodeconfig", c.NodeConfig.Name)

	// load and validate the config
	if c.NodeConfig == nil {
		// Should have been picked earlier. Do not requeue
		return nil
	}

	// ESLEE_TODO: nodeconifg에서 OS IMG 설정하게 해줄것인가
	// config := c.NodeConfig.Spec
	// err := config.IsValid()
	// if err != nil {
	// 	// Should have been picked earlier. Do not requeue
	// 	m.SetError(err.Error(), capierrors.InvalidConfigurationMachineError)
	// 	return nil
	// }

	// clear an error if one was previously set
	c.clearError()

	// look for associated BMH
	host, err := c.getHost(ctx, c.client, c.Log)
	if err != nil {
		c.SetError("Failed to get the BaremetalHost for the Metal3Machine") //,
		// 	capierrors.CreateMachineError,
		// )
		return err
	}

	// ESLEE_TODO: nodeconfig에서 임의로 BMH 고르게 하는 기능을 줄것인가?
	// // no BMH found, trying to choose from available ones
	// if host == nil {
	// 	host, err = c.chooseHost(ctx)
	// 	if err != nil {
	// 		c.SetError("Failed to pick a BaremetalHost for the Metal3Machine")//,
	// 		return err
	// 	}
	// 	if host == nil {
	// 		c.Log.Info("No available host found. Requeuing.")
	// 		return &RequeueAfterError{RequeueAfter: requeueAfter}
	// 	}
	// 	m.Log.Info("Associating machine with host", "host", host.Name)
	// } else {
	// 	m.Log.Info("Machine already associated with host", "host", host.Name)
	// }

	// // A machine bootstrap not ready case is caught in the controller
	// // ReconcileNormal function
	// err = c.getUserData(ctx, host)
	// if err != nil {
	// 	if _, ok := err.(HasRequeueAfterError); !ok {
	// 		m.SetError("Failed to set the UserData for the Metal3Machine",
	// 			capierrors.CreateMachineError,
	// 		)
	// 	}
	// 	return err
	// }

	// err = m.setHostConsumerRef(ctx, host)
	// if err != nil {
	// 	if _, ok := err.(HasRequeueAfterError); !ok {
	// 		m.SetError("Failed to associate the BaremetalHost to the Metal3Machine",
	// 			capierrors.CreateMachineError,
	// 		)
	// 	}
	// 	return err
	// }

	// err = m.setBMCSecretLabel(ctx, host)
	// if err != nil {
	// 	if _, ok := err.(HasRequeueAfterError); !ok {
	// 		m.SetError("Failed to associate the BaremetalHost to the Metal3Machine",
	// 			capierrors.CreateMachineError,
	// 		)
	// 	}
	// 	return err
	// }

	err = c.ensureAnnotation(ctx, host)
	if err != nil {
		c.SetError("Failed to annotate the NodeConfig")
		// if _, ok := err.(HasRequeueAfterError); !ok {
		// 	m.SetError("Failed to annotate the Metal3Machine",
		// 		capierrors.CreateMachineError,
		// 	)
		// }
		return err
	}

	c.Log.Info("Finished associating machine")
	return nil
}

// GetBaremetalHostID return the provider identifier for this machine
func (c *ConfigManager) GetBaremetalHostID(ctx context.Context) (*string, error) {
	// look for associated BMH
	host, err := c.getHost(ctx, c.client, c.Log)
	if err != nil {
		c.SetError("Failed to get a BaremetalHost for the NodeConfig")
		return nil, err
	}
	if host == nil {
		c.Log.Info("BaremetalHost not associated, requeuing")
		// 	return nil, &RequeueAfterError{RequeueAfter: requeueAfter}
		return nil, err
	}
	if host.Status.Provisioning.State == bmh.StateProvisioned {
		return pointer.StringPtr(string(host.ObjectMeta.UID)), nil
	}
	c.Log.Info("Provisioning BaremetalHost, requeuing")
	// return nil, &RequeueAfterError{RequeueAfter: requeueAfter}
	return nil, nil
}

// getHost gets the associated host by looking for an annotation on the machine
// that contains a reference to the host. Returns nil if not found. Assumes the
// host is in the same namespace as the machine.

// func getHost(ctx context.Context) (*bmh.BareMetalHost, error) {
// 	host, err := getHost(ctx, m.Metal3Machine, m.client, m.Log)
// 	if err != nil || host == nil {
// 		return host, err
// 	}
// 	helper, err := patch.NewHelper(host, m.client)
// 	return host, helper, err
// }

func (c *ConfigManager) getHost(ctx context.Context, cl client.Client, mLog logr.Logger) (*bmh.BareMetalHost, error) {
	annotations := c.NodeConfig.ObjectMeta.GetAnnotations()
	if annotations == nil {
		return nil, nil
	}
	hostKey, ok := annotations[HostAnnotation]
	if !ok {
		return nil, nil
	}
	hostNamespace, hostName, err := cache.SplitMetaNamespaceKey(hostKey)
	if err != nil {
		mLog.Error(err, "Error parsing annotation value", "annotation key", hostKey)
		return nil, err
	}

	host := bmh.BareMetalHost{}
	key := client.ObjectKey{
		Name:      hostName,
		Namespace: hostNamespace,
	}
	err = cl.Get(ctx, key, &host)
	if apierrors.IsNotFound(err) {
		mLog.Info("Annotated host not found", "host", hostKey)
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &host, nil
}

// ensureAnnotation makes sure the machine has an annotation that references the
// host and uses the API to update the machine if necessary.
func (c *ConfigManager) ensureAnnotation(ctx context.Context, host *bmh.BareMetalHost) error {
	annotations := c.NodeConfig.ObjectMeta.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}
	hostKey, err := cache.MetaNamespaceKeyFunc(host)
	if err != nil {
		c.Log.Error(err, "Error parsing annotation value", "annotation key", hostKey)
		return err
	}
	existing, ok := annotations[HostAnnotation]
	if ok {
		if existing == hostKey {
			return nil
		}
		c.Log.Info("Warning: found stray annotation for host on machine. Overwriting.", "host", existing)
	}
	annotations[HostAnnotation] = hostKey
	c.NodeConfig.ObjectMeta.SetAnnotations(annotations)

	return nil
}

// HasAnnotation makes sure the nodeconfig has an annotation that references a host
func (c *ConfigManager) HasAnnotation() bool {
	annotations := c.NodeConfig.ObjectMeta.GetAnnotations()
	if annotations == nil {
		return false
	}
	_, ok := annotations[HostAnnotation]
	return ok
}

// SetError sets the ErrorMessage and ErrorReason fields on the machine and logs
// the message. It assumes the reason is invalid configuration, since that is
// currently the only relevant MachineStatusError choice.
func (c *ConfigManager) SetError(message string) {
	c.NodeConfig.Status.FailureMessage = &message
	// c.NodeConfig.Status.FailureReason = &reason
}

// clearError removes the ErrorMessage from the machine's Status if set. Returns
// nil if ErrorMessage was already nil. Returns a RequeueAfterError if the
// machine was updated.
func (c *ConfigManager) clearError() {
	if c.NodeConfig.Status.FailureMessage != nil {
		c.NodeConfig.Status.FailureMessage = nil
	}
}
