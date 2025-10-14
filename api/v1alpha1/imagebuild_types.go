/*
Copyright 2025.

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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

const ImageBuildFinalizer = "bib.cluster.x-k8s.io/imagebuild"

// --- Provisioner Definitions ---

// AnsibleSpec defines the parameters for Ansible-based provisioning.
type AnsibleSpec struct {
	// Repo is the URL of a Git repository containing Ansible playbooks.
	// +kubebuilder:validation:Required
	Repo string `json:"repo"`

	// CredentialsSecretName is the name of a Secret used for pulling the Git repository.
	// The secret must be of type 'kubernetes.io/ssh-auth' or 'kubernetes.io/basic-auth'.
	// +optional
	CredentialsSecretName string `json:"credentialsSecretName,omitempty"`

	// Branch is the Git branch to check out. Defaults to "main".
	// +kubebuilder:default:="main"
	// +optional
	Branch string `json:"branch,omitempty"`

	// Playbook is the path to the main playbook file within the repo.
	// +kubebuilder:validation:Required
	Playbook string `json:"playbook"`

	// ExtraVars is a raw JSON object of key-value pairs to be passed as extra variables to the playbook.
	// Corresponds to the --extra-vars or -e flag.
	// +kubebuilder:pruning:PreserveUnknownFields
	// +optional
	ExtraVars *apiextensionsv1.JSON `json:"extraVars,omitempty"`
}

// [Future Support] PackerSpec defines the parameters for Packer-based provisioning.
type PackerSpec struct {
	// Repo is the URL of a Git repository containing Packer templates.
	// +kubebuilder:validation:Required
	Repo string `json:"repo"`

	// CredentialsSecretName is the name of a Secret used for pulling the Git repository.
	// The secret must be of type 'kubernetes.io/ssh-auth' or 'kubernetes.io/basic-auth'.
	// +optional
	CredentialsSecretName string `json:"credentialsSecretName,omitempty"`

	// Branch is the Git branch to check out.
	// +optional
	Branch string `json:"branch,omitempty"`

	// TemplatePath is the path to the Packer template file (HCL or JSON) within the repo.
	// +kubebuilder:validation:Required
	TemplatePath string `json:"templatePath"`
}

// +kubebuilder:validation:XValidation:rule="(has(self.ansible) ? 1 : 0) + (has(self.packer) ? 1 : 0) <= 1",message="at most one of ansible or packer can be specified"
// ProvisionerSpec defines the provisioning method and its parameters.
type ProvisionerSpec struct {
	// +optional
	Ansible *AnsibleSpec `json:"ansible,omitempty"`
	// +optional
	Packer *PackerSpec `json:"packer,omitempty"`
}

// --- Output Definitions ---

// OutputFormat defines the supported artifact formats.
// +kubebuilder:validation:Enum=tgz;qcow2
type OutputFormat string

const (
	// FormatTGZ specifies a .tar.gz rootfs archive.
	FormatTGZ OutputFormat = "tgz"
	// FormatQCOW2 specifies a QEMU Copy-On-Write v2 disk image.
	FormatQCOW2 OutputFormat = "qcow2"
)

// PVCOutput defines a PersistentVolumeClaim as the output destination.
type PVCOutput struct {
	// Name of the PersistentVolumeClaim in the same namespace.
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// SubPath is an optional path within the PVC to store artifacts.
	// If not specified, the operator will create a default path in the format "<namespace>/<imagebuild-name>".
	// +optional
	SubPath string `json:"subPath,omitempty"`

	// CreateIfMissing, if true, instructs the operator to create the PVC if it does not exist.
	// +kubebuilder:default:=false
	// +optional
	CreateIfMissing bool `json:"createIfMissing,omitempty"`
}

// ObjectStorageOutput defines an S3-compatible bucket as the output destination.
type ObjectStorageOutput struct {
	// Bucket is the name of the S3 bucket to upload to.
	// +kubebuilder:validation:Required
	Bucket string `json:"bucket"`

	// Region for the bucket.
	// +optional
	Region string `json:"region,omitempty"`

	// CredentialsSecretName is the name of a Secret containing the access credentials.
	// The secret must contain keys `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.
	// +kubebuilder:validation:Required
	CredentialsSecretName string `json:"credentialsSecretName"`
}

// RegistryOutput defines a container image registry as the output destination.
type RegistryOutput struct {
	// Destination is the full destination path for the container image (e.g., "quay.io/my-org/my-image:latest").
	// +kubebuilder:validation:Required
	Destination string `json:"destination"`

	// PullSecretName is the name of a 'kubernetes.io/dockerconfigjson' secret for registry authentication.
	// +kubebuilder:validation:Required
	PullSecretName string `json:"pullSecretName"`
}

// +kubebuilder:validation:XValidation:rule="(has(self.pvc) ? 1 : 0) + (has(self.objectStorage) ? 1 : 0) + (has(self.registry) ? 1 : 0) == 1",message="exactly one of pvc, objectStorage, or registry must be specified"
// OutputSpec defines the destination for the built artifacts.
type OutputSpec struct {
	// ImageName is a base name for the output files (e.g., "ubuntu-2204-kube-1.29").
	// Not used for the Registry output type, as the name is part of the destination.
	// +optional
	ImageName string `json:"imageName,omitempty"`

	// +optional
	PVC *PVCOutput `json:"pvc,omitempty"`
	// +optional
	ObjectStorage *ObjectStorageOutput `json:"objectStorage,omitempty"`
	// +optional
	Registry *RegistryOutput `json:"registry,omitempty"`

	// Formats is the list of artifact formats to produce.
	// Supported values are "tgz" (for a .tar.gz rootfs archive) and "qcow2".
	// Defaults to ["tgz", "qcow2"] if not specified.
	// +kubebuilder:default:={"tgz", "qcow2"}
	// +optional
	Formats []OutputFormat `json:"formats,omitempty"`
}

// --- Publish Definitions ---

// AWSPublishSpec defines the parameters for publishing the image as an AMI in AWS.
type AWSPublishSpec struct {
	// Region is the AWS region where the AMI will be created.
	// +kubebuilder:validation:Required
	Region string `json:"region"`

	// AMIName is the name for the created AMI.
	// +kubebuilder:validation:Required
	AMIName string `json:"amiName"`

	// InstanceType is the instance type to use for the import task. e.g. "t3.small".
	// See https://docs.aws.amazon.com/vm-import/latest/userguide/vmie_prereqs.html#vmimport-instance-types
	// +kubebuilder:validation:Required
	InstanceType string `json:"instanceType"`

	// SourceS3Bucket is the name of an S3 bucket the operator can use to temporarily
	// upload the qcow2 image for the AMI import process.
	// +kubebuilder:validation:Required
	SourceS3Bucket string `json:"sourceS3Bucket"`

	// CredentialsSecretName is the name of a Secret containing the AWS credentials.
	// The secret must contain keys `AWS_ACCESS_KEY_ID` and `AWS_SECRET_ACCESS_KEY`.
	// +kubebuilder:validation:Required
	CredentialsSecretName string `json:"credentialsSecretName"`
}

// MaaSPublishSpec defines the parameters for publishing the image to a MaaS server.
type MaaSPublishSpec struct {
	// APIURL is the URL of the MaaS API endpoint (e.g., "http://maas.example.com/MAAS").
	// +kubebuilder:validation:Required
	APIURL string `json:"apiUrl"`

	// ImageName is the name for the image being uploaded to MaaS.
	// +kubebuilder:validation:Required
	ImageName string `json:"imageName"`

	// CredentialsSecretName is the name of a Secret containing the MaaS API key.
	// The secret must contain a key named `MAAS_API_KEY`.
	// +kubebuilder:validation:Required
	CredentialsSecretName string `json:"credentialsSecretName"`
}

// +kubebuilder:validation:XValidation:rule="(has(self.aws) ? 1 : 0) + (has(self.maas) ? 1 : 0) == 1",message="exactly one of aws or maas must be specified"
// PublishSpec defines the target infrastructure provider to publish the image to.
type PublishSpec struct {
	// +optional
	AWS *AWSPublishSpec `json:"aws,omitempty"`
	// +optional
	MaaS *MaaSPublishSpec `json:"maas,omitempty"`
}

// ImageBuildSpec defines the desired state of ImageBuild.
type ImageBuildSpec struct {
	// Architecture specifies the target architecture for the build.
	// Supported values are "amd64" and "arm64".
	// +kubebuilder:validation:Enum=amd64;arm64
	// +kubebuilder:default:="amd64"
	// +optional
	Architecture string `json:"arch,omitempty"`

	// BaseImage is the starting container image for the build.
	BaseImage string `json:"baseImage"`

	// BaseImagePullSecretName is the name of a 'kubernetes.io/dockerconfigjson' secret
	// to use for pulling the BaseImage from a private registry.
	// +optional
	BaseImagePullSecretName string `json:"baseImagePullSecretName,omitempty"`

	// Provisioner defines the build steps. This is optional.
	// If omitted, the base image's filesystem will be used directly.
	// +optional
	Provisioner *ProvisionerSpec `json:"provisioner,omitempty"`

	// Output defines where the final artifacts should be stored.
	Output OutputSpec `json:"output"`

	// Publish defines the final infrastructure provider target. This is optional.
	// If omitted, only the artifacts in 'output' will be created.
	// +optional
	Publish *PublishSpec `json:"publish,omitempty"`
}

// ImageBuildPhase represents the high-level state of the build.
type ImageBuildPhase string

const (
	// PhasePending is the initial state before any action is taken.
	PhasePending ImageBuildPhase = "Pending"
	// PhaseBuilding means the builder pod is running.
	PhaseBuilding ImageBuildPhase = "Building"
	// PhasePublishing means the build is complete and artifacts are being published.
	PhasePublishing ImageBuildPhase = "Publishing"
	// PhaseSucceeded means the build and any publishing steps completed successfully.
	PhaseSucceeded ImageBuildPhase = "Succeeded"
	// PhaseFailed means the build or a publishing step has failed.
	PhaseFailed ImageBuildPhase = "Failed"
)

const (
	BaseImageReady   clusterv1beta1.ConditionType = "BaseImageReady"
	BuilderPodReady  clusterv1beta1.ConditionType = "BuilderPodReady"
	ProvisionerReady clusterv1beta1.ConditionType = "ProvisionerReady"
	OutputReady      clusterv1beta1.ConditionType = "OutputReady"
	PublishReady     clusterv1beta1.ConditionType = "PublishReady"
)

// ImageBuildContitionTypes is the list of all condition types.
var ImageBuildConditionTypes = []clusterv1beta1.ConditionType{
	BaseImageReady,
	BuilderPodReady,
	ProvisionerReady,
	OutputReady,
	PublishReady,
}

// ImageBuildStatus defines the observed state of ImageBuild.
type ImageBuildStatus struct {
	// Phase is a simple, high-level summary of the current build state.
	// +optional
	Phase ImageBuildPhase `json:"phase,omitempty"`

	// Conditions represent the latest available observations of an ImageBuild's state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions clusterv1beta1.Conditions `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type"`

	// StartTime is the time at which the build pod was created.
	// +optional
	StartTime *metav1.Time `json:"startTime,omitempty"`

	// CompletionTime is the time at which the build pod finished.
	// +optional
	CompletionTime *metav1.Time `json:"completionTime,omitempty"`

	// BuilderPodName is the name of the pod executing the build.
	// +optional
	BuilderPodName string `json:"builderPodName,omitempty"`

	// OutputURL is the final location of the built artifact, such as an S3 URL or container image reference.
	// +optional
	OutputURL string `json:"outputURL,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="BaseImage",type="string",JSONPath=".spec.baseImage"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type=='Ready')].reason"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ImageBuild is the Schema for the imagebuilds API
type ImageBuild struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ImageBuildSpec   `json:"spec,omitempty"`
	Status ImageBuildStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ImageBuildList contains a list of ImageBuild
type ImageBuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageBuild `json:"items"`
}

// GetConditions returns the list of conditions for an ImageBuild API object.
func (ib *ImageBuild) GetConditions() clusterv1beta1.Conditions {
	return ib.Status.Conditions
}

// SetConditions will set the given conditions on an ImageBuild object.
func (ib *ImageBuild) SetConditions(conditions clusterv1beta1.Conditions) {
	ib.Status.Conditions = conditions
}

func init() {
	SchemeBuilder.Register(&ImageBuild{}, &ImageBuildList{})
}
