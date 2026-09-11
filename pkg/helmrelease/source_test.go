package helmrelease

import (
	"context"
	"testing"
	"time"

	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	kubectlscheme "k8s.io/kubectl/pkg/scheme"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSourceWithDefaults(t *testing.T) {
	testCases := []struct {
		name     string
		source   Source
		expected Source
	}{
		{
			name:   "a chart name is enough for an OCIRepository",
			source: Source{ChartName: "hello-world", Namespace: "org-giantswarm"},
			expected: Source{
				Kind:      SourceKindOCIRepository,
				Name:      "hello-world",
				Namespace: "org-giantswarm",
				ChartName: "hello-world",
				URL:       DefaultRegistryURL + "/hello-world",
				SemVer:    "*",
				Interval:  defaultSourceInterval,
			},
		},
		{
			name:   "a tag wins over the semver default and loses its v prefix",
			source: Source{ChartName: "hello-world", Namespace: "org-giantswarm", Tag: "v1.2.3"},
			expected: Source{
				Kind:      SourceKindOCIRepository,
				Name:      "hello-world",
				Namespace: "org-giantswarm",
				ChartName: "hello-world",
				URL:       DefaultRegistryURL + "/hello-world",
				Tag:       "1.2.3",
				Interval:  defaultSourceInterval,
			},
		},
		{
			name: "a HelmRepository defaults to the registry root and has no ref",
			source: Source{
				Kind:      SourceKindHelmRepository,
				ChartName: "hello-world",
				Namespace: "org-giantswarm",
			},
			expected: Source{
				Kind:      SourceKindHelmRepository,
				Name:      "hello-world",
				Namespace: "org-giantswarm",
				ChartName: "hello-world",
				URL:       DefaultRegistryURL,
				Interval:  defaultSourceInterval,
			},
		},
		{
			name: "nothing set by the caller is overwritten",
			source: Source{
				Kind:      SourceKindOCIRepository,
				Name:      "custom-name",
				Namespace: "org-giantswarm",
				ChartName: "hello-world",
				URL:       "oci://example.com/charts/hello-world",
				SemVer:    ">=1.0.0",
				Interval:  5 * time.Minute,
			},
			expected: Source{
				Kind:      SourceKindOCIRepository,
				Name:      "custom-name",
				Namespace: "org-giantswarm",
				ChartName: "hello-world",
				URL:       "oci://example.com/charts/hello-world",
				SemVer:    ">=1.0.0",
				Interval:  5 * time.Minute,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			if actual := tc.source.withDefaults(); actual != tc.expected {
				t.Errorf("expected %+v, got %+v", tc.expected, actual)
			}
		})
	}
}

func TestEnsureSourceOCIRepository(t *testing.T) {
	ctx := context.Background()
	c := newFakeClient()

	source := Source{ChartName: "hello-world", Namespace: "org-giantswarm", Tag: "1.2.3"}
	if err := EnsureSource(ctx, c, source); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repo := getOCIRepository(t, c, "hello-world", "org-giantswarm")
	if repo.Spec.URL != DefaultRegistryURL+"/hello-world" {
		t.Errorf("unexpected url: %s", repo.Spec.URL)
	}
	if repo.Spec.Reference.Tag != "1.2.3" {
		t.Errorf("unexpected tag: %s", repo.Spec.Reference.Tag)
	}

	// An existing source is left alone rather than erroring.
	if err := EnsureSource(ctx, c, Source{ChartName: "hello-world", Namespace: "org-giantswarm", Tag: "9.9.9"}); err != nil {
		t.Fatalf("unexpected error on second ensure: %v", err)
	}
	if repo := getOCIRepository(t, c, "hello-world", "org-giantswarm"); repo.Spec.Reference.Tag != "1.2.3" {
		t.Errorf("existing OCIRepository was modified: tag is %s", repo.Spec.Reference.Tag)
	}
}

func TestEnsureSourceHelmRepository(t *testing.T) {
	ctx := context.Background()
	c := newFakeClient()

	source := Source{Kind: SourceKindHelmRepository, ChartName: "hello-world", Namespace: "org-giantswarm"}
	if err := EnsureSource(ctx, c, source); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repo := &sourcev1.HelmRepository{}
	if err := c.Get(ctx, types.NamespacedName{Name: "hello-world", Namespace: "org-giantswarm"}, repo); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.Spec.Type != sourcev1.HelmRepositoryTypeOCI {
		t.Errorf("expected an OCI HelmRepository for an oci:// url, got type %q", repo.Spec.Type)
	}
}

func TestEnsureSourceUnknownKind(t *testing.T) {
	if err := EnsureSource(context.Background(), newFakeClient(), Source{Kind: "GitRepository", ChartName: "hello-world"}); err == nil {
		t.Error("expected an error for an unknown source kind")
	}
}

func TestDeleteSource(t *testing.T) {
	ctx := context.Background()
	c := newFakeClient()

	source := Source{ChartName: "hello-world", Namespace: "org-giantswarm"}
	if err := EnsureSource(ctx, c, source); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := DeleteSource(ctx, c, source); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repo := &sourcev1.OCIRepository{}
	err := c.Get(ctx, types.NamespacedName{Name: "hello-world", Namespace: "org-giantswarm"}, repo)
	if err == nil {
		t.Error("expected the OCIRepository to be gone")
	}

	// Deleting something that isn't there is not an error.
	if err := DeleteSource(ctx, c, source); err != nil {
		t.Errorf("unexpected error deleting a missing source: %v", err)
	}
}

func TestUpdateOCIRepositoryTag(t *testing.T) {
	ctx := context.Background()
	c := newFakeClient()

	if err := EnsureSource(ctx, c, Source{ChartName: "hello-world", Namespace: "org-giantswarm"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := UpdateOCIRepositoryTag(ctx, c, "hello-world", "org-giantswarm", "v1.2.4"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	repo := getOCIRepository(t, c, "hello-world", "org-giantswarm")
	if repo.Spec.Reference.Tag != "1.2.4" {
		t.Errorf("unexpected tag: %s", repo.Spec.Reference.Tag)
	}
	if repo.Spec.Reference.SemVer != "" {
		t.Errorf("expected the semver range to be cleared, got %q", repo.Spec.Reference.SemVer)
	}
}

func TestUpdateOCIRepositoryTagNotFound(t *testing.T) {
	if err := UpdateOCIRepositoryTag(context.Background(), newFakeClient(), "missing", "org-giantswarm", "1.2.3"); err == nil {
		t.Error("expected an error for a missing OCIRepository")
	}
}

func newFakeClient() cr.Client {
	return fake.NewClientBuilder().WithScheme(kubectlscheme.Scheme).Build()
}

func getOCIRepository(t *testing.T, c cr.Client, name, namespace string) *sourcev1.OCIRepository {
	t.Helper()

	repo := &sourcev1.OCIRepository{ObjectMeta: metav1.ObjectMeta{}}
	if err := c.Get(context.Background(), types.NamespacedName{Name: name, Namespace: namespace}, repo); err != nil {
		t.Fatalf("unexpected error getting OCIRepository '%s/%s': %v", namespace, name, err)
	}
	return repo
}
