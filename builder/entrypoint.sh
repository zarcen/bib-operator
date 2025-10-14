#!/bin/sh
set -ex

echo "--- Starting image build ---"
echo "Base Image: ${BASE_IMAGE}"
echo "Architecture: ${ARCHITECTURE}"

# --- Authentication Setup (for pulling the base image) ---
AUTH_FILE="/etc/baseimage-pull-secret/.dockerconfigjson"

# Clone the provisioning repository
# The git-sync init container will handle this in the final version.
# For now, we'll do it here if the repo is specified.
if [ -n "$GIT_REPO" ]; then
    echo "Cloning repository ${GIT_REPO}..."
    git clone --branch "${GIT_BRANCH}" "${GIT_REPO}" /source
fi

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

# Run Ansible provisioner if a playbook is specified
if [ -n "$PLAYBOOK" ]; then
    echo "Running Ansible playbook ${PLAYBOOK}..."
    # The --connection=chroot tells Ansible to run against the mounted filesystem
    ansible-playbook --connection=chroot --inventory="${mount_path}," "/source/${PLAYBOOK}"
fi

# Unmount, create tarball, and clean up
echo "Creating TGZ archive at /output/${OUTPUT_FILENAME}.tgz"
buildah umount "$container"
# We re-mount to ensure all changes are flushed to the filesystem before tarring.
buildah mount "$container"
tar -czf "/output/${OUTPUT_FILENAME}.tgz" -C "$mount_path" .
buildah umount "$container"
buildah rm "$container"

echo "--- Build complete! ---"