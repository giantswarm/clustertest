package client

import "k8s.io/apimachinery/pkg/api/errors"

// isUnsuccessfulConnectionError are errors returned from the api-server that are likely due to still being setup
func isUnsuccessfulConnectionError(err error) bool {
	return errors.IsServiceUnavailable(err) || errors.IsTimeout(err) ||
		errors.IsServerTimeout(err) || errors.IsUnexpectedServerError(err)
}

// isSuccessfulConnectionError are errors returned from an api-server when a cluster is finished setting up.
// These could be things like resource not found or permissions issues.
func isSuccessfulConnectionError(err error) bool {
	if _, ok := err.(errors.APIStatus); ok && !isUnsuccessfulConnectionError(err) {
		return true
	}

	return false
}
