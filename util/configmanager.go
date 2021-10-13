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
	"github.com/pkg/errors"
	bootstrapv1 "github.com/tmax-cloud/nodeconfig-operator/api/v1alpha1"
	"github.com/tmax-cloud/nodeconfig-operator/util/cloudinit"
	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	"sigs.k8s.io/cluster-api/util"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ConfigManager is used to build cloud-init, BMC-meta, ...
type ConfigManager struct {
	client client.Client

	NodeConfig *bootstrapv1.NodeConfig
	Log        logr.Logger
}

// NewConfigManager returns a new helper for managing a config
func (c *ConfigManager) NewConfigManager(client client.Client,
	nodeconfig *bootstrapv1.NodeConfig,
	configLog logr.Logger) (*ConfigManager, error) {

	return &ConfigManager{
		client: client,

		NodeConfig: nodeconfig,
		Log:        configLog,
	}, nil
}

// Associate associates the nodeconfig with the baremetal machine
// It's invoked by the Config Controller
func (c *ConfigManager) Associate(ctx context.Context, nConfig *bootstrapv1.NodeConfig) error {
	c.Log.Info("Associating nodeconfig", "NC.Status", c.NodeConfig.Name)
	// just... exception handling
	if c.NodeConfig == nil {
		// Should have been picked earlier. Do not requeue
		c.SetError("NodeConfig undefined")
		return nil
	}
	// clear an error if one was previously set
	c.clearError()

	// ESLEE_TODO: nodeconifg에서 Default OS IMG 설정하게 해줄것인가
	// config := c.NodeConfig.Spec
	// err := config.IsValid()
	// if err != nil {
	// 	// Should have been picked earlier. Do not requeue
	// 	m.SetError(err.Error(), capierrors.InvalidConfigurationMachineError)
	// 	return nil
	// }

	// look for associated BMH
	bmhost, _ := getHost(ctx, c.NodeConfig, c.client, c.Log)
	bmhHelper, err := patch.NewHelper(bmhost, c.client)
	if err != nil {
		c.SetError("Failed to get the BaremetalHost for the NodeConfig")
		return err
	}

	// Assign node configs(cloud init) to the BMH
	if err = c.setHostSpec(ctx, bmhost, c.NodeConfig); err != nil {
		c.SetError(err.Error())
		return err
	}
	if err = bmhHelper.Patch(ctx, bmhost); err != nil {
		c.SetError(err.Error())
		return err
	}
	c.Log.Info("Success to set host for association!", "BMH.spec", bmhost.Spec)

	// Add owner reference to the BMH
	c.NodeConfig.ObjectMeta.SetOwnerReferences(
		util.EnsureOwnerRef(c.NodeConfig.GetOwnerReferences(),
			metav1.OwnerReference{
				APIVersion: bmhost.APIVersion,
				Kind:       "BareMetalHost",
				Name:       bmhost.Name,
				UID:        bmhost.UID,
			}))

	return nil
}

// FindHost return true when it founds the associated host by looking for an annotation
// on the machine that contains a reference to the host.
func (c *ConfigManager) FindHost(ctx context.Context) (*bmh.BareMetalHost, bool) {
	if host, err := getHost(ctx, c.NodeConfig, c.client, c.Log); err != nil {
		c.Log.Error(err, "unknown error occurred at finding the BMH")
	} else if host != nil {
		provState := host.Status.Provisioning.State
		if (provState == "ready" || provState == "inspecting" ||
			provState == "registering" || provState == "match profile" ||
			provState == "available") && host.Status.OperationalStatus == "OK" {
			return host, true
		}
		return host, false
	}
	return nil, false
}

// getHost gets the associated host by looking for an annotation on the machine
// that contains a reference to the host. Returns nil if not found. Assumes the
// host is in the same namespace as the machine.
func getHost(ctx context.Context, nConfig *bootstrapv1.NodeConfig,
	cl client.Client, mLog logr.Logger) (*bmh.BareMetalHost, error) {

	// Set BMH search key
	hostNamespace, hostName := nConfig.Namespace, nConfig.Name
	host := bmh.BareMetalHost{}
	key := client.ObjectKey{
		Name:      hostName,
		Namespace: hostNamespace,
	}
	if err := cl.Get(ctx, key, &host); apierrors.IsNotFound(err) {
		mLog.Info("Can't find target the BMH CR", "host", hostName)
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &host, nil
}

// details. It will then update the host via the kube API.
func (c *ConfigManager) setHostSpec(ctx context.Context, host *bmh.BareMetalHost, nConfig *bootstrapv1.NodeConfig) error {
	c.Log.Info("setHostSpec")
	// Not provisioning while we do not have the UserData and images
	if c.NodeConfig.Spec.Image == nil || c.NodeConfig.Status.UserData == nil {
		c.Log.Error(nil, "Unknown errer...")
		return nil
	}

	host.Spec.Image = &bmh.Image{
		URL:          c.NodeConfig.Spec.Image.URL,
		Checksum:     string(c.NodeConfig.Spec.Image.Checksum),
		ChecksumType: bmh.ChecksumType(c.NodeConfig.ChecksumType()),
	}
	host.Spec.UserData = c.NodeConfig.Status.UserData

	// c.Log.Info("set BMH", "BMH.prov.state", host.Status.Provisioning.State)
	// Start to provisioning only when BMH provisioning state is 'ready'
	if host.Status.Provisioning.State == "ready" {
		host.Spec.Online = true
	}

	return nil
}

// CreateNodeInitConfig creates cloud-init
func (c *ConfigManager) CreateNodeInitConfig(ctx context.Context) (string, error) {
	c.Log.Info("Creating BootstrapData for the node")

	var cloudInitData []byte
	var cloudinitName string
	var err error
	if cloudInitData, err = cloudinit.NewNode(&cloudinit.NodeInput{
		BaseUserData: cloudinit.BaseUserData{
			AdditionalFiles:   c.NodeConfig.Spec.Files,
			NTP:               c.NodeConfig.Spec.NTP,
			CloudInitCommands: c.NodeConfig.Spec.CloudInitCommands,
			Users:             c.NodeConfig.Spec.Users,
		},
	}); err != nil {
		c.Log.Error(err, "failed to create node configuration")
		return "", err
	}

	if cloudinitName, err = c.storeBootstrapData(ctx, cloudInitData); err != nil {
		if apierrors.IsAlreadyExists(err) {
			c.Log.Info("cloudinit secret " + c.NodeConfig.Namespace + "/" + c.NodeConfig.Name + "is already created")
		}
		c.Log.Error(err, "failed to store bootstrap data")
		return "", err
	}
	return cloudinitName, nil
}

// CreateBareMetalHost creates BMH if there is not
func (c *ConfigManager) CreateBareMetalHost(ctx context.Context) error {
	c.Log.Info("Creating BareMetalHost for the node")
	if !c.NodeConfig.CheckBMHDetails() {
		c.Log.Error(nil, "Invalid BMH input")
	}
	// c.Log.Info("ESLEE: BMH info test",
	// 	"addr", c.NodeConfig.Spec.BMC.Address,
	// 	"userid", c.NodeConfig.Spec.BMC.Username,
	// 	"pw", c.NodeConfig.Spec.BMC.Password,
	// 	"img_url", c.NodeConfig.Spec.Image.URL,
	// 	"img_checksum", c.NodeConfig.Spec.Image.Checksum)

	// ESLEE: Todo - BMH config validation
	bmhost := &bmh.BareMetalHost{}
	bmhost.ObjectMeta.Name = c.NodeConfig.Name
	bmhost.ObjectMeta.Namespace = c.NodeConfig.Namespace
	bmhost.Spec.Online = false
	bmhost.Spec.BootMode = bmh.BootMode(c.NodeConfig.BootMode())
	bmhost.Spec.BMC.Address = c.NodeConfig.Spec.BMC.Address
	bmhost.Spec.BMC.CredentialsName = c.NodeConfig.Name + "-bmc-secret"
	bmhost.Spec.BMC.DisableCertificateVerification = true

	var secret *corev1.Secret
	var err error
	// Create BMH-credential (BMC info)
	if secret, err = c.storeBMHCredentials(ctx, bmhost); err != nil {
		c.Log.Error(err, "failed to store BMC credentials")
		return err
	}

	// Create BMH
	if err = c.client.Create(ctx, bmhost); err != nil {
		c.Log.Info("failed to create BareMetalHost")
		return err
	}

	// Set owner reference (the BMH owns BMC-credential)
	if err = c.setBMHCredentialsOwner(ctx, bmhost, secret); err != nil {
		c.Log.Info("failed to set BMC-credential owner")
		return err
	}

	c.Log.Info("Success to create BMH", "BMH.spec", bmhost.Spec, "BMH.status", bmhost.Status)
	return nil
}

// storeBootstrapData creates a new secret with the data passed in as input,
// sets the reference in the configuration status and ready to true.
func (c *ConfigManager) storeBootstrapData(ctx context.Context, data []byte) (string, error) {
	c.Log.Info("Store the Bootstrap data", "secret", c.NodeConfig.Status.DataSecretName)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.NodeConfig.Name,
			Namespace: c.NodeConfig.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: bootstrapv1.GroupVersion.String(),
					Kind:       "NodeConfig",
					Name:       c.NodeConfig.Name,
					UID:        c.NodeConfig.UID,
					Controller: pointer.BoolPtr(true),
				},
			},
		},
		Data: map[string][]byte{
			"value": data,
		},
	}

	if err := c.client.Create(ctx, secret); err != nil {
		return "", errors.Wrapf(err, "failed to create bootstrap data secret for NodeConfig %s/%s", c.NodeConfig.Namespace, c.NodeConfig.Name)
	}

	return secret.Name, nil
}

func (c *ConfigManager) setBMHCredentialsOwner(ctx context.Context, bmhost *bmh.
	BareMetalHost, secret *corev1.Secret) error {
	secret.OwnerReferences = []metav1.OwnerReference{
		{
			APIVersion: bmh.GroupVersion.String(),
			Kind:       "BareMetalHost",
			Name:       bmhost.Name,
			UID:        bmhost.UID,
			Controller: pointer.BoolPtr(true),
		},
	}

	if helper, err := patch.NewHelper(secret, c.client); err != nil {
		return errors.Wrapf(err, "Unknown error: fail to create helper")
	} else if err = helper.Patch(ctx, secret); err != nil {
		return errors.Wrapf(err, "Fail to patch BMH credential")
	}
	return nil
}

// storeBMHCredentials creates a new secret with the BMH data passed in as input
func (c *ConfigManager) storeBMHCredentials(ctx context.Context, bmhost *bmh.BareMetalHost) (*corev1.Secret, error) {
	c.Log.Info("Store the BMC secret", "BMC", c.NodeConfig.Spec.BMC)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.NodeConfig.Name + "-bmc-secret",
			Namespace: c.NodeConfig.Namespace,
		},
		Type: "Opaque",
		Data: map[string][]byte{
			"username": []byte(c.NodeConfig.Spec.BMC.Username),
			"password": []byte(c.NodeConfig.Spec.BMC.Password),
		},
	}

	if err := c.client.Create(ctx, secret); err != nil {
		return nil, errors.Wrapf(err, "failed to create BMC secret for BareMetalHost %s/%s", c.NodeConfig.Namespace, c.NodeConfig.Name)
	}
	return secret, nil
}

// SetError sets the ErrorMessage and ErrorReason fields on the machine and logs
// the message. It assumes the reason is invalid configuration, since that is
// currently the only relevant MachineStatusError choice.
func (c *ConfigManager) SetError(message string) {
	c.NodeConfig.Status.FailureMessage = &message
}

// clearError removes the ErrorMessage from the machine's Status if set. Returns
// nil if ErrorMessage was already nil. Returns a RequeueAfterError if the
// machine was updated.
func (c *ConfigManager) clearError() {
	if c.NodeConfig.Status.FailureMessage != nil {
		c.NodeConfig.Status.FailureMessage = nil
	}
}
