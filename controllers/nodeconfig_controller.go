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
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	//ESLEE bsutil "sigs.k8s.io/cluster-api/bootstrap/util"
	// "sigs.k8s.io/cluster-api/util/patch"

	bootstrapv1 "github.com/tmax-cloud/nodeconfig-operator/api/v1alpha1"
)

// NodeConfigReconciler reconciles a NodeConfig object
type NodeConfigReconciler struct {
	Client client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
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

//+kubebuilder:rbac:groups=cache.tmax.io,resources=nodeconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=cache.tmax.io,resources=nodeconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=cache.tmax.io,resources=nodeconfigs/finalizers,verbs=update

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
	// _ = log.FromContext(ctx)
	log := r.Log.WithValues("nodeconfig", req.NamespacedName)

	// Lookup the node config
	config := &bootstrapv1.NodeConfig{}
	if err := r.Client.Get(ctx, req.NamespacedName, config); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "failed to get config")
		return ctrl.Result{}, err
	}

	//ESLEE // Look up the owner of this NodeConfig if there is one
	// configOwner, err := bsutil.GetConfigOwner(ctx, r.Client, config)
	// if apierrors.IsNotFound(err) {
	// 	// Could not find the owner yet, this is not an error and will rereconcile when the owner gets set.
	// 	return ctrl.Result{}, nil
	// }
	// if err != nil {
	// 	log.Error(err, "failed to get owner")
	// 	return ctrl.Result{}, err
	// }
	// if configOwner == nil {
	// 	return ctrl.Result{}, nil
	// }
	// log = log.WithValues("kind", configOwner.GetKind(), "version", configOwner.GetResourceVersion(), "name", configOwner.GetName())

	scope := &Scope{
		Logger: log,
		Config: config,
		//ESLEE ConfigOwner: configOwner,
	}

	//ESLEE // Initialize the patch helper.
	// patchHelper, err := patch.NewHelper(config, r.Client)
	// if err != nil {
	// 	return ctrl.Result{}, err
	// }

	switch {
	// Migrate plaintext data to secret.
	case config.Status.BootstrapData != nil: // && config.Status.DataSecretName == nil:
		if err := r.storeBootstrapData(ctx, scope, config.Status.BootstrapData); err != nil {
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
		//ESLEE return ctrl.Result{}, patchHelper.Patch(ctx, config)
		// Reconcile status for machines that already have a secret reference, but our status isn't up to date.
		// This case solves the pivoting scenario (or a backup restore) which doesn't preserve the status subresource on objects.
		//ESLEE case configOwner.DataSecretName() != nil && (!config.Status.Ready || config.Status.DataSecretName == nil):
		// 	config.Status.Ready = true
		// 	config.Status.DataSecretName = configOwner.DataSecretName()
		// 	return ctrl.Result{}, patchHelper.Patch(ctx, config)
	}

	// It's a Node join
	return r.joinNode(ctx, scope)
	// return ctrl.Result{}, err
}

func (r *NodeConfigReconciler) joinNode(ctx context.Context, scope *Scope) (_ ctrl.Result, reterr error) {
	scope.Info("Creating BootstrapData for the node")

	//ESLEE verbosityFlag := ""
	// if scope.Config.Spec.Verbosity != nil {
	// 	verbosityFlag = fmt.Sprintf("--v %s", strconv.Itoa(int(*scope.Config.Spec.Verbosity)))
	// }

	cloudInitData, err := cloudinit.NewNode(&cloudinit.NodeInput{
		BaseUserData: cloudinit.BaseUserData{
			AdditionalFiles:   scope.Config.Spec.Files,
			NTP:               scope.Config.Spec.NTP,
			CloudInitCommands: scope.Config.Spec.CloudInitCommands,
			Users:             scope.Config.Spec.Users,
			//ESLEE KubeadmVerbosity:  verbosityFlag,
		},
	})
	if err != nil {
		scope.Error(err, "failed to create node configuration")
		return ctrl.Result{}, err
	}

	if err := r.storeBootstrapData(ctx, scope, cloudInitData); err != nil {
		scope.Error(err, "failed to store bootstrap data")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// storeBootstrapData creates a new secret with the data passed in as input,
// sets the reference in the configuration status and ready to true.
func (r *NodeConfigReconciler) storeBootstrapData(ctx context.Context, scope *Scope, data []byte) error {
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
		return errors.Wrapf(err, "failed to create bootstrap data secret for KubeadmConfig %s/%s", scope.Config.Namespace, scope.Config.Name)
	}

	scope.Config.Status.DataSecretName = pointer.StringPtr(secret.Name)
	scope.Config.Status.Ready = true
	return nil
}
