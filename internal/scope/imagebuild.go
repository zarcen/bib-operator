package scope

import (
	"context"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"sigs.k8s.io/cluster-api/util/conditions"
	"sigs.k8s.io/cluster-api/util/patch"
	"sigs.k8s.io/controller-runtime/pkg/client"

	bibv1alpha1 "github.com/zarcen/bib-operator/api/v1alpha1"
)

// ImageBuildScope defines the scope of an ImageBuild resource.
type ImageBuildScope struct {
	client.Client
	patchHelper *patch.Helper
	Logger      logr.Logger

	ImageBuild *bibv1alpha1.ImageBuild
}

func NewImageBuildScope(client client.Client, logger logr.Logger, ib *bibv1alpha1.ImageBuild) (*ImageBuildScope, error) {
	if client == nil {
		return nil, errors.New("invalid arguments: client is nil")
	}
	if ib == nil {
		return nil, errors.New("invalid arguments: imageBuild is nil")
	}

	helper, err := patch.NewHelper(ib, client)
	if err != nil {
		return nil, errors.Errorf("failed to initialize the patch helper: %v", err)
	}

	return &ImageBuildScope{
		Client:      client,
		patchHelper: helper,
		Logger:      logger,
		ImageBuild:  ib,
	}, nil
}

func (s *ImageBuildScope) Close(ctx context.Context) error {
	return s.PatchObject(ctx)
}

// PatchObject persists the machine spec and status.
func (s *ImageBuildScope) PatchObject(ctx context.Context) error {
	return s.patchHelper.Patch(
		ctx,
		s.ImageBuild)
}

func (s *ImageBuildScope) InitializeConditions() {
	// Set conditions to be Unknown for all conditions that are not yet set.
	for _, conditionType := range bibv1alpha1.ImageBuildConditionTypes {
		if !conditions.Has(s.ImageBuild, conditionType) {
			conditions.MarkUnknown(s.ImageBuild, conditionType, "Initializing", "Unknown")
		}
	}
}
