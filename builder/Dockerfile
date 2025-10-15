FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive

# Install all dependencies.
RUN apt-get update && \
    apt-get install -y --no-install-recommends \
        # Core dependencies
        ca-certificates \
        curl \
        git \
        python3 \
        python3-pip \
        # Main build tools
        ansible-core \
        ansible \
        buildah \
        qemu-utils \
        libguestfs-tools \
        jq \
        tar \
    && \
    # Install Python-based tools
    pip install --break-system-packages awscli \
    && \
    # Clean up
    apt-get clean && \
    rm -rf /var/lib/apt/lists/*

# Copy our new entrypoint script and make it executable
COPY entrypoint.sh /workspace/entrypoint.sh
RUN chmod +x /workspace/entrypoint.sh

# Set the entrypoint for the container
ENTRYPOINT ["/workspace/entrypoint.sh"]

WORKDIR /workspace
CMD ["/bin/bash"]