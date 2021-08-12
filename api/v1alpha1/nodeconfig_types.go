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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Format specifies the output format of the bootstrap data
// +kubebuilder:validation:Enum=cloud-config
type Format string

const (
	// CloudConfig make the bootstrap data to be of cloud-config format
	CloudConfig Format = "cloud-config"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// NodeConfigSpec defines the desired state of NodeConfig
type NodeConfigSpec struct {
	// BMC specifies the BMC configuration
	BMC *BMC `json:"bmc"`

	// Image holds the details of the image to be provisioned.
	Image *Image `json:"image"`

	// Files specifies extra files to be passed to user_data upon creation.
	// +optional
	Files []File `json:"files,omitempty"`

	// CloudInitCommands specifies extra commands to run after systemd
	// +optional
	CloudInitCommands []string `json:"cloudInitCommands,omitempty"`

	// Users specifies extra users to add
	// +optional
	Users []User `json:"users,omitempty"`

	// NTP specifies NTP configuration
	// +optional
	NTP *NTP `json:"ntp,omitempty"`

	// Format specifies the output format of the bootstrap data
	// +optional
	// Format Format `json:"format,omitempty"`

}

// NodeConfigStatus defines the observed state of NodeConfig
type NodeConfigStatus struct {
	// Ready indicates the BootstrapData field is ready to be consumed
	Ready bool `json:"ready,omitempty"`

	// DataSecretName is the name of the secret that stores the bootstrap data script.
	// +optional
	DataSecretName *string `json:"dataSecretName,omitempty"`

	// UserData references the Secret that holds user data needed by the bare metal
	// operator. The Namespace is optional; it will default to the metal3machine's
	// namespace if not specified.
	UserData *corev1.SecretReference `json:"userData,omitempty"`

	// BootstrapData will be a cloud-init script for now.
	//
	// Deprecated: This field has been deprecated in v1alpha3 and
	// will be removed in a future version. Switch to DataSecretName.
	//
	// +optional
	// BootstrapData []byte `json:"bootstrapData,omitempty"`

	// FailureReason will be set on non-retryable errors
	// +optional
	FailureReason string `json:"failureReason,omitempty"`

	// FailureMessage will be set in the event that there is a terminal problem
	// reconciling the metal3machine and will contain a more verbose string suitable
	// for logging and human consumption.
	//
	// This field should not be set for transitive errors that a controller
	// faces that are expected to be fixed automatically over
	// time (like service outages), but instead indicate that something is
	// fundamentally wrong with the metal3machine's spec or the configuration of
	// the controller, and that manual intervention is required. Examples
	// of terminal errors would be invalid combinations of settings in the
	// spec, values that are unsupported by the controller, or the
	// responsible controller itself being critically misconfigured.
	//
	// Any transient errors that occur during the reconciliation of
	// metal3machines can be added as events to the metal3machine object
	// and/or logged in the controller's output.
	// +optional
	FailureMessage *string `json:"failureMessage,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// NodeConfig is the Schema for the nodeconfigs API
type NodeConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   NodeConfigSpec   `json:"spec,omitempty"`
	Status NodeConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// NodeConfigList contains a list of NodeConfig
type NodeConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []NodeConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&NodeConfig{}, &NodeConfigList{})
}

// Encoding specifies the cloud-init file encoding.
// +kubebuilder:validation:Enum=base64;gzip;gzip+base64
type Encoding string

const (
	// Base64 implies the contents of the file are encoded as base64.
	Base64 Encoding = "base64"
	// Gzip implies the contents of the file are encoded with gzip.
	Gzip Encoding = "gzip"
	// GzipBase64 implies the contents of the file are first base64 encoded and then gzip encoded.
	GzipBase64 Encoding = "gzip+base64"
)

// BootMode is the boot mode of the system
// +kubebuilder:validation:Enum=UEFI;legacy
type BootMode string

// Allowed boot mode from metal3
const (
	UEFI            BootMode = "UEFI"
	Legacy          BootMode = "legacy"
	DefaultBootMode BootMode = UEFI
)

// BootMode returns the boot method to use for the host.
func (nc *NodeConfig) BootMode() BootMode {
	mode := nc.Spec.BMC.BootMode
	if mode == "" {
		return DefaultBootMode
	}
	return mode
}

// BMC contains the information necessary to communicate with
// the baremetal host
type BMC struct {
	// Address holds the URL for accessing the controller on the network.
	Address string `json:"address"`

	// Which MAC address will PXE boot? This is optional for some
	// types, but required for libvirt VMs driven by vbmc.
	// +kubebuilder:validation:Pattern=`[0-9a-fA-F]{2}(:[0-9a-fA-F]{2}){5}`
	// +optional
	BootMACAddress string `json:"bootMACAddress,omitempty"`

	// Select the method of initializing the hardware during
	// boot. Defaults to UEFI.
	// +optional
	BootMode BootMode `json:"bootMode,omitempty"`

	// ID/PW for authenticating with the BMC
	Username string `json:"username"`
	Password string `json:"password"`
}

// ChecksumType holds the algorithm name for the checksum
// +kubebuilder:validation:Enum=md5;sha256;sha512
type ChecksumType string

const (
	// MD5 checksum type
	MD5 ChecksumType = "md5"
	// SHA256 checksum type
	SHA256 ChecksumType = "sha256"
	// SHA512 checksum type
	SHA512 ChecksumType = "sha512"

	DefaultChecksumType ChecksumType = MD5
)

// ChecksumType returns the checksum method to use for image validation.
func (nc *NodeConfig) ChecksumType() ChecksumType {
	checksum_type := nc.Spec.Image.ChecksumType
	if checksum_type == "" {
		return DefaultChecksumType
	}
	return checksum_type
}

// Image holds the details of an image either to provisioned or that
// has been provisioned.
type Image struct {
	// URL is a location of an image to deploy.
	URL string `json:"url"`

	// Checksum is the checksum for the image.
	Checksum string `json:"checksum"`

	// ChecksumType is the checksum algorithm for the image.
	// e.g md5, sha256, sha512
	ChecksumType ChecksumType `json:"checksumType,omitempty"`
}

func (nc *NodeConfig) checkBMHDetails() bool {
	if nc.Spec.BMC.Address != "" &&
		nc.Spec.BMC.Username != "" &&
		nc.Spec.BMC.Password != "" &&
		nc.Spec.Image.URL != "" &&
		nc.Spec.Image.Checksum != "" {
		return true
	}
	return false
}

// File defines the input for generating write_files in cloud-init.
type File struct {
	// Path specifies the full path on disk where to store the file.
	Path string `json:"path"`

	// Owner specifies the ownership of the file, e.g. "root:root".
	// +optional
	Owner string `json:"owner,omitempty"`

	// Permissions specifies the permissions to assign to the file, e.g. "0640".
	// +optional
	Permissions string `json:"permissions,omitempty"`

	// Encoding specifies the encoding of the file contents.
	// +optional
	Encoding Encoding `json:"encoding,omitempty"`

	// Content is the actual content of the file.
	Content string `json:"content"`
}

// User defines the input for a generated user in cloud-init.
type User struct {
	// Name specifies the user name
	Name string `json:"name"`

	// Gecos specifies the gecos to use for the user
	// +optional
	Gecos *string `json:"gecos,omitempty"`

	// Groups specifies the additional groups for the user
	// +optional
	Groups *string `json:"groups,omitempty"`

	// HomeDir specifies the home directory to use for the user
	// +optional
	HomeDir *string `json:"homeDir,omitempty"`

	// Inactive specifies whether to mark the user as inactive
	// +optional
	Inactive *bool `json:"inactive,omitempty"`

	// Shell specifies the user's shell
	// +optional
	Shell *string `json:"shell,omitempty"`

	// Passwd specifies a hashed password for the user
	// +optional
	Passwd *string `json:"passwd,omitempty"`

	// PrimaryGroup specifies the primary group for the user
	// +optional
	PrimaryGroup *string `json:"primaryGroup,omitempty"`

	// LockPassword specifies if password login should be disabled
	// +optional
	LockPassword *bool `json:"lockPassword,omitempty"`

	// Sudo specifies a sudo role for the user
	// +optional
	Sudo *string `json:"sudo,omitempty"`

	// SSHAuthorizedKeys specifies a list of ssh authorized keys for the user
	// +optional
	SSHAuthorizedKeys []string `json:"sshAuthorizedKeys,omitempty"`
}

// NTP defines input for generated ntp in cloud-init
type NTP struct {
	// Servers specifies which NTP servers to use
	// +optional
	Servers []string `json:"servers,omitempty"`

	// Enabled specifies whether NTP should be enabled
	// +optional
	Enabled *bool `json:"enabled,omitempty"`
}
