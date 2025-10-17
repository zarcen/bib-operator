#!/bin/sh
set -ex

# --- Builder API Contract ---
# This script is a consumer of the bib-operator Builder API. It receives its
# configuration from the following environment variables:
#
# - BASE_IMAGE:           The source container image for the build.
# - ARCHITECTURE:         The target architecture (e.g., amd64).
# - OUTPUT_FILENAME:      (Optional) The base filename for the output artifacts.
# - ANSIBLE_GIT_REPO:     (Optional) The Git repo for the Ansible provisioner.
# - ANSIBLE_GIT_BRANCH:   (Optional) The Git branch to clone.
# - ANSIBLE_PLAYBOOK:     (Optional) The path to the Ansible playbook.
# -----------------------------

echo "--- Starting image build ---"
echo "Base Image: ${BASE_IMAGE}"
echo "Architecture: ${ARCHITECTURE}"

# --- Authentication Setup (for pulling the base image) ---
AUTH_FILE="/etc/baseimage-pull-secret/.dockerconfigjson"

# Create a working container from the base image
if [ -f "$AUTH_FILE" ]; then
    echo "Auth file found, using it for buildah."
    container=$(buildah from --authfile "${AUTH_FILE}" --arch "${ARCHITECTURE}" "${BASE_IMAGE}")
else
    echo "No auth file found, proceeding without authentication."
    container=$(buildah from --arch "${ARCHITECTURE}" "${BASE_IMAGE}")
fi
echo "Created container: $container"

# Mount the container's filesystem
mount_path=$(buildah mount "$container")
echo "Container mounted at: $mount_path"

echo "Preparing chroot environment with device nodes..."
mount --bind /dev "${mount_path}/dev"

# Clone the provisioning repository
# The git-sync init container will handle this in the final version.
# For now, we'll do it here if the repo is specified.
if [ -n "$ANSIBLE_GIT_REPO" ]; then
    echo "Cloning repository ${ANSIBLE_GIT_REPO}..."
    git clone --branch "${ANSIBLE_GIT_BRANCH}" "${ANSIBLE_GIT_REPO}" /source
fi

# Run Ansible provisioner if a playbook is specified
if [ -n "$ANSIBLE_PLAYBOOK" ]; then
    echo "Running Ansible playbook ${ANSIBLE_PLAYBOOK}..."
    # The --connection=chroot tells Ansible to run against the mounted filesystem
    ansible-playbook --connection=chroot --inventory="${mount_path}," "/source/${ANSIBLE_PLAYBOOK}"
fi

echo "Cleaning up chroot environment..."
umount "${mount_path}/dev"

# Unmount, create tarball, and clean up
echo "Creating TGZ archive at /output/${OUTPUT_FILENAME}.tgz"
buildah umount "$container"
# We re-mount to ensure all changes are flushed to the filesystem before tarring.
buildah mount "$container"
tar -czf "/output/${OUTPUT_FILENAME}.tgz" -C "$mount_path" .
buildah umount "$container"
buildah rm "$container"

echo "--- Build complete! ---"