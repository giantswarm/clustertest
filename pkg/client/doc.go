// package client provides a thin wrapper around the controller-runtime client.
//
// It provides standard operations (such as Get, Patch, Delete) for interacting with a Kubernetes cluster as well some helper
// functions for making it easier to perform certain operations.
//
// Note: The client when created is set to use lazy discovery and doesn't pre-cache CRDs from the cluster.
// This is to allow for creation of a Client instance for a Workload Cluster before the api-server is ready by using the
// kubeconfig available from the Management Cluster secret.
//
// For the full list of available functions make sure to also check [cr.Client] for the controller-runtime methods.
package client
