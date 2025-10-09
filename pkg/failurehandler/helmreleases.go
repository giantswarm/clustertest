package failurehandler

import (
	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest/v2"
	"github.com/giantswarm/clustertest/v2/pkg/application"
	"github.com/giantswarm/clustertest/v2/pkg/logger"
)

// HelmReleasesNotReady collects debug information for all HelmReleases in the organization namespace in the
// management cluster that currently are not ready.
func HelmReleasesNotReady(framework *clustertest.Framework, cluster *application.Cluster) FailureHandler {
	return Wrap(func() {
		ctx, cancel := newContext()
		defer cancel()

		logger.Log("Gathering HelmRelease status information for debugging")

		helmReleaseList := &helmv2.HelmReleaseList{}
		err := framework.MC().List(ctx, helmReleaseList, ctrl.InNamespace(cluster.Organization.GetNamespace()))
		if err != nil {
			logger.Log("Failed to get HelmReleases - %v", err)
			return
		}

		for _, hr := range helmReleaseList.Items {
			ready := false
			for _, condition := range hr.Status.Conditions {
				if condition.Type == "Ready" && condition.Status == metav1.ConditionTrue {
					ready = true
					break
				}
			}

			if !ready {
				logger.Log("HelmRelease '%s/%s' is not ready:", hr.Namespace, hr.Name)
				for _, condition := range hr.Status.Conditions {
					logger.Log("  Condition: Type=%s, Status=%s, Reason=%s, Message=%s",
						condition.Type, condition.Status, condition.Reason, condition.Message)
				}

				// Log recent events for this HelmRelease
				events := &corev1.EventList{}
				err := framework.MC().List(ctx, events, ctrl.InNamespace(hr.Namespace),
					ctrl.MatchingFields{"involvedObject.name": hr.Name})
				if err == nil {
					logger.Log("  Recent events:")
					for _, event := range events.Items {
						if event.InvolvedObject.Kind == "HelmRelease" {
							logger.Log("    %s: %s", event.Reason, event.Message)
						}
					}
				}
			}
		}
	})
}
