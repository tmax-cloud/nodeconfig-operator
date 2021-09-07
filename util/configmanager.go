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

type ConfigManager struct {
	client client.Client

	NodeConfig *bootstrapv1.NodeConfig
	Log        logr.Logger
}

const (
// ProviderName is exported.
// ProviderName = "metal3"
// HostAnnotation is the key for an annotation that should go on a Metal3Machine to
// reference what BareMetalHost it corresponds to.
// HostAnnotation = "metal3.io/BareMetalHost"
)

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
func (c *ConfigManager) Associate(ctx context.Context) error {
	c.Log.Info("Associating nodeconfig", "nodeconfig", c.NodeConfig.Name)

	// load and validate the config
	if c.NodeConfig == nil {
		// Should have been picked earlier. Do not requeue
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
	bmhost, bmhHelper, err := c.getHost(ctx)
	if err != nil {
		c.SetError("Failed to get the BaremetalHost for the NodeConfig")
		return err
	}
	c.Log.Info("Success to get host for association!")

	// Assign node configs(cloud init) to the BMH
	if err = c.setHostSpec(ctx, bmhost); err != nil {
		c.SetError(err.Error())
	}
	c.Log.Info("Success to set host (Image, userData) for association!")

	// Add owner reference to the BMH
	c.NodeConfig.ObjectMeta.SetOwnerReferences(
		util.EnsureOwnerRef(c.NodeConfig.GetOwnerReferences(),
			metav1.OwnerReference{
				APIVersion: bmhost.APIVersion,
				Kind:       "BareMetalHost",
				Name:       bmhost.Name,
				UID:        bmhost.UID,
			}))

	err = bmhHelper.Patch(ctx, bmhost)
	if err != nil {
		return err
	}

	c.Log.Info("Finished associating machine", "BMH.status", bmhost.Status)
	return nil
}

// findHost return true when it founds the associated host by looking for an annotation
// on the machine that contains a reference to the host.
func (c *ConfigManager) FindHost(ctx context.Context) (*bmh.BareMetalHost, bool) {
	// ESLEE: todo - error 발생했으면 status에 찍을 것인가
	if host, err := getHost(ctx, c.NodeConfig, c.client, c.Log); err != nil {
		c.Log.Error(err, "ESLEE_tmp: unexpected err")
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
func (c *ConfigManager) getHost(ctx context.Context) (*bmh.BareMetalHost, *patch.Helper, error) {
	host, err := getHost(ctx, c.NodeConfig, c.client, c.Log)
	if err != nil || host == nil {
		return host, nil, err
	}
	helper, err := patch.NewHelper(host, c.client)
	return host, helper, err
}

func getHost(ctx context.Context, nConfig *bootstrapv1.NodeConfig,
	cl client.Client, mLog logr.Logger) (*bmh.BareMetalHost, error) {
	// mLog.Info("ESLEE: Start to find host")
	// annotations := nConfig.ObjectMeta.GetAnnotations()
	// if annotations == nil {
	// 	return nil, nil
	// }
	// hostKey, ok := annotations[HostAnnotation]
	// if !ok {
	// 	// mLog.Info("ESLEE: no metal3/baremetalhost annotation")
	// 	return nil, nil
	// }
	// hostNamespace, hostName, err := cache.SplitMetaNamespaceKey(hostKey)
	// if err != nil {
	// 	mLog.Error(err, "Error parsing annotation value", "annotation key", hostKey)
	// 	return nil, err
	// }
	hostNamespace, hostName := nConfig.Namespace, nConfig.Name

	// mLog.Info("ESLEE_TMP: Where's my BMH")
	host := bmh.BareMetalHost{}
	key := client.ObjectKey{
		Name:      hostName,
		Namespace: hostNamespace,
	}
	err := cl.Get(ctx, key, &host)
	if apierrors.IsNotFound(err) {
		mLog.Info("Can't find target BareMetalHost CR", "host", hostName)
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return &host, err
}

// details. It will then update the host via the kube API.
func (c *ConfigManager) setHostSpec(ctx context.Context, host *bmh.BareMetalHost) error {
	// Not provisioning while we do not have the UserData and images
	if host.Spec.Image == nil && c.NodeConfig.Status.UserData != nil {
		host.Spec.Image = &bmh.Image{
			URL:          c.NodeConfig.Spec.Image.URL,
			Checksum:     string(c.NodeConfig.Spec.Image.Checksum),
			ChecksumType: bmh.ChecksumType(c.NodeConfig.ChecksumType()),
		}
	}
	if c.NodeConfig.Status.UserData != nil {
		host.Spec.UserData = c.NodeConfig.Status.UserData
		if host.Spec.UserData != nil && host.Spec.UserData.Namespace == "" {
			host.Spec.UserData.Namespace = host.Namespace
		}
		// c.Log.Info("ESLEE_TMP: set BMH", "userdata", host.Spec.UserData)
	}

	// c.Log.Info("ESLEE_tmp: provisioning state", "bmh-state", host.Status.Provisioning.State)
	if host.Status.Provisioning.State == "ready" {
		host.Spec.Online = true
	}

	return nil
}

func (c *ConfigManager) CreateNodeInitConfig(ctx context.Context) (string, error) {
	c.Log.Info("Creating BootstrapData for the node")

	cloudInitData, err := cloudinit.NewNode(&cloudinit.NodeInput{
		BaseUserData: cloudinit.BaseUserData{
			AdditionalFiles:   c.NodeConfig.Spec.Files,
			NTP:               c.NodeConfig.Spec.NTP,
			CloudInitCommands: c.NodeConfig.Spec.CloudInitCommands,
			Users:             c.NodeConfig.Spec.Users,
		},
	})
	if err != nil {
		c.Log.Error(err, "failed to create node configuration")
		return "", err
	}

	cloudinitName, err := c.storeBootstrapData(ctx, cloudInitData)
	if err != nil {
		c.Log.Error(err, "failed to store bootstrap data")
		return "", err
	}
	return cloudinitName, nil
}

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
	bmhost.Spec.BMC.Address = c.NodeConfig.Spec.BMC.Address
	bmhost.Spec.BMC.CredentialsName = c.NodeConfig.Name + "-bmc-secret"
	bmhost.Spec.BootMode = bmh.BootMode(c.NodeConfig.BootMode())
	bmhost.Spec.BMC.DisableCertificateVerification = true

	// c.Log.Info("ESLEE: BMH info test", "bmhost", bmhost.Spec)

	if err := c.client.Create(ctx, bmhost); err != nil {
		return errors.Wrapf(err, "failed to create BareMetalHost")
	}

	if err := c.storeBMHCredentials(ctx, bmhost); err != nil {
		c.Log.Error(err, "failed to store BMC credentials")
		return err
	}
	return nil
}

// storeBootstrapData creates a new secret with the data passed in as input,
// sets the reference in the configuration status and ready to true.
func (c *ConfigManager) storeBootstrapData(ctx context.Context, data []byte) (string, error) {
	c.Log.Info("Store the Bootstrap data", "ready", c.NodeConfig.Status.Ready, "secret", c.NodeConfig.Status.DataSecretName)
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

	//Deprecated -> changing code position
	// c.NodeConfig.Status.UserData = &corev1.SecretReference{
	// 	Name:      secret.Name,
	// 	Namespace: secret.Namespace,
	// }
	// scope.Info("ESLEE_TMP: Store the Bootstrap data - success!", "status.secret", scope.Config.Status.DataSecretName, "status.ready", scope.Config.Status.Ready)
	return secret.Name, nil
}

// storeBMHCredentials creates a new secret with the BMH data passed in as input
func (c *ConfigManager) storeBMHCredentials(ctx context.Context, bmhost *bmh.BareMetalHost) error {
	c.Log.Info("Store the BMC secret", "BMC", c.NodeConfig.Spec.BMC)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.NodeConfig.Name + "-bmc-secret",
			Namespace: c.NodeConfig.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: bmh.GroupVersion.String(),
					Kind:       "BareMetalHost",
					Name:       bmhost.Name,
					UID:        bmhost.UID,
					Controller: pointer.BoolPtr(true),
				},
			},
		},
		Type: "Opaque",
		Data: map[string][]byte{
			"username": []byte(c.NodeConfig.Spec.BMC.Username),
			"password": []byte(c.NodeConfig.Spec.BMC.Password),
		},
	}

	if err := c.client.Create(ctx, secret); err != nil {
		return errors.Wrapf(err, "failed to create BMC secret for BareMetalHost %s/%s", c.NodeConfig.Namespace, c.NodeConfig.Name)
	}
	return nil
}

// Deprecated: v0.0.3
// ensureAnnotation makes sure the config has an annotation that references the
// host and uses the API to update the config if necessary.
func (c *ConfigManager) EnsureAnnotation(ctx context.Context) error { //, host *bmh.BareMetalHost) error {
	// hostKey := c.NodeConfig.ObjectMeta.GetNamespace() + "/" + c.NodeConfig.ObjectMeta.GetName() //	GetAnnotations()
	// annotations := make(map[string]string)
	// existing, ok := annotations[HostAnnotation]
	// if ok {
	// 	if existing == hostKey {
	// 		return nil
	// 	}
	// 	c.Log.Info("Warning: found stray annotation for host on machine. Overwriting.", "host", existing)
	// }
	// annotations[HostAnnotation] = hostKey
	// c.NodeConfig.ObjectMeta.SetAnnotations(annotations)

	// c.Log.Info("ESLEE_TMP: set annotation", "key", hostKey, "val", annotations)
	return nil
}

// Deprecated: v0.0.3
// HasAnnotation makes sure the nodeconfig has an annotation that references a host
func (c *ConfigManager) HasAnnotation() bool {
	// annotations := c.NodeConfig.ObjectMeta.GetAnnotations()
	// if annotations == nil {
	// 	return false
	// }
	// _, ok := annotations[HostAnnotation]
	// return ok
	return true
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
