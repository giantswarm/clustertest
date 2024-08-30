package utils

import "os"

// GetBaseLabels returns a map of labels based on specific available environment variables being found
func GetBaseLabels() map[string]string {
	baseLabels := map[string]string{}

	// If found, populate details about Tekton run as labels
	if os.Getenv("TEKTON_PIPELINE_RUN") != "" {
		baseLabels["cicd.giantswarm.io/pipelinerun"] = os.Getenv("TEKTON_PIPELINE_RUN")
	}
	if os.Getenv("TEKTON_TASK_RUN") != "" {
		baseLabels["cicd.giantswarm.io/taskrun"] = os.Getenv("TEKTON_TASK_RUN")
	}
	if os.Getenv("CICD_PR_NUMBER") != "" {
		baseLabels["cicd.giantswarm.io/pr"] = os.Getenv("CICD_PR_NUMBER")
	}
	if os.Getenv("CICD_TRIGGER_USER") != "" {
		baseLabels["cicd.giantswarm.io/triggered-by"] = os.Getenv("CICD_TRIGGER_USER")
	}
	if os.Getenv("CICD_REPO") != "" {
		baseLabels["cicd.giantswarm.io/repo"] = os.Getenv("CICD_REPO")
	}

	return baseLabels
}
