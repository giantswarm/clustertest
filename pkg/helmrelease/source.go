package helmrelease

import (
	"context"
	"fmt"
	"strings"
	"time"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest/v5/pkg/logger"
)

// SourceKind is the kind of Flux source CR a HelmRelease pulls its chart from.
type SourceKind string

const (
	// SourceKindOCIRepository is referenced from a HelmRelease through spec.chartRef.
	SourceKindOCIRepository SourceKind = "OCIRepository"
	// SourceKindHelmRepository is referenced from a HelmRelease through spec.chart.spec.sourceRef.
	SourceKindHelmRepository SourceKind = "HelmRepository"
)

// DefaultRegistryURL is the Giant Swarm OCI registry the published Helm charts live in.
const DefaultRegistryURL = "oci://gsoci.azurecr.io/charts/giantswarm"

// defaultSourceInterval is how often a source re-checks the registry when the caller
// doesn't ask for something else.
const defaultSourceInterval = 1 * time.Minute

// Source describes the Flux source CR a HelmRelease pulls its chart from. The zero value
// of every field has a Giant Swarm default, so a chart published to the Giant Swarm
// registry only needs ChartName and Namespace.
type Source struct {
	// Kind of source CR to create. Defaults to [SourceKindOCIRepository].
	Kind SourceKind
	// Name of the source CR. Defaults to ChartName.
	Name string
	// Namespace of the source CR.
	Namespace string
	// ChartName is the name of the chart in the registry. Used to build the default URL
	// and to default Name.
	ChartName string
	// URL of the registry. Defaults to [DefaultRegistryURL], suffixed with the chart name
	// for an OCIRepository. An "oci://" URL makes a HelmRepository an OCI one.
	URL string
	// Tag pins the exact chart version to pull. OCIRepository only. A leading "v" is
	// stripped, as Giant Swarm charts are tagged without it.
	Tag string
	// SemVer is the range of versions to select the latest from. OCIRepository only, and
	// only consulted when Tag is empty. Defaults to "*".
	SemVer string
	// Interval between reconciliations. Defaults to one minute.
	Interval time.Duration
}

// withDefaults returns a copy of the Source with every unset field filled in.
func (s Source) withDefaults() Source {
	if s.Kind == "" {
		s.Kind = SourceKindOCIRepository
	}
	if s.Name == "" {
		s.Name = s.ChartName
	}
	if s.Interval == 0 {
		s.Interval = defaultSourceInterval
	}

	s.Tag = strings.TrimPrefix(s.Tag, "v")
	if s.Kind == SourceKindOCIRepository && s.Tag == "" && s.SemVer == "" {
		s.SemVer = "*"
	}

	if s.URL == "" {
		s.URL = DefaultRegistryURL
		if s.Kind == SourceKindOCIRepository {
			s.URL = fmt.Sprintf("%s/%s", DefaultRegistryURL, s.ChartName)
		}
	}

	return s
}

// EnsureSource creates the source CR the given Source describes, or is a no-op if it
// already exists. An existing source is left untouched, so use [UpdateOCIRepositoryTag]
// to move an OCIRepository to a different chart version.
func EnsureSource(ctx context.Context, c cr.Client, source Source) error {
	s := source.withDefaults()

	var obj cr.Object
	switch s.Kind {
	case SourceKindOCIRepository:
		obj = &sourcev1.OCIRepository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      s.Name,
				Namespace: s.Namespace,
			},
			Spec: sourcev1.OCIRepositorySpec{
				URL:      s.URL,
				Interval: metav1.Duration{Duration: s.Interval},
				Reference: &sourcev1.OCIRepositoryRef{
					Tag:    s.Tag,
					SemVer: s.SemVer,
				},
			},
		}
	case SourceKindHelmRepository:
		repoType := sourcev1.HelmRepositoryTypeDefault
		if strings.HasPrefix(s.URL, "oci://") {
			repoType = sourcev1.HelmRepositoryTypeOCI
		}
		obj = &sourcev1.HelmRepository{
			ObjectMeta: metav1.ObjectMeta{
				Name:      s.Name,
				Namespace: s.Namespace,
			},
			Spec: sourcev1.HelmRepositorySpec{
				Type:     repoType,
				URL:      s.URL,
				Interval: metav1.Duration{Duration: s.Interval},
			},
		}
	default:
		return fmt.Errorf("unknown source kind %q", s.Kind)
	}

	logger.Log("Ensuring %s '%s/%s' (url: %s)", s.Kind, s.Namespace, s.Name, s.URL)

	err := c.Create(ctx, obj)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("creating %s '%s/%s': %w", s.Kind, s.Namespace, s.Name, err)
	}
	return nil
}

// DeleteSource deletes the source CR the given Source describes, or is a no-op if it does
// not exist. Only Kind, Name (or ChartName) and Namespace are read.
func DeleteSource(ctx context.Context, c cr.Client, source Source) error {
	s := source.withDefaults()

	var obj cr.Object
	switch s.Kind {
	case SourceKindOCIRepository:
		obj = &sourcev1.OCIRepository{ObjectMeta: metav1.ObjectMeta{Name: s.Name, Namespace: s.Namespace}}
	case SourceKindHelmRepository:
		obj = &sourcev1.HelmRepository{ObjectMeta: metav1.ObjectMeta{Name: s.Name, Namespace: s.Namespace}}
	default:
		return fmt.Errorf("unknown source kind %q", s.Kind)
	}

	logger.Log("Deleting %s '%s/%s'", s.Kind, s.Namespace, s.Name)

	err := c.Delete(ctx, obj)
	if err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("deleting %s '%s/%s': %w", s.Kind, s.Namespace, s.Name, err)
	}
	return nil
}

// UpdateOCIRepositoryTag points an existing OCIRepository at a different chart version.
// This is how an upgrade is triggered for a HelmRelease that pulls through spec.chartRef:
// the version lives on the source, not on the HelmRelease. A leading "v" is stripped from
// the tag, and any semver range is cleared so the tag is what takes effect.
func UpdateOCIRepositoryTag(ctx context.Context, c cr.Client, name, namespace, tag string) error {
	tag = strings.TrimPrefix(tag, "v")

	repo := &sourcev1.OCIRepository{}
	if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, repo); err != nil {
		return fmt.Errorf("getting OCIRepository '%s/%s': %w", namespace, name, err)
	}

	if repo.Spec.Reference == nil {
		repo.Spec.Reference = &sourcev1.OCIRepositoryRef{}
	}
	repo.Spec.Reference.Tag = tag
	repo.Spec.Reference.SemVer = ""

	logger.Log("Updating OCIRepository '%s/%s' to tag '%s'", namespace, name, tag)

	if err := c.Update(ctx, repo); err != nil {
		return fmt.Errorf("updating OCIRepository '%s/%s': %w", namespace, name, err)
	}
	return nil
}

// EnsureOCIRepository creates an OCIRepository pointing at the Giant Swarm OCI
// registry for the given chart, or is a no-op if it already exists.
//
// Deprecated: use [EnsureSource], which also covers HelmRepository sources and pinning a
// chart version.
func EnsureOCIRepository(ctx context.Context, c cr.Client, name, namespace, chartName string) error {
	return EnsureSource(ctx, c, Source{
		Kind:      SourceKindOCIRepository,
		Name:      name,
		Namespace: namespace,
		ChartName: chartName,
	})
}

// DeleteOCIRepository deletes the named OCIRepository, or is a no-op if it does not exist.
//
// Deprecated: use [DeleteSource], which also covers HelmRepository sources.
func DeleteOCIRepository(ctx context.Context, c cr.Client, name, namespace string) error {
	return DeleteSource(ctx, c, Source{
		Kind:      SourceKindOCIRepository,
		Name:      name,
		Namespace: namespace,
	})
}
