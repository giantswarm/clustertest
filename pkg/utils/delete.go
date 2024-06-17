package utils

const (
	// DeleteAnnotation is added to Organizations created during testing.
	// This is to ensure only those with this annotation can be deleted to avoid accidentally deleting a shared Org.
	DeleteAnnotation = "e2e-test-cleanup"
)

// SafeToDelete checks if the provided annotations contains an annotation specific to E2E testing
func SafeToDelete(annotations map[string]string) bool {
	_, ok := annotations[DeleteAnnotation]
	return ok
}
