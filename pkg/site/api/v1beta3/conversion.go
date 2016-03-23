package v1beta3

import (
	"k8s.io/kubernetes/pkg/runtime"

	oapi "github.com/openshift/origin/pkg/api"
	siteapi "github.com/openshift/origin/pkg/site/api"
)

func addConversionFuncs(scheme *runtime.Scheme) {
	err := scheme.AddDefaultingFuncs(
		func(obj *SiteSpec) {
			if len(obj.Type) == 0 {
				obj.Type = "local"
			}
		},
	)
	if err != nil {
		panic(err)
	}

	err = scheme.AddConversionFuncs()
	if err != nil {
		panic(err)
	}

	if err := scheme.AddFieldLabelConversionFunc("v1beta3", "Site",
		oapi.GetFieldLabelConversionFunc(siteapi.SiteToSelectableFields(&siteapi.Site{}), nil),
	); err != nil {
		panic(err)
	}
}
