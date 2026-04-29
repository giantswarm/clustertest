package helmrelease

import (
	"context"
	"fmt"
	"time"

	sourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
)

const ociRegistryURL = "oci://gsoci.azurecr.io/charts/giantswarm"

// EnsureOCIRepository creates an OCIRepository pointing at the Giant Swarm OCI
// registry for the given chart, or is a no-op if it already exists.
func EnsureOCIRepository(ctx context.Context, c cr.Client, name, namespace, chartName string) error {
	repo := &sourcev1beta2.OCIRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: sourcev1beta2.OCIRepositorySpec{
			URL:      fmt.Sprintf("%s/%s", ociRegistryURL, chartName),
			Interval: metav1.Duration{Duration: 1 * time.Minute},
			Reference: &sourcev1beta2.OCIRepositoryRef{
				SemVer: "*",
			},
		},
	}

	err := c.Create(ctx, repo)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating OCIRepository: %w", err)
	}
	return nil
}

// DeleteOCIRepository deletes the named OCIRepository, or is a no-op if it does not exist.
func DeleteOCIRepository(ctx context.Context, c cr.Client, name, namespace string) error {
	repo := &sourcev1beta2.OCIRepository{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := c.Delete(ctx, repo)
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
