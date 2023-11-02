package testuser

import (
	"bytes"
	"context"
	"encoding/base64"
	"html/template"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/wait"
)

// Create handles the creation of a ServiceAccount with cluster-admin permission within the cluster
// and generated a new Kubernetes client that authenticates as that account.
func Create(ctx context.Context, kubeClient *client.Client) (*client.Client, error) {
	// ServiceAccount
	if err := kubeClient.Create(ctx, &serviceAccount); err != nil {
		return nil, err
	}
	// Secret
	if err := kubeClient.Create(ctx, &secret); err != nil {
		return nil, err
	}
	// ClusterRoleBinding
	if err := kubeClient.Create(ctx, &clusterRoleBinding); err != nil {
		return nil, err
	}

	var ca string
	var token string

	err := wait.For(
		func() (bool, error) {
			var populatedSecret corev1.Secret
			err := kubeClient.Get(ctx, types.NamespacedName{Name: secret.ObjectMeta.Name, Namespace: secret.ObjectMeta.Namespace}, &populatedSecret)
			if err != nil {
				return false, err
			}

			ca = base64.StdEncoding.EncodeToString(populatedSecret.Data["ca.crt"])
			token = string(populatedSecret.Data["token"])

			return (ca != "" && token != ""), nil
		},
		wait.WithTimeout(5*time.Minute),
		wait.WithInterval(1*time.Second),
	)
	if err != nil {
		return nil, err
	}

	t := template.Must(template.New("kubeconfig").Parse(kubeConfigTemplate))
	var buf bytes.Buffer
	err = t.Execute(&buf, templateVars{
		Endpoint:    kubeClient.GetAPIServerEndpoint(),
		AccountName: accountName,
		CA:          ca,
		Token:       token,
	})
	if err != nil {
		return nil, err
	}

	return client.NewFromRawKubeconfig(buf.String())
}
