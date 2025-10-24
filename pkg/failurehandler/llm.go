package failurehandler

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest/v2"
	"github.com/giantswarm/clustertest/v2/pkg/application"
	"github.com/giantswarm/clustertest/v2/pkg/logger"
	"github.com/giantswarm/clustertest/v2/pkg/wait"
)

const (
	// llmJobTimeout is the maximum time to wait for the LLM job to complete
	llmJobTimeout = 15 * time.Minute
	// llmJobImage is the container image to use for the LLM job
	llmJobImage = "gsoci.azurecr.io/giantswarm/shoot:latest"
	// llmOtelEndpoint is the OpenTelemetry endpoint for the LLM job
	llmOtelEndpoint = "http://otlp-gateway.kube-system.svc.cluster.local:4318"
)

// LLMPrompt creates a Kubernetes Job that uses an LLM to investigate issues in the cluster.
// The Job runs with a timeout and is automatically cleaned up after completion via Kubernetes TTL controller.
// The query parameter specifies what the LLM should investigate.
//
// Example usage:
//
//	failurehandler.LLMPrompt(framework, cluster, "Investigate why the HelmReleases are not Ready")
//	failurehandler.LLMPrompt(framework, cluster, "Investigate why the Apps are not Ready")
//	failurehandler.LLMPrompt(framework, cluster, "Investigate pods with several restarts")
func LLMPrompt(framework *clustertest.Framework, cluster *application.Cluster, query string) FailureHandler {
	return Wrap(func() {
		ctx, cancel := context.WithTimeout(context.Background(), llmJobTimeout+time.Minute)
		defer cancel()

		mcClient := framework.MC()
		namespace := cluster.Organization.GetNamespace()

		// Generate a unique job name using the cluster name
		jobName := generateJobName(cluster.Name)
		logger.Log("Creating LLM investigation Job '%s' in namespace '%s'", jobName, namespace)

		// Create the Job
		job := createLLMJob(jobName, namespace, cluster.Name, query)
		err := mcClient.Create(ctx, job)
		if err != nil {
			logger.Log("Failed to create LLM investigation Job - %v", err)
			return
		}

		// Wait for the Job to complete
		logger.Log("Waiting for LLM investigation Job to complete (timeout: %v)", llmJobTimeout)
		waitErr := wait.For(
			isJobCompleted(ctx, mcClient, jobName, namespace),
			wait.WithTimeout(llmJobTimeout),
		)

		if waitErr != nil {
			logger.Log("LLM investigation Job did not complete in time or failed - %v", waitErr)
		}

		// Get and log the Job's logs
		pods := &corev1.PodList{}
		err = mcClient.List(ctx, pods,
			ctrl.InNamespace(namespace),
			ctrl.MatchingLabels{"job-name": jobName},
		)
		if err != nil {
			logger.Log("Failed to get pods for LLM investigation Job - %v", err)
			return
		}

		if len(pods.Items) == 0 {
			logger.Log("No pods found for LLM investigation Job")
			return
		}

		// Log output from the pod (there should only be one)
		pod := &pods.Items[0]
		logger.Log("Getting logs from LLM investigation Job pod '%s'", pod.Name)

		logs, err := mcClient.GetLogs(ctx, pod, nil)
		if err != nil {
			logger.Log("Failed to get logs from LLM investigation Job pod '%s' - %v", pod.Name, err)
			return
		}

		// Log the output with clear markers for readability
		logger.Log("==================== LLM INVESTIGATION RESULTS START ====================")
		logger.Log("Query: %s", query)
		logger.Log("=========================================================================")

		// Split logs into lines and log each separately for better readability
		logLines := strings.Split(logs, "\n")
		for _, line := range logLines {
			if line != "" {
				logger.Log("%s", line)
			}
		}

		logger.Log("==================== LLM INVESTIGATION RESULTS END ========================")
	})
}

// isJobCompleted returns a WaitCondition that checks if a Job has completed (either successfully or with failure)
func isJobCompleted(ctx context.Context, client ctrl.Client, jobName, namespace string) wait.WaitCondition {
	return func() (bool, error) {
		job := &batchv1.Job{}
		err := client.Get(ctx, ctrl.ObjectKey{Name: jobName, Namespace: namespace}, job)
		if err != nil {
			logger.Log("Failed to get Job status - %v", err)
			return false, err
		}

		// Check if Job has completed successfully
		if job.Status.Succeeded > 0 {
			logger.Log("LLM investigation Job completed successfully")
			return true, nil
		}

		// Check if Job has failed
		if job.Status.Failed > 0 {
			logger.Log("LLM investigation Job failed (Failed count: %d)", job.Status.Failed)
			return true, nil
		}

		// Check if Job is still active
		if job.Status.Active > 0 {
			logger.Log("LLM investigation Job is still running (Active count: %d)", job.Status.Active)
			return false, nil
		}

		// Job hasn't started yet
		logger.Log("LLM investigation Job hasn't started yet")
		return false, nil
	}
}

// generateJobName creates a unique job name using the cluster name and a random suffix.
// The random suffix ensures uniqueness when multiple tests run on the same cluster.
func generateJobName(clusterName string) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	const randomLength = 6

	// Generate random suffix
	randomSuffix := make([]byte, randomLength)
	for i := range randomSuffix {
		randomSuffix[i] = charset[rand.Intn(len(charset))]
	}

	return fmt.Sprintf("llm-investigate-%s-%s", clusterName, string(randomSuffix))
}

// createLLMJob creates the Kubernetes Job manifest for running the LLM investigation
func createLLMJob(jobName, namespace, clusterName, query string) *batchv1.Job {
	backoffLimit := int32(0)
	ttlSecondsAfterFinished := int32(300) // Auto-cleanup 5 minutes after completion
	runAsUser := int64(10001)
	runAsGroup := int64(10001)
	fsGroup := int64(10001)
	runAsNonRoot := true
	allowPrivilegeEscalation := false

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
			Labels: map[string]string{
				"giantswarm.io/managed-by": "cluster-test-suites",
			},
		},
		Spec: batchv1.JobSpec{
			BackoffLimit:            &backoffLimit,
			TTLSecondsAfterFinished: &ttlSecondsAfterFinished,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"observability.giantswarm.io/tenant": "giantswarm",
					},
				},
				Spec: corev1.PodSpec{
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &runAsNonRoot,
						RunAsUser:    &runAsUser,
						RunAsGroup:   &runAsGroup,
						FSGroup:      &fsGroup,
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:            "shoot",
							Image:           llmJobImage,
							ImagePullPolicy: corev1.PullAlways,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
							},
							Env: []corev1.EnvVar{
								{
									Name:  "QUERY",
									Value: query,
								},
								{
									Name: "OPENAI_API_KEY",
									ValueFrom: &corev1.EnvVarSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: "openai-api-key",
											},
											Key: "OPENAI_API_KEY",
										},
									},
								},
								{
									Name:  "OTEL_EXPORTER_OTLP_ENDPOINT",
									Value: llmOtelEndpoint,
								},
								{
									Name:  "OTEL_RESOURCE_ATTRIBUTES",
									Value: "service.name=shoot",
								},
								{
									Name:  "KUBECONFIG",
									Value: "/home/app/k8s/kubeconfig.yaml",
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "kubeconfig",
									MountPath: "/home/app/k8s",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "kubeconfig",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: fmt.Sprintf("%s-kubeconfig", clusterName),
									Items: []corev1.KeyToPath{
										{
											Key:  "value",
											Path: "kubeconfig.yaml",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}
