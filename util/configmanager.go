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
	corev1 "k8s.io/api/core/v1"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/cache"
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
	ProviderName = "metal3"
	// HostAnnotation is the key for an annotation that should go on a Metal3Machine to
	// reference what BareMetalHost it corresponds to.
	HostAnnotation = "metal3.io/BareMetalHost"
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

	// c.Log.Info("ESLEE_TMP: Before eunsureAnnotation")
	// err := c.EnsureAnnotation(ctx)
	// if err != nil {
	// 	c.SetError("Failed to annotate the NodeConfig")
	// 	// c.Log.Error(nil, "ESLEE_TMP: Failed to annotate the NodeConfig")
	// 	return err
	// }

	// ESLEE_TODO: nodeconifg에서 OS IMG 설정하게 해줄것인가
	// config := c.NodeConfig.Spec
	// err := config.IsValid()
	// if err != nil {
	// 	// Should have been picked earlier. Do not requeue
	// 	m.SetError(err.Error(), capierrors.InvalidConfigurationMachineError)
	// 	return nil
	// }

	// look for associated BMH
	// defer func() {
	bmhost, bmhHelper, err := c.getHost(ctx)
	if err != nil {
		// if host != nil {
		// 	c.Log.Error(err, "Host not nil", "host", host) //,
		// } else {
		// 	c.Log.Error(err, "Host nil") //,
		// }

		c.SetError("Failed to get the BaremetalHost for the NodeConfig")
		return err
	}
	c.Log.Info("ESLEE: Get host success!", "part", bmhost.Status)
	if err = c.setHostSpec(ctx, bmhost); err != nil {
		c.SetError(err.Error())
	}

	c.NodeConfig.ObjectMeta.SetOwnerReferences(
		util.EnsureOwnerRef(c.NodeConfig.GetOwnerReferences(),
			metav1.OwnerReference{
				APIVersion: bmhost.APIVersion,
				Kind:       "BareMetalHost",
				Name:       bmhost.Name,
				UID:        bmhost.UID,
			}))

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
	err = bmhHelper.Patch(ctx, bmhost)
	if err != nil {
		// if aggr, ok := err.(kerrors.Aggregate); ok {
		// 	for _, kerr := range aggr.Errors() {
		// 		if apierrors.IsConflict(kerr) {
		// 			return &RequeueAfterError{}
		// 		}
		// 	}
		// }
		return err
	}

	c.Log.Info("Finished associating machine")
	return nil
}

// GetBaremetalHostID return the provider identifier for this machine
// func (c *ConfigManager) GetBaremetalHostID(ctx context.Context) (*string, error) {
// 	// look for associated BMH
// 	host, err := c.getHost(ctx, c.client, c.Log)
// 	if err != nil {
// 		c.SetError("Failed to get a BaremetalHost for the NodeConfig")
// 		return nil, err
// 	}
// 	if host == nil {
// 		c.Log.Info("BaremetalHost not associated, requeuing")
// 		// 	return nil, &RequeueAfterError{RequeueAfter: requeueAfter}
// 		return nil, err
// 	}
// 	if host.Status.Provisioning.State == bmh.StateProvisioned {
// 		return pointer.StringPtr(string(host.ObjectMeta.UID)), nil
// 	}
// 	c.Log.Info("Provisioning BaremetalHost, requeuing")
// 	// return nil, &RequeueAfterError{RequeueAfter: requeueAfter}
// 	return nil, nil
// }

// findHost return true when it founds the associated host by looking for an annotation
// on the machine that contains a reference to the host.
func (c *ConfigManager) FindHost(ctx context.Context) bool {
	host, err := getHost(ctx, c.NodeConfig, c.client, c.Log)
	// ESLEE: todo - error 발생했으면 status에 찍을 것인가
	if err != nil && host != nil {
		return true
	}
	return false
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

// func getHost(ctx context.Context, cl client.Client, mLog logr.Logger) (*bmh.BareMetalHost, error) {
func getHost(ctx context.Context, nConfig *bootstrapv1.NodeConfig,
	cl client.Client, mLog logr.Logger) (*bmh.BareMetalHost, error) {
	mLog.Info("ESLEE: Start to find host")
	annotations := nConfig.ObjectMeta.GetAnnotations()
	if annotations == nil {
		mLog.Info("ESLEE: no annotation")
		return nil, nil
	}
	hostKey, ok := annotations[HostAnnotation]
	if !ok {
		mLog.Info("ESLEE: no metal3/baremetalhost annotation")
		return nil, nil
	}
	hostNamespace, hostName, err := cache.SplitMetaNamespaceKey(hostKey)
	if err != nil {
		mLog.Error(err, "Error parsing annotation value", "annotation key", hostKey)
		return nil, err
	}

	// mLog.Info("ESLEE_TMP: Where's my BMH")
	host := bmh.BareMetalHost{}
	key := client.ObjectKey{
		Name:      hostName,
		Namespace: hostNamespace,
	}
	// mLog.Info("ESLEE_TMP: hello#1", "host", host, "key", key)
	err = cl.Get(ctx, key, &host)
	// mLog.Info("ESLEE_TMP: hello#2", "host after get", host)
	// mLog.Info("ESLEE_TMP: hello#3", "host namespace", hostNamespace)
	if apierrors.IsNotFound(err) {
		mLog.Info("Annotated host not found", "host", hostKey)
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	mLog.Info("ESLEE_TMP: Found host", "host", hostKey)
	return &host, err
}

// setHostSpec will ensure the host's Spec is set according to the machine's
// details. It will then update the host via the kube API. If UserData does not
// include a Namespace, it will default to the Metal3Machine's namespace.
func (c *ConfigManager) setHostSpec(ctx context.Context, host *bmh.BareMetalHost) error {

	// We only want to update the image setting if the host does not
	// already have an image.
	//
	// A host with an existing image is already provisioned and
	// upgrades are not supported at this time. To re-provision a
	// host, we must fully deprovision it and then provision it again.
	// Not provisioning while we do not have the UserData
	if c.NodeConfig.Status.UserData != nil {
		// if host.Spec.Image == nil && m.Metal3Machine.Status.UserData != nil {
		// checksumType := ""
		// if m.Metal3Machine.Spec.Image.ChecksumType != nil {
		// 	checksumType = *m.Metal3Machine.Spec.Image.ChecksumType
		// }
		// host.Spec.Image = &bmh.Image{
		// 	URL:          m.Metal3Machine.Spec.Image.URL,
		// 	Checksum:     m.Metal3Machine.Spec.Image.Checksum,
		// 	ChecksumType: bmh.ChecksumType(checksumType),
		// 	DiskFormat:   m.Metal3Machine.Spec.Image.DiskFormat,
		// }
		host.Spec.UserData = c.NodeConfig.Status.UserData
		if host.Spec.UserData != nil && host.Spec.UserData.Namespace == "" {
			host.Spec.UserData.Namespace = host.Namespace
		}
		// c.Log.Info("ESLEE_TMP: set BMH", "userdata", host.Spec.UserData)

		// Set metadata from gathering from Spec.metadata and from the template.
		// if m.Metal3Machine.Status.MetaData != nil {
		// 	host.Spec.MetaData = m.Metal3Machine.Status.MetaData
		// }
		// if host.Spec.MetaData != nil && host.Spec.MetaData.Namespace == "" {
		// 	host.Spec.MetaData.Namespace = m.Machine.Namespace
		// }
		// if m.Metal3Machine.Status.NetworkData != nil {
		// 	host.Spec.NetworkData = m.Metal3Machine.Status.NetworkData
		// }
		// if host.Spec.NetworkData != nil && host.Spec.NetworkData.Namespace == "" {
		// 	host.Spec.NetworkData.Namespace = m.Machine.Namespace
		// }
	}
	// Set automatedCleaningMode from metal3Machine.spec.automatedCleaningMode.
	// if host.Spec.AutomatedCleaningMode != bmh.AutomatedCleaningMode(m.Metal3Machine.Spec.AutomatedCleaningMode) {
	// 	host.Spec.AutomatedCleaningMode = bmh.AutomatedCleaningMode(m.Metal3Machine.Spec.AutomatedCleaningMode)
	// }

	host.Spec.Online = true

	return nil
}

// ensureAnnotation makes sure the config has an annotation that references the
// host and uses the API to update the config if necessary.
func (c *ConfigManager) EnsureAnnotation(ctx context.Context) error { //, host *bmh.BareMetalHost) error {
	hostKey := c.NodeConfig.ObjectMeta.GetNamespace() + "/" + c.NodeConfig.ObjectMeta.GetName() //	GetAnnotations()
	annotations := make(map[string]string)
	existing, ok := annotations[HostAnnotation]
	if ok {
		if existing == hostKey {
			return nil
		}
		c.Log.Info("Warning: found stray annotation for host on machine. Overwriting.", "host", existing)
	}
	annotations[HostAnnotation] = hostKey
	c.NodeConfig.ObjectMeta.SetAnnotations(annotations)

	// c.Log.Info("ESLEE_TMP: set annotation", "key", hostKey, "val", annotations)
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

func (c *ConfigManager) CreateBareMetalHost(ctx context.Context) error { //}, scope *Scope) error {
	c.Log.Info("Creating BootstrapData for the node")
	if !c.NodeConfig.CheckBMHDetails() {
		c.Log.Error(nil, "ESLEE: BMH Undefined")
	}
	c.Log.Info("ESLEE: BMH info test",
		"addr", c.NodeConfig.Spec.BMC.Address,
		"userid", c.NodeConfig.Spec.BMC.Username,
		"pw", c.NodeConfig.Spec.BMC.Password,
		"img_url", c.NodeConfig.Spec.Image.URL,
		"img_checksum", c.NodeConfig.Spec.Image.Checksum)

	// ESLEE: Todo - BMH config validation
	bmhost := &bmh.BareMetalHost{}
	bmhost.ObjectMeta.Name = c.NodeConfig.Name
	bmhost.ObjectMeta.Namespace = c.NodeConfig.Namespace
	bmhost.Spec.Online = false
	bmhost.Spec.BMC.Address = c.NodeConfig.Spec.BMC.Address
	bmhost.Spec.BMC.CredentialsName = c.NodeConfig.Name + "-bmc-secret"
	bmhost.Spec.BootMode = bmh.BootMode(c.NodeConfig.BootMode())
	bmhost.Spec.BMC.DisableCertificateVerification = true
	bmhost.Spec.Image = &bmh.Image{
		URL:          c.NodeConfig.Spec.Image.URL,
		Checksum:     string(c.NodeConfig.Spec.Image.Checksum),
		ChecksumType: bmh.ChecksumType(c.NodeConfig.ChecksumType()),
	}

	c.Log.Info("ESLEE: BMH info test", "bmhost", bmhost.Spec)

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
