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

	"github.com/tmax-cloud/nodeconfig-operator/util"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/cluster-api/util/patch"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
//+kubebuilder:rbac:groups=metal3.io,resources=baremetalhosts/spec,verbs=get;update;patch
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

	// Todo: input NodeConfig Validation with webhook
	// Fetch the NodeConfig instance.
	config := &bootstrapv1.NodeConfig{}
	if err := r.Client.Get(ctx, req.NamespacedName, config); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Can't find the NodeConfig. maybe it was deleted")
			return ctrl.Result{}, nil
		}
		log.Info("Unknown: Can't find the NodeConfig")
		return ctrl.Result{}, err
	}

	// Do nothing if the state of NC is already 'ready'
	// log.Info("NC status check", "NC.Status.Ready", config.Status.Ready)
	if config.Status.Ready {
		log.Info("The work related to NodeConfig has already completed", "config name", config.Name)
		return ctrl.Result{}, nil
	}

	// Create a helper for managing the baremetal container hosting the machine.
	configMgr, err := r.ConfigManager.NewConfigManager(r.Client, config, log)
	if err != nil {
		return ctrl.Result{}, errors.Wrapf(err, "Failed to create helper for managing the configMgr")
	}

	// Initialize the patch helper.
	patchHelper, err := patch.NewHelper(config, r.Client)
	if err != nil {
		return ctrl.Result{}, errors.Wrap(err, "failed to init patch helper")
	}
	// // Always patch nodeconfig exiting this function so we can persist any nodeconfig changes.
	defer func() {
		if err := patchHelper.Patch(ctx, configMgr.NodeConfig); err != nil {
			log.Info("failed to Patch nodeconfig")
		}
		log.Info("End nodeconfig operator reconcile", "NodeConfig.Status", configMgr.NodeConfig.Status)
	}()

	// Create CloudInit data as nodeinitconfig
	if cloudinitName, err := configMgr.CreateNodeInitConfig(ctx); err != nil {
		configMgr.SetError("Failed to create a cloudinit config")
		return ctrl.Result{}, err
	} else {
		// Set secret reference
		configMgr.NodeConfig.Status.UserData = &corev1.SecretReference{
			Name:      cloudinitName,
			Namespace: config.Namespace,
		}
	}

	// Create the BareMetalHost CR
	if bmh, isAvail := configMgr.FindHost(ctx); bmh == nil {
		log.Info("The BMH looking for was not found. Now create a BMH")
		if err := configMgr.CreateBareMetalHost(ctx); err != nil {
			configMgr.SetError("Failed to create BareMetalHost")
			return ctrl.Result{}, err
		}
	} else if !isAvail {
		configMgr.SetError("The found BareMetalHost is not available. " +
			"BMH.provisioning.state: " + string(bmh.Status.Provisioning.State))
		// Delete the NodeConfig
		log.Info("The found BareMetalHost is not available. Delete the nodeconfig")

		if err := r.Client.Delete(ctx, bmh); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "Failed to delete the BMH %s/%s", config.Namespace, config.Name)
		}
		if err := r.Client.Delete(ctx, config); err != nil {
			return ctrl.Result{}, errors.Wrapf(err, "Failed to delete the NodeConfig %s/%s", config.Namespace, config.Name)
		}
		return ctrl.Result{}, nil
	}

	// Associate the baremetalhost hosting the machine
	if err = configMgr.Associate(ctx, config); err != nil {
		configMgr.SetError("Failed to associate the NodeConfig to a BaremetalHost")
		return ctrl.Result{}, errors.Wrapf(err, "Failed to associate the NodeConfig to a BaremetalHost")
	}

	configMgr.NodeConfig.Status.Ready = true
	// Initialize the patch helper.
	if err := patchHelper.Patch(ctx, config); err != nil {
		log.Info("Failed to patch the nodeconfig")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
