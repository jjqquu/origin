package validation

import (
	"k8s.io/kubernetes/pkg/api/validation"
	"k8s.io/kubernetes/pkg/util/validation/field"

	"github.com/openshift/origin/pkg/site/api"
)

func ValidateSiteName(name string, prefix bool) (bool, string) {
	return validation.NameIsDNSSubdomain(name, prefix)
}

func ValidateSite(site *api.Site) field.ErrorList {
	allErrs := validation.ValidateObjectMeta(&site.ObjectMeta, false, ValidateSiteName, field.NewPath("metadata"))

	// Only validate spec. All status fields are optional and can be updated later.
	// address is required.
	if len(site.Spec.Address.Url) == 0 {
		allErrs = append(allErrs, field.Required(field.NewPath("spec", "address.url"), ""))
	}

	return allErrs
}

func ValidateSiteUpdate(site, oldSite *api.Site) field.ErrorList {
	allErrs := validation.ValidateObjectMetaUpdate(&site.ObjectMeta, &oldSite.ObjectMeta, field.NewPath("metadata"))

	return allErrs
}
