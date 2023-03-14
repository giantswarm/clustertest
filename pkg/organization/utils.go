package organization

import (
	orgv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
)

// SafeToDelete checks if the Org CR contains an annotation specific to E2E testing
func SafeToDelete(orgCR orgv1alpha1.Organization) bool {
	_, ok := orgCR.GetAnnotations()[DeleteAnnotation]
	return ok
}
