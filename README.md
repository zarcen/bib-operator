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
[bootc-image-builder](https://github.com/osbuild/bootc-image-builder) is a modern project for creating bootable OSTree-based images directly from container images. It is an excellent choice for users committed to an immutable, container-centric infrastructure. `bib-operator` offers a more traditional and flexible alternative, producing standard artifacts like `.qcow2` disks and `.tgz` root filesystems. It allows users to leverage existing Ansible playbooks without adopting the OSTree ecosystem, providing a smoother transition for teams with established configuration management.

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

---
## Local Development

Setting up a local development environment is streamlined with `kind` and `Tilt` for a fast feedback loop.

### Prerequisites
* [Go](https://golang.org/) (1.24+)
* [Docker](https://www.docker.com/) (or another container runtime like Colima)
* [kubectl](https://kubernetes.io/docs/tasks/tools/install-kubectl/)
* [Tilt](https://tilt.dev/)

### Setup

1.  **Create a local Kubernetes cluster using `kind`**:
    The project `Makefile` contains helpers for this. This command will create a new `kind` cluster ready for development.
    ```bash
    make kind-cluster
    ```

2.  **Start the development environment with Tilt**:
    This command starts the Tilt server, which builds and deploys the operator to your `kind` cluster. It will open a web UI and automatically update your running controller whenever you save a change to a Go file.
    ```bash
    tilt up
    ```

3.  **Clean up**:
    When you're finished, stop the Tilt server (`Ctrl+C`) and delete the local cluster.
    ```bash
    make kind-delete
    ```

## The Builder API

To support "Bring Your Own Image" (BYOI), the `bib-operator` defines a stable contract for how it passes build parameters to a builder container. Any container image that respects this contract can be used as a builder.

The operator passes all configuration to the builder pod via **environment variables**. A compatible builder image must be able to read its instructions from the following variables:

| Variable | Required? | Description |
| :--- | :--- | :--- |
| `BASE_IMAGE` | Yes | The source container image for the build (e.g., `ubuntu:24.04`). |
| `ARCHITECTURE` | Yes | The target architecture for the build (e.g., `amd64`, `arm64`). |
| `OUTPUT_FILENAME`| Optional | The base filename for the output artifacts (e.g., `ubuntu-2404-golden`). |
| `ANSIBLE_GIT_REPO` | Optional | The Git repository URL for the Ansible provisioner. |
| `ANSIBLE_GIT_BRANCH`| Optional | The Git branch to clone for the Ansible provisioner. |
| `ANSIBLE_PLAYBOOK` | Optional | The path to the main Ansible playbook within the Git repository. |