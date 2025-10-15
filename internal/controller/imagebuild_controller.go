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

package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/cluster-api/util/conditions"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	bibv1alpha1 "github.com/zarcen/bib-operator/api/v1alpha1"
	"github.com/zarcen/bib-operator/internal/scope"
	clusterv1beta1 "sigs.k8s.io/cluster-api/api/v1beta1"
)

var builderPodPrefix = "imgbldr-"

// ImageBuildReconciler reconciles a ImageBuild object
type ImageBuildReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=bib.cluster.x-k8s.io,resources=imagebuilds,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=bib.cluster.x-k8s.io,resources=imagebuilds/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=bib.cluster.x-k8s.io,resources=imagebuilds/finalizers,verbs=update
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=persistentvolumeclaims,verbs=get;list;watch;create

func (r *ImageBuildReconciler) Reconcile(ctx context.Context, req ctrl.Request) (retRes ctrl.Result, reterr error) {
	logger := log.FromContext(ctx)

	// Fetch the ImageBuild resource
	var ib bibv1alpha1.ImageBuild
	if err := r.Get(ctx, req.NamespacedName, &ib); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("ImageBuild resource not found. Ignoring since object must be deleted.")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get ImageBuild resource")
		return ctrl.Result{}, err
	}
	// Add the finalizer if it doesn't exist
	if !controllerutil.ContainsFinalizer(&ib, bibv1alpha1.ImageBuildFinalizer) {
		controllerutil.AddFinalizer(&ib, bibv1alpha1.ImageBuildFinalizer)
		if err := r.Update(ctx, &ib); err != nil {
			return ctrl.Result{Requeue: true}, err
		}
	}

	// Create a scope for the imagebuild
	ibs, err := scope.NewImageBuildScope(r.Client, logger, &ib)
	if err != nil {
		logger.Error(err, "Failed to create scope for imagebuild")
		return ctrl.Result{}, err
	}
	// Always close the scope when exiting this function so we can persist any changes.
	defer func() {
		if err := ibs.Close(ctx); err != nil && reterr == nil {
			reterr = err
			retRes = ctrl.Result{}
		}
	}()
	ibs.InitializeConditions()

	// Handle deletion
	if !ib.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, ibs)
	}

	// Check if a builder pod already exists
	builderPod := &corev1.Pod{}
	builderPodName := fmt.Sprintf("%s%s", builderPodPrefix, ib.Name)
	err = r.Get(ctx, types.NamespacedName{Name: builderPodName, Namespace: ib.Namespace}, builderPod)

	if err != nil && apierrors.IsNotFound(err) {
		// Pod does not exist, create it
		logger.Info("Builder pod not found. Creating a new one.")

		// Construct the desired pod object
		desiredPod, err := r.constructBuilderPod(ctx, &ib)
		if err != nil {
			logger.Error(err, "Failed to construct builder pod spec")
			conditions.MarkFalse(&ib, bibv1alpha1.BuilderPodReady, "BuildPodNotReady", clusterv1beta1.ConditionSeverityError, "%s", err.Error())
			return ctrl.Result{}, err
		}

		if err := ctrl.SetControllerReference(&ib, desiredPod, r.Scheme); err != nil {
			logger.Error(err, "Failed to set owner reference on builder pod")
			return ctrl.Result{}, err
		}

		// Create the pod in the cluster
		if err := r.Create(ctx, desiredPod); err != nil {
			logger.Error(err, "Failed to create builder pod")
			// TODO: Update status to Failed
			return ctrl.Result{}, err
		}

		// TODO: Update status to Building
		logger.Info("Successfully created builder pod", "PodName", desiredPod.Name)
		return ctrl.Result{Requeue: true}, nil // Requeue to check pod status later
	} else if err != nil {
		logger.Error(err, "Failed to get builder pod")
		return ctrl.Result{}, err
	}

	// 4. If pod exists, check its status (we will implement this logic next)
	logger.Info("Builder pod already exists", "PodPhase", builderPod.Status.Phase)
	// TODO: Handle Pod Succeeded, Failed, etc.

	return ctrl.Result{}, nil
}

// constructBuilderPod creates the Pod resource definition based on the ImageBuild spec.
func (r *ImageBuildReconciler) constructBuilderPod(_ context.Context, imageBuild *bibv1alpha1.ImageBuild) (*corev1.Pod, error) {
	podName := fmt.Sprintf("%s%s", builderPodPrefix, imageBuild.Name)
	privileged := true
	runAsUser := int64(0)

	// Initialize slices for env vars and mounts
	envVars := []corev1.EnvVar{
		{Name: "BASE_IMAGE", Value: imageBuild.Spec.BaseImage},
		{Name: "ARCHITECTURE", Value: imageBuild.Spec.Architecture},
	}
	volumes := []corev1.Volume{
		{Name: "containers-storage", VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}}},
	}
	volumeMounts := []corev1.VolumeMount{
		{Name: "containers-storage", MountPath: "/var/lib/containers/storage"},
	}

	// Check if a pull secret is specified
	if imageBuild.Spec.BaseImagePullSecretName != "" {
		// Define the volume that points to the secret
		volumes = append(volumes, corev1.Volume{
			Name: "baseimage-pull-secret",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: imageBuild.Spec.BaseImagePullSecretName,
				},
			},
		})

		// Mount the secret into the builder container at the expected path
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "baseimage-pull-secret",
			MountPath: "/etc/baseimage-pull-secret",
			ReadOnly:  true,
		})
	}

	// Check if the optional Provisioner field is set
	if imageBuild.Spec.Provisioner != nil {
		// Check which type of provisioner is set (e.g., Ansible)
		if imageBuild.Spec.Provisioner.Ansible != nil {
			envVars = append(envVars,
				corev1.EnvVar{Name: "GIT_REPO", Value: imageBuild.Spec.Provisioner.Ansible.Repo},
				corev1.EnvVar{Name: "GIT_BRANCH", Value: imageBuild.Spec.Provisioner.Ansible.Branch},
				corev1.EnvVar{Name: "PLAYBOOK", Value: imageBuild.Spec.Provisioner.Ansible.Playbook},
			)
			// Add a volume for the git repo
			volumes = append(volumes, corev1.Volume{
				Name:         "source-repo",
				VolumeSource: corev1.VolumeSource{EmptyDir: &corev1.EmptyDirVolumeSource{}},
			})
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      "source-repo",
				MountPath: "/source",
			})
		}
	}

	// Check if the optional PVC output field is set
	if imageBuild.Spec.Output.PVC != nil {
		envVars = append(envVars, corev1.EnvVar{Name: "OUTPUT_FILENAME", Value: imageBuild.Spec.Output.ImageName})
		volumes = append(volumes, corev1.Volume{
			Name: "output-pvc",
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: imageBuild.Spec.Output.PVC.Name,
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "output-pvc",
			MountPath: "/output",
		})
	}

	// Create a nodeSelector map based on the requested architecture.
	nodeSelector := make(map[string]string)
	if imageBuild.Spec.Architecture != "" {
		nodeSelector["kubernetes.io/arch"] = imageBuild.Spec.Architecture
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: imageBuild.Namespace,
		},
		Spec: corev1.PodSpec{
			NodeSelector:  nodeSelector,
			RestartPolicy: corev1.RestartPolicyNever,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser: &runAsUser,
			},
			Containers: []corev1.Container{
				{
					Name:  "builder",
					Image: "ghcr.io/zarcen/bib-operator/builder:0.1.1",
					SecurityContext: &corev1.SecurityContext{
						Privileged: &privileged,
					},
					Env:          envVars,
					VolumeMounts: volumeMounts,
				},
			},
			Volumes: volumes,
		},
	}
	return pod, nil
}

// cleanupBuilderPod deletes the builder Pod resource if it exists.
func (r *ImageBuildReconciler) cleanupBuilderPod(ctx context.Context, imageBuild *bibv1alpha1.ImageBuild) error {
	podName := fmt.Sprintf("%s%s", builderPodPrefix, imageBuild.Name)
	err := r.Delete(ctx, &corev1.Pod{ObjectMeta: metav1.ObjectMeta{Name: podName, Namespace: imageBuild.Namespace}})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

func (r *ImageBuildReconciler) reconcileDelete(ctx context.Context, ibs *scope.ImageBuildScope) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	imageBuild := ibs.ImageBuild

	// redundant check the deletion timestamp
	if !imageBuild.ObjectMeta.DeletionTimestamp.IsZero() {
		if controllerutil.ContainsFinalizer(imageBuild, bibv1alpha1.ImageBuildFinalizer) {
			logger.Info("Performing cleanup for ImageBuild resource")

			// cleanup logic (e.g., delete the builder pod if it's running)
			err := r.cleanupBuilderPod(ctx, imageBuild)
			if err != nil {
				logger.Error(err, "Failed to cleanup builder pod")
				// TODO: Update status to Failed
				return ctrl.Result{}, err
			}

			controllerutil.RemoveFinalizer(imageBuild, bibv1alpha1.ImageBuildFinalizer)
			if err := r.Update(ctx, imageBuild); err != nil {
				return ctrl.Result{}, err
			}
		}
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ImageBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&bibv1alpha1.ImageBuild{}).
		Owns(&corev1.Pod{}). // watch Pods created by ImageBuild resources
		Named("imagebuild").
		Complete(r)
}
