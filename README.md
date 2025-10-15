# bib-operator

![bib-operator logo](assets/bib-operator-logo.png)

**bib** (**B**ootable **I**mage **B**uilder) is a Kubernetes operator for building golden OS images in a declarative, cloud-native way. It provides a fast, flexible, and repeatable workflow for creating machine images for environments like the Kubernetes Cluster API (CAPI).

---
## Status of the Project

**Alpha**: This project is currently in the early stages of development. The API is subject to change, and it is not yet recommended for production use. We welcome early testers and contributors!

---
## Core Concepts

`bib-operator` replaces traditional, script-based image building with a declarative Kubernetes API. Users define the desired state of a machine image by creating an `ImageBuild` custom resource. The operator then orchestrates the entire build process inside the cluster, from provisioning to publishing.

This approach offers several key advantages:
* **Speed**: Builds run in containers, eliminating the slow overhead of traditional VM-based (QEMU) methods.
* **Declarative**: The `ImageBuild` CR is the single source of truth for your image definition, enabling GitOps workflows.
* **Extensible**: Natively supports multiple provisioners (like Ansible) and can publish artifacts to various targets (like AWS AMIs).

---
## Comparison to Other Works

The `bib-operator` project aims to fill a specific gap in the image-building ecosystem.

#### Kubernetes SIGs `image-builder`
The existing [image-builder](https://github.com/kubernetes-sigs/image-builder) for CAPI is a powerful tool based on Ansible and Packer. However, its build process relies on launching a full virtual machine using QEMU, which is resource-intensive and slow. `bib-operator` adopts the same goal but completely replaces the execution engine with a fast, container-native workflow, significantly reducing build times.

#### `bootc-image-builder`
[bootc-image-builder](https of osbuild/bootc-image-builder) is a modern project for creating bootable OSTree-based images directly from container images. It is an excellent choice for users committed to an immutable, container-centric infrastructure. `bib-operator` offers a more traditional and flexible alternative, producing standard artifacts like `.qcow2` disks and `.tgz` root filesystems. It allows users to leverage existing Ansible playbooks without adopting the OSTree ecosystem, providing a smoother transition for teams with established configuration management.

**Why `bib-operator`?** It combines the declarative, API-driven approach of modern tools with the speed of container-native builds and the flexibility of traditional image formats and provisioners.

---
## Project Structure

The repository is organized as a standard Kubebuilder project with a dedicated `builder/` directory for the build environment.

```
bib-operator/
├── api/
│   └── v1alpha1/
│       └── imagebuild_types.go      # CRD schema definition
├── internal/
│   └── controller/
│       └── imagebuild_controller.go # Operator reconciliation logic
├── builder/
│   ├── Dockerfile                   # Dockerfile for the "builder" container
│   └── entrypoint.sh                # Build logic script
├── config/
│   ├── crd/
│   ├── manager/
│   └── rbac/
├── Makefile
└── Tiltfile                         # For local development
```