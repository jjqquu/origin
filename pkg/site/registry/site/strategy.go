package site

import (
	"fmt"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/registry/generic"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/validation/field"

	"github.com/openshift/origin/pkg/site/api"
	"github.com/openshift/origin/pkg/site/api/validation"
)

type siteStrategy struct {
	runtime.ObjectTyper
	kapi.NameGenerator
}

// Strategy is the default logic that applies when creating and updating Site objects.
var Strategy = siteStrategy{kapi.Scheme, kapi.SimpleNameGenerator}

// NamespaceScoped returns false because all Site objects do NOT need to be within a namespace.
func (siteStrategy) NamespaceScoped() bool {
	return false
}

func MatchSite(label labels.Selector, field fields.Selector) generic.Matcher {
	return &generic.SelectionPredicate{
		Label: label,
		Field: field,
		GetAttrs: func(obj runtime.Object) (labels.Set, fields.Set, error) {
			site, ok := obj.(*api.Site)
			if !ok {
				return nil, nil, fmt.Errorf("given object is not a site.")
			}
			return labels.Set(site.ObjectMeta.Labels), api.SiteToSelectableFields(site), nil
		},
	}
}

// PrepareForCreate clears fields that are not allowed to be set by end users on creation.
func (siteStrategy) PrepareForCreate(obj runtime.Object) {
	site := obj.(*api.Site)
	site.Status.Phase = api.SitePending
}

// Validate validates a new site.
func (siteStrategy) Validate(ctx kapi.Context, obj runtime.Object) field.ErrorList {
	site := obj.(*api.Site)
	return validation.ValidateSite(site)
}

// Canonicalize normalizes the object after validation.
func (siteStrategy) Canonicalize(obj runtime.Object) {
}

// AllowCreateOnUpdate is false for site; this means a POST is
// needed to create one.
func (siteStrategy) AllowCreateOnUpdate() bool {
	return false
}

// PrepareForUpdate clears fields that are not allowed to be set by end users on update.
func (siteStrategy) PrepareForUpdate(obj, old runtime.Object) {
	site := obj.(*api.Site)
	oldSite := old.(*api.Site)
	// update is not allowed to set status, but reset phase as pending
	site.Status = oldSite.Status
	site.Status.Phase = api.SitePending
}

// ValidateUpdate is the default update validation for an end user.
func (siteStrategy) ValidateUpdate(ctx kapi.Context, obj, old runtime.Object) field.ErrorList {
	allErrs := validation.ValidateSite(obj.(*api.Site))
	return append(allErrs, validation.ValidateSiteUpdate(obj.(*api.Site), old.(*api.Site))...)
}
func (siteStrategy) AllowUnconditionalUpdate() bool {
	return true
}

type siteStatusStrategy struct {
	siteStrategy
}

var StatusStrategy = siteStatusStrategy{Strategy}

func (siteStatusStrategy) PrepareForUpdate(obj, old runtime.Object) {
	site := obj.(*api.Site)
	oldSite := old.(*api.Site)
	// update is not allowed to set spec
	site.Spec = oldSite.Spec
}

// ValidateUpdate is the default update validation for an end user.
func (siteStatusStrategy) ValidateUpdate(ctx kapi.Context, obj, old runtime.Object) field.ErrorList {
	return validation.ValidateSiteUpdate(obj.(*api.Site), old.(*api.Site))
}
