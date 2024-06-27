package client

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetEventsForResources returns all existing events related to the provided resource
func (c *Client) GetEventsForResource(ctx context.Context, resource cr.Object, extraFieldSelectors ...fields.Selector) (*corev1.EventList, error) {
	events := &corev1.EventList{}

	fieldSelectors := append(extraFieldSelectors, fields.OneTermEqualSelector("involvedObject.name", resource.GetName()))

	if resource.GetNamespace() != "" {
		fieldSelectors = append(fieldSelectors, fields.OneTermEqualSelector("involvedObject.namespace", resource.GetNamespace()))
	}

	// Get the Object kind from the schema
	gvks, unversioned, err := c.Scheme().ObjectKinds(resource)
	if err != nil {
		return events, err
	}
	if !unversioned && len(gvks) == 1 {
		fieldSelectors = append(fieldSelectors, fields.OneTermEqualSelector("involvedObject.kind", gvks[0].Kind))
	}

	err = c.List(ctx, events, cr.MatchingFieldsSelector{
		Selector: fields.AndSelectors(fieldSelectors...),
	})

	return events, err
}

// GetNormalEventsForResource returns all events related to the provided resource that have a type of "Normal"
func (c *Client) GetNormalEventsForResource(ctx context.Context, resource cr.Object) (*corev1.EventList, error) {
	return c.GetEventsForResource(ctx, resource, fields.OneTermEqualSelector("type", corev1.EventTypeNormal))
}

// GetWarningEventsForResource returns all events related to the provided resource that have a type of "Warning"
func (c *Client) GetWarningEventsForResource(ctx context.Context, resource cr.Object) (*corev1.EventList, error) {
	return c.GetEventsForResource(ctx, resource, fields.OneTermEqualSelector("type", corev1.EventTypeWarning))
}
