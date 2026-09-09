package helmrelease

import (
	"testing"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	sourcev1beta2 "github.com/fluxcd/source-controller/api/v1beta2"
	"k8s.io/apimachinery/pkg/runtime"
	kubectlscheme "k8s.io/kubectl/pkg/scheme"
)

// TestSchemeRegistration asserts that the package init registers every Flux type the
// helpers here and the clustertest client operate on. kubectlscheme.Scheme is the scheme
// handed to controller-runtime in pkg/client, so a missing registration surfaces at
// runtime as "no kind is registered for the type" on the first Get/Create.
func TestSchemeRegistration(t *testing.T) {
	testCases := []struct {
		name string
		obj  runtime.Object
	}{
		{name: "HelmRelease helm.toolkit.fluxcd.io/v2", obj: &helmv2.HelmRelease{}},
		{name: "OCIRepository source.toolkit.fluxcd.io/v1", obj: &sourcev1.OCIRepository{}},
		{name: "HelmRepository source.toolkit.fluxcd.io/v1", obj: &sourcev1.HelmRepository{}},
		{name: "OCIRepository source.toolkit.fluxcd.io/v1beta2", obj: &sourcev1beta2.OCIRepository{}},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			gvks, _, err := kubectlscheme.Scheme.ObjectKinds(tc.obj)
			if err != nil {
				t.Fatalf("expected %T to be registered in the scheme: %v", tc.obj, err)
			}
			if len(gvks) == 0 {
				t.Fatalf("expected at least one GroupVersionKind for %T", tc.obj)
			}
		})
	}
}
