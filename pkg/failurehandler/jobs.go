package failurehandler

import (
	batchv1 "k8s.io/api/batch/v1"

	"github.com/giantswarm/clustertest"
	"github.com/giantswarm/clustertest/pkg/application"
	"github.com/giantswarm/clustertest/pkg/logger"
)

// JobsUnsuccessful collects debug information for all Jobs in the workload cluster that haven't completed
// successfully. This information includes events for the Jobs and the status of all their conditions
func JobsUnsuccessful(framework *clustertest.Framework, cluster *application.Cluster) FailureHandler {
	return Wrap(func() {
		ctx, cancel := newContext()
		defer cancel()

		logger.Log("Attempting to get debug info for failed Jobs")

		wcClient, err := framework.WC(cluster.Name)
		if err != nil {
			logger.Log("Failed to get client for workload cluster - %v", err)
			return
		}

		jobList := &batchv1.JobList{}
		err = wcClient.List(ctx, jobList)
		if err != nil {
			logger.Log("Failed to get list of deployments")
			return
		}

		for i := range jobList.Items {
			job := jobList.Items[i]
			if job.Status.Succeeded == 0 && job.Status.Active == 0 {
				logger.Log("Job %s/%s has not succeeded. (Number Failed: '%d')", job.Namespace, job.Name, job.Status.Failed)
				for _, condition := range job.Status.Conditions {
					logger.Log("Job '%s' condition: Type='%s', Status='%s', Message='%s'", job.Name, condition.Type, condition.Status, condition.Message)
				}

				{
					// Events
					events, err := wcClient.GetEventsForResource(ctx, &job)
					if err != nil {
						logger.Log("Failed to get events for Job '%s' - %v", job.Name, err)
					} else {
						for _, event := range events.Items {
							logger.Log("Job '%s' Event: Reason='%s', Message='%s', Last Occurred='%v'", job.Name, event.Reason, event.Message, event.LastTimestamp)
						}
					}
				}
			}
		}
	})
}
