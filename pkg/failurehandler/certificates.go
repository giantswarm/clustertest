package failurehandler

import (
	acmev1 "github.com/cert-manager/cert-manager/pkg/apis/acme/v1"
	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest/v5"
	"github.com/giantswarm/clustertest/v5/pkg/application"
	"github.com/giantswarm/clustertest/v5/pkg/logger"
)

// CertificatesNotReady collects debug information for all cert-manager Certificate resources in the given namespace
// that are not currently Ready. For each failing Certificate it also logs associated CertificateRequests, ACME
// Orders, and ACME Challenges by following the owner reference chain.
func CertificatesNotReady(framework *clustertest.Framework, cluster *application.Cluster, namespace string) FailureHandler {
	return Wrap(func() {
		ctx, cancel := newContext()
		defer cancel()

		logger.Log("Attempting to get debug info for non-ready Certificates in namespace '%s'", namespace)

		wcClient, err := framework.WC(cluster.Name)
		if err != nil {
			logger.Log("Failed to get client for workload cluster - %v", err)
			return
		}

		certList := &certmanagerv1.CertificateList{}
		if err := wcClient.List(ctx, certList, ctrl.InNamespace(namespace)); err != nil {
			logger.Log("Failed to list Certificates - %v", err)
			return
		}

		// Pre-fetch Orders and Challenges once to avoid repeated API calls per Certificate.
		orderList := &acmev1.OrderList{}
		if err := wcClient.List(ctx, orderList, ctrl.InNamespace(namespace)); err != nil {
			logger.Log("Failed to list Orders - %v", err)
		}

		challengeList := &acmev1.ChallengeList{}
		if err := wcClient.List(ctx, challengeList, ctrl.InNamespace(namespace)); err != nil {
			logger.Log("Failed to list Challenges - %v", err)
		}

		for i := range certList.Items {
			cert := certList.Items[i]
			if isCertReady(&cert) {
				continue
			}

			logger.Log("Certificate '%s/%s' is not ready:", cert.Namespace, cert.Name)
			for _, condition := range cert.Status.Conditions {
				logger.Log("  Condition: Type=%s, Status=%s, Reason=%s, Message=%s",
					condition.Type, condition.Status, condition.Reason, condition.Message)
			}
			if cert.Status.LastFailureTime != nil {
				logger.Log("  LastFailureTime: %s", cert.Status.LastFailureTime)
			}
			if cert.Status.FailedIssuanceAttempts != nil {
				logger.Log("  FailedIssuanceAttempts: %d", *cert.Status.FailedIssuanceAttempts)
			}

			events, err := wcClient.GetEventsForResource(ctx, &cert)
			if err != nil {
				logger.Log("  Failed to get events for Certificate '%s' - %v", cert.Name, err)
			} else {
				for _, event := range events.Items {
					logger.Log("  Event: Reason='%s', Message='%s', Last Occurred='%v'",
						event.Reason, event.Message, event.LastTimestamp)
				}
			}

			crList := &certmanagerv1.CertificateRequestList{}
			if err := wcClient.List(ctx, crList,
				ctrl.InNamespace(namespace),
				ctrl.MatchingLabels{"cert-manager.io/certificate-name": cert.Name},
			); err != nil {
				logger.Log("  Failed to list CertificateRequests for Certificate '%s' - %v", cert.Name, err)
				continue
			}

			for j := range crList.Items {
				cr := crList.Items[j]
				logger.Log("  CertificateRequest '%s':", cr.Name)
				for _, condition := range cr.Status.Conditions {
					logger.Log("    Condition: Type=%s, Status=%s, Reason=%s, Message=%s",
						condition.Type, condition.Status, condition.Reason, condition.Message)
				}
				if cr.Status.FailureTime != nil {
					logger.Log("    FailureTime: %s", cr.Status.FailureTime)
				}

				for k := range orderList.Items {
					order := orderList.Items[k]
					if !hasOwnerNamed(order.OwnerReferences, cr.Name) {
						continue
					}
					logger.Log("    Order '%s': State='%s', Reason='%s'",
						order.Name, order.Status.State, order.Status.Reason)
					if order.Status.FailureTime != nil {
						logger.Log("      FailureTime: %s", order.Status.FailureTime)
					}

					for l := range challengeList.Items {
						challenge := challengeList.Items[l]
						if !hasOwnerNamed(challenge.OwnerReferences, order.Name) {
							continue
						}
						logger.Log("      Challenge '%s': State='%s', Reason='%s', Processing=%v, Presented=%v",
							challenge.Name, challenge.Status.State, challenge.Status.Reason,
							challenge.Status.Processing, challenge.Status.Presented)
					}
				}
			}
		}
	})
}

func isCertReady(cert *certmanagerv1.Certificate) bool {
	for _, condition := range cert.Status.Conditions {
		if condition.Type == certmanagerv1.CertificateConditionReady && condition.Status == cmmeta.ConditionTrue {
			return true
		}
	}
	return false
}

func hasOwnerNamed(refs []metav1.OwnerReference, name string) bool {
	for _, ref := range refs {
		if ref.Name == name {
			return true
		}
	}
	return false
}
