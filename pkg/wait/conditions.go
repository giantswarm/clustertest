package wait

import (
	"context"
	"fmt"

	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
)

// WaitCondition is a function performing a condition check for if we need to keep waiting
type WaitCondition func() (done bool, err error)

// IsClusterReadyCondition returns a WaitCondition to check when a cluster is considered ready and accessible
func IsClusterReadyCondition(ctx context.Context, kubeClient *client.Client, clusterName string, namespace string, clientMap map[string]*client.Client) WaitCondition {
	return func() (bool, error) {
		logger.Log("Checking for valid Kubeconfig for cluster %s", clusterName)

		var kubeconfigSecret corev1.Secret
		err := kubeClient.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-kubeconfig", clusterName), Namespace: namespace}, &kubeconfigSecret)
		if cr.IgnoreNotFound(err) != nil {
			return false, err
		} else if apierrors.IsNotFound(err) {
			// Kubeconfig not yet available
			logger.Log("kubeconfig secret not yet available")
			return false, nil
		}

		if len(kubeconfigSecret.Data["value"]) == 0 {
			// Kubeconfig data not yet available
			logger.Log("kubeconfig secret not yet populated")
			return false, nil
		}

		kubeconfig := string(kubeconfigSecret.Data["value"])
		wcClient, err := client.NewFromRawKubeconfig(string(kubeconfig))
		if err != nil {
			return false, err
		}

		if err := wcClient.CheckConnection(); err != nil {
			// Cluster not yet ready
			logger.Log("connection to api-server not yet available")
			return false, nil
		}

		logger.Log("Got valid kubeconfig!")

		// Store client for later
		clientMap[clusterName] = wcClient

		return true, nil
	}
}

// IsResourceDeleted returns a WaitCondition that checks if the given resource has been deleted from the cluster yet
func IsResourceDeleted(ctx context.Context, kubeClient *client.Client, resource cr.Object) WaitCondition {
	return func() (bool, error) {
		logger.Log("Checking if resource '%s' still exists", resource.GetName())
		err := kubeClient.Client.Get(ctx, cr.ObjectKeyFromObject(resource), resource, &cr.GetOptions{})
		if cr.IgnoreNotFound(err) != nil {
			return false, err
		} else if apierrors.IsNotFound(err) {
			return true, nil
		}

		return false, nil
	}
}

// DoesResourceExist returns a WaitCondition that checks if the given resource exists in the cluster
func DoesResourceExist(ctx context.Context, kubeClient *client.Client, resource cr.Object) WaitCondition {
	return func() (bool, error) {
		if err := kubeClient.Client.Get(ctx, cr.ObjectKeyFromObject(resource), resource); err != nil {
			logger.Log("Waiting for resource '%s' to be created", resource.GetName())
			return false, nil
		}

		return true, nil
	}
}

// IsNumNodesReady returns a WaitCondition that checks if the number of ready nodes matching the given labels equals or exceeds the expectedNodes value
func IsNumNodesReady(ctx context.Context, kubeClient *client.Client, expectedNodes int, labels *cr.MatchingLabels) WaitCondition {
	return func() (bool, error) {
		logger.Log("Checking for ready control plane nodes")

		nodes := &corev1.NodeList{}
		err := kubeClient.List(ctx, nodes, labels)
		if err != nil {
			return false, err
		}

		readyNodes := 0

		for _, node := range nodes.Items {
			for _, condition := range node.Status.Conditions {
				if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
					readyNodes += 1
				}
			}
		}

		logger.Log("%d nodes ready, expecting %d", readyNodes, expectedNodes)

		if readyNodes < expectedNodes {
			return false, nil
		}

		return true, nil
	}
}
