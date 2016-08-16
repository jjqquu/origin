package api

import "k8s.io/kubernetes/pkg/fields"

// SiteToSelectableFields returns a label set that represents the object
// changes to the returned keys require registering conversions for existing versions using Scheme.AddFieldLabelConversionFunc
func SiteToSelectableFields(site *Site) fields.Set {
	return fields.Set{
		"metadata.name": site.Name,
	}
}
