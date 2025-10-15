# -*- mode: Python -*-

# --- Configuration ---
IMG_NAME = 'bib-operator-controller-manager:dev'
# The name of local kind cluster (should match what it is in the Makefile)
KIND_CLUSTER_NAME = 'kind-bib-operator'
# ---------------------

# --- Local Helper Functions ---

# Helper for logging messages
info = print

def load_images(images):
    """Load images into the kind cluster.
    Args:
        images: A list of image tags to load.
    """
    if not images:
        info("No images to load, skipping.")
        return

    info("Loading {} image(s) into kind cluster '{}'...".format(len(images), KIND_CLUSTER_NAME))
    for image in images:
        cmd = "kind load docker-image --name {} {}".format(KIND_CLUSTER_NAME, image)
        local(cmd)
    info("Image loading complete.")

# ---------------------------------

# Use allow_k8s_contexts as a safety rail to ensure Tilt only
# deploys to your specified kind cluster.
allow_k8s_contexts('kind-{}'.format(KIND_CLUSTER_NAME))

# Define how to build the operator's container image.
# We capture the name of the newly built image in the 'built_image' variable.
docker_build(
    IMG_NAME,
    '.',
    live_update=[
        sync('./', '/workspace'),
        run(
            'go install -v ./cmd/main.go',
            trigger=['./cmd/**/*.go', './api/**/*.go', './internal/**/*.go']
        )
    ]
)

info('Built image: {}'.format(IMG_NAME))
load_images([IMG_NAME])

local_resource(
    'Install CRDs',
    cmd='make install',
    deps=['api/v1alpha1/imagebuild_types.go']
)

local_resource(
    'deploy-operator',
    cmd='make deploy IMG={}'.format(IMG_NAME),
    deps=['internal', 'api', 'cmd']
)


info('Tilt is ready! Your dev environment is live at http://localhost:10350/')