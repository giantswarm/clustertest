package testuser

import (
	"bytes"
	"context"
	"encoding/base64"
	"html/template"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	cr "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/giantswarm/clustertest/pkg/client"
	"github.com/giantswarm/clustertest/pkg/logger"
	"github.com/giantswarm/clustertest/pkg/wait"
)

// Create handles the creation of a ServiceAccount with cluster-admin permission within the cluster
// and generated a new Kubernetes client that authenticates as that account.
func Create(ctx context.Context, kubeClient *client.Client) (*client.Client, error) {
	// default namespace is created by controllers after a while
	err := waitForNamespace(ctx, kubeClient)
	if err != nil {
		return nil, err
	}

	existing, err := doesUserExist(ctx, kubeClient)
	if err != nil {
		return nil, err
	}

	if !existing {
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
	}

	var ca string
	var token string

	err = wait.For(
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

func waitForNamespace(ctx context.Context, kubeClient *client.Client) error {
	err := wait.For(
		func() (bool, error) {
			var namespace corev1.Namespace
			err := kubeClient.Get(ctx, types.NamespacedName{Name: serviceAccount.Namespace}, &namespace)
			if err != nil {
				logger.Log("Waiting for %s namespace. Error: %v", serviceAccount.Namespace, err)
				return false, err
			}

			logger.Log("Namespace %s exists", serviceAccount.Namespace)
			return true, nil
		},
		wait.WithTimeout(5*time.Minute),
		wait.WithInterval(5*time.Second),
	)
	return err
}

func doesUserExist(ctx context.Context, kubeClient *client.Client) (bool, error) {
	var existingAccount corev1.ServiceAccount
	err := kubeClient.Get(ctx, cr.ObjectKeyFromObject(&serviceAccount), &existingAccount)
	if err != nil && !errors.IsNotFound(err) {
		return false, err
	} else if errors.IsNotFound(err) {
		return false, nil
	}
	return true, nil
}
