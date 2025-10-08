package organization

import (
	"fmt"
	"strings"

	templateorg "github.com/giantswarm/kubectl-gs/v2/pkg/template/organization"
	orgv1alpha1 "github.com/giantswarm/organization-operator/api/v1alpha1"
	"k8s.io/client-go/kubernetes/scheme"

	"github.com/giantswarm/clustertest/v2/pkg/utils"
)

func init() {
	_ = orgv1alpha1.AddToScheme(scheme.Scheme)
}

// Org contains details about an Organization
type Org struct {
	Name string

	namespace string
}

// NewRandomOrg returns an Org with a randomly generated name
func NewRandomOrg() *Org {
	return New(utils.GenerateRandomName("t"))
}

// New returns a new Org with the provided name
func New(name string) *Org {
	return &Org{
		Name:      name,
		namespace: fmt.Sprintf("org-%s", name),
	}
}

// NewFromNamespace returns a new Org, taking the name from the passed namespace.
func NewFromNamespace(namespace string) *Org {
	return &Org{
		Name:      strings.TrimPrefix(namespace, "org-"),
		namespace: namespace,
	}
}

// GetNamespace returns the associated namespace for the Organization
func (o *Org) GetNamespace() string {
	return o.namespace
}

// Build generates the Organization CR for applying to the cluster
func (o *Org) Build() (*orgv1alpha1.Organization, error) {
	orgCR, err := templateorg.NewOrganizationCR(templateorg.Config{
		Name: o.Name,
	})
	if err != nil {
		return nil, err
	}

	// We want to add an annotation to track if we should be removing the org when done or not
	// This check will allow us to re-use existing orgs too without accidentally deleting the org when done
	orgCR.Annotations = map[string]string{
		utils.DeleteAnnotation: "true",
	}

	// If found, populate details about Tekton run as labels
	orgCR.Labels = utils.GetBaseLabels()

	orgCR.Status.Namespace = o.GetNamespace()

	return orgCR, err
}
