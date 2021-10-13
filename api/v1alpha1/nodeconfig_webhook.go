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
	"fmt"
	"os/exec"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/errors"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var nodeconfiglog = logf.Log.WithName("nodeconfig-resource")

// SetupWebhookWithManager init webhook
func (r *NodeConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
//kubebuilder:webhook:path=/mutate-bootstrap-tmax-io-v1alpha1-nodeconfig,mutating=true,failurePolicy=fail,sideEffects=None,groups=bootstrap.tmax.io,resources=nodeconfigs,verbs=create;update,versions=v1alpha1,name=mnodeconfig.kb.io,admissionReviewVersions={v1,v1beta1}
var _ webhook.Defaulter = &NodeConfig{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *NodeConfig) Default() {
	nodeconfiglog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
//+kubebuilder:webhook:path=/validate-bootstrap-tmax-io-v1alpha1-nodeconfig,mutating=false,failurePolicy=fail,sideEffects=None,groups=bootstrap.tmax.io,resources=nodeconfigs,verbs=create;update,versions=v1alpha1,name=vnodeconfig.kb.io,admissionReviewVersions={v1,v1beta1}

var _ webhook.Validator = &NodeConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *NodeConfig) ValidateCreate() error {
	nodeconfiglog.Info("validate create", "name", r.Name)
	var errs []error

	if r.Spec.Image == nil || r.Spec.Image.Checksum == "" {
		errs = append(errs, fmt.Errorf("image value not set"))
		return errors.NewAggregate(errs)
	}
	if r.Spec.BMC == nil {
		errs = append(errs, fmt.Errorf("BMC value not set"))
		return errors.NewAggregate(errs)
	}

	if err := r.osImageValidation(r.Spec.Image.URL, r.Spec.Image.Checksum); err != nil {
		errs = append(errs, err)
		return errors.NewAggregate(errs)
	}
	if err := r.bmcValidation(r.Spec.BMC); err != nil {
		errs = append(errs, err)
		return errors.NewAggregate(errs)
	}

	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *NodeConfig) ValidateUpdate(old runtime.Object) error {
	nodeconfiglog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *NodeConfig) ValidateDelete() error {
	nodeconfiglog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *NodeConfig) osImageValidation(imageURL string, checksumURL string) error {
	cmd := exec.Command("curl", checksumURL)
	stdout, err := cmd.Output()
	slices := strings.Split(string(stdout[:]), " ")

	if err != nil {
		return err
	}
	if len(slices) != 3 {
		return fmt.Errorf("checksum format error")
	}
	if !strings.Contains(imageURL, strings.TrimSpace(string(slices[2]))) {
		return fmt.Errorf("checksum is different from image name")
	}
	return nil
}

func (r *NodeConfig) bmcValidation(bmcInfo *BMC) error {
	bmcAddr := bmcInfo.Address  // "192.168.111.204"
	bmcUser := bmcInfo.Username // "USERID"
	bmcPwd := bmcInfo.Password  // "PASSW0RD"

	cmd := exec.Command("ipmitool", "-I", "lanplus", "-H", bmcAddr, "-U", bmcUser, "-P", bmcPwd, "power", "status")
	if stdout, err := cmd.Output(); err != nil {
		return fmt.Errorf("failed to BMC validation. check the BMC address or account")
	} else if !strings.Contains(string(stdout), "Chassis Power") {
		return fmt.Errorf("unknown error. IPMI connection failed")
	}
	return nil
}
