package testuser

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	accountName = "e2e-test-account"
	namespace   = metav1.NamespaceDefault
)

var serviceAccount = corev1.ServiceAccount{
	ObjectMeta: metav1.ObjectMeta{
		Name:      accountName,
		Namespace: namespace,
	},
}

var secret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      fmt.Sprintf("%s-secret", accountName),
		Namespace: namespace,
		Annotations: map[string]string{
			"kubernetes.io/service-account.name": accountName,
		},
	},
	Type: corev1.SecretTypeServiceAccountToken,
}

var clusterRoleBinding = rbacv1.ClusterRoleBinding{
	ObjectMeta: metav1.ObjectMeta{
		Name: accountName,
	},
	Subjects: []rbacv1.Subject{
		{
			Kind:      rbacv1.ServiceAccountKind,
			Name:      accountName,
			Namespace: namespace,
		},
	},
	RoleRef: rbacv1.RoleRef{
		APIGroup: "rbac.authorization.k8s.io",
		Kind:     "ClusterRole",
		Name:     "cluster-admin",
	},
}
