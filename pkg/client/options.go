package client

import (
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
	cr "sigs.k8s.io/controller-runtime/pkg/client"
)

type DoesNotHaveLabels []string

func (m DoesNotHaveLabels) ApplyToList(opts *cr.ListOptions) {
	selector := labels.NewSelector()
	for _, label := range m {
		req, err := labels.NewRequirement(label, selection.DoesNotExist, nil)
		if err == nil {
			selector = selector.Add(*req)
		}
	}
	opts.LabelSelector = selector
}

func (m DoesNotHaveLabels) ApplyToDeleteAllOf(opts *cr.DeleteAllOfOptions) {
	m.ApplyToList(&opts.ListOptions)
}
