package helmrelease

import (
	"context"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest/v4/pkg/client"
	"github.com/giantswarm/clustertest/v4/pkg/logger"
	"github.com/giantswarm/clustertest/v4/pkg/wait"
)

// IsHelmReleaseReady returns a WaitCondition that polls the named HelmRelease and
// becomes true when its Ready condition is True.
func IsHelmReleaseReady(ctx context.Context, c cr.Client, name, namespace string) wait.WaitCondition {
	return isHelmReleaseReady(ctx, c, types.NamespacedName{Name: name, Namespace: namespace})
}

// IsAppOrHelmReleaseReady returns a WaitCondition that becomes true as soon as
// either an App CR or a HelmRelease with the given name/namespace reaches a Ready
// state. Use this for default apps that may be deployed as either kind depending on
// cluster chart version.
//
// Get errors (including NotFound) are suppressed so the outer Eventually keeps
// polling until one of the two kinds appears and is Ready, or the timeout fires.
func IsAppOrHelmReleaseReady(ctx context.Context, c *client.Client, name, namespace string) wait.WaitCondition {
	appCheck := wait.IsAppDeployed(ctx, c, name, namespace)
	hrCheck := isHelmReleaseReady(ctx, c, types.NamespacedName{Name: name, Namespace: namespace})

	return func() (bool, error) {
		if ok, _ := appCheck(); ok {
			return true, nil
		}
		if ok, _ := hrCheck(); ok {
			return true, nil
		}
		return false, nil
	}
}

func isHelmReleaseReady(ctx context.Context, c cr.Client, name types.NamespacedName) wait.WaitCondition {
	return func() (bool, error) {
		hr := &helmv2.HelmRelease{}
		if err := c.Get(ctx, name, hr); err != nil {
			return false, err
		}

		condition := apimeta.FindStatusCondition(hr.Status.Conditions, "Ready")
		if condition == nil {
			logger.Log("HelmRelease '%s' has no Ready condition yet", name.Name)
			return false, nil
		}

		if condition.Status == metav1.ConditionTrue {
			logger.Log("HelmRelease '%s' is Ready", name.Name)
			return true, nil
		}

		logger.Log("HelmRelease '%s' not yet ready: %s - %s", name.Name, condition.Reason, condition.Message)
		return false, nil
	}
}
