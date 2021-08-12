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

package controllers

//ESLEE TODO: add owner_referencing fucntion of nodeconfig (owner: a baremetalhost with the same node name)

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/tmax-cloud/nodeconfig-operator/cloudinit"
	"github.com/tmax-cloud/nodeconfig-operator/util"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	//ESLEE bsutil "sigs.k8s.io/cluster-api/bootstrap/util"

	ctrllog "sigs.k8s.io/controller-runtime/pkg/log"

	bootstrapv1 "github.com/tmax-cloud/nodeconfig-operator/api/v1alpha1"
)

// NodeConfigReconciler reconciles a NodeConfig object
type NodeConfigReconciler struct {
	Client        client.Client
	ConfigManager util.ConfigManager
	Log           logr.Logger
	Scheme        *runtime.Scheme
}

type Scope struct {
	logr.Logger
	Config *bootstrapv1.NodeConfig
	//ESLEE ConfigOwner *bsutil.ConfigOwner
}

// SetupWithManager sets up the controller with the Manager.
func (r *NodeConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bootstrapv1.NodeConfig{}).
		Complete(r)
}

//+kubebuilder:rbac:groups=bootstrap.tmax.io,resources=nodeconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bootstrap.tmax.io,resources=nodeconfigs/status,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bootstrap.tmax.io,resources=nodeconfigs/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=secrets;events;configmaps,verbs=get;list;watch;create;update;patch;delete

// Add RBAC rules to access cluster-api resources
//+kubebuilder:rbac:groups=metal3.io,resources=baremetalhosts,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=metal3.io,resources=baremetalhosts/status,verbs=get;update;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the NodeConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.8.3/pkg/reconcile
func (r *NodeConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrllog.FromContext(ctx)
	log.Info("Start nodeconfig operator reconcile")

	// Fetch the NodeConfig instance.
	config := &bootstrapv1.NodeConfig{}
	if err := r.Client.Get(ctx, req.NamespacedName, config); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to get config")
		return ctrl.Result{}, err
	}

	scope := &Scope{
		Logger: log,
		Config: config,
	}

	// Initialize the patch helper.
	patchHelper, err := patch.NewHelper(config, r.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to init patch helper")
	}
	// Always patch nodeconfig exiting this function so we can persist any nodeconfig changes.
	defer func() {
		err := patchHelper.Patch(ctx, config)
		if err != nil {
			log.Info("failed to Patch nodeconfig")
		}
	}()

	// Create a helper for managing the baremetal container hosting the machine.
	configMgr, err := r.ConfigManager.NewConfigManager(r.Client, config, log)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to create helper for managing the configMgr")
	}

	// Check if the nodeconfig was associated with a baremetalhost
	if !configMgr.HasAnnotation() {
		err := configMgr.EnsureAnnotation(ctx)
		if err != nil {
			configMgr.SetError("Failed to annotate the NodeConfig")
			// c.Log.Error(nil, "ESLEE_TMP: Failed to annotate the NodeConfig")
			return ctrl.Result{}, err
		}
		log.Info("ESLEE_TMP: Ends up checking annotation")
	}

	if !config.Status.Ready {
		// log.Info("ESLEE_TMP: the userData secret already created")
		// log.Info("ESLEE_TMP: before createNodeConfig call", "already userdata", config.Status.UserData, "ready?", config.Status.Ready)
		if err := r.createNodeConfig(ctx, scope); err != nil {
			log.Info("ESLEE: createNodeConfig failed!", "err_mgs", err.Error())
			return ctrl.Result{}, err
		}
		if err := patchHelper.Patch(ctx, config); err != nil {
			log.Info("failed to Patch nodeconfig")
			return ctrl.Result{}, err
		}
	}

	// Skip the association
	if config.ObjectMeta.OwnerReferences != nil {
		// log.Info("ESLEE_TMP: already associated", "ownerRef", config.ObjectMeta.OwnerReferences)
		return ctrl.Result{}, nil
	}

	// Create the BareMetalHost CR
	if !configMgr.FindHost(ctx) {
		log.Info("ESLEE: no target BMH founded. Now create the BareMetalHost")

		// ESLEE: todo (not done)
		if err := configMgr.CreateBareMetalHost(ctx); err != nil {
			log.Info("ESLEE: createBareMetalHost failed!", "err_mgs", err.Error())
			return ctrl.Result{}, err
		}
	}

	//Associate the baremetalhost hosting the machine
	err = configMgr.Associate(ctx)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "failed to associate the NodeConfig to a BaremetalHost")
	}

	log.Info("ESLEE: End reconcile")
	return ctrl.Result{}, nil
}

func (r *NodeConfigReconciler) createNodeConfig(ctx context.Context, scope *Scope) error {
	scope.Info("Creating BootstrapData for the node")

	cloudInitData, err := cloudinit.NewNode(&cloudinit.NodeInput{
		BaseUserData: cloudinit.BaseUserData{
			AdditionalFiles:   scope.Config.Spec.Files,
			NTP:               scope.Config.Spec.NTP,
			CloudInitCommands: scope.Config.Spec.CloudInitCommands,
			Users:             scope.Config.Spec.Users,
		},
	})
	if err != nil {
		scope.Error(err, "failed to create node configuration")
		return err
	}

	if err := r.storeBootstrapData(ctx, scope, cloudInitData); err != nil {
		scope.Error(err, "failed to store bootstrap data")
		return err
	}
	return nil
}

// storeBootstrapData creates a new secret with the data passed in as input,
// sets the reference in the configuration status and ready to true.
func (r *NodeConfigReconciler) storeBootstrapData(ctx context.Context, scope *Scope, data []byte) error {
	scope.Info("Store the Bootstrap data", "ready", scope.Config.Status.Ready, "secret", scope.Config.Status.DataSecretName)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      scope.Config.Name,
			Namespace: scope.Config.Namespace,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: bootstrapv1.GroupVersion.String(),
					Kind:       "NodeConfig",
					Name:       scope.Config.Name,
					UID:        scope.Config.UID,
					Controller: pointer.BoolPtr(true),
				},
			},
		},
		Data: map[string][]byte{
			"value": data,
		},
	}

	if err := r.Client.Create(ctx, secret); err != nil {
		return errors.Wrapf(err, "failed to create bootstrap data secret for NodeConfig %s/%s", scope.Config.Namespace, scope.Config.Name)
	}

	// ESLEE: Deprecated datasecretname
	// scope.Config.Status.DataSecretName = pointer.StringPtr(secret.Name)
	scope.Config.Status.Ready = true
	scope.Config.Status.UserData = &corev1.SecretReference{
		Name:      secret.Name,
		Namespace: secret.Namespace,
	}
	// scope.Info("ESLEE_TMP: Store the Bootstrap data - success!", "status.secret", scope.Config.Status.DataSecretName, "status.ready", scope.Config.Status.Ready)
	return nil
}
