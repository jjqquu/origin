package validation

import (
	"testing"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util/validation/field"

	"github.com/openshift/origin/pkg/site/api"
)

func TestValidateSite(t *testing.T) {
	successCases := []api.Site{
		{
			ObjectMeta: api.ObjectMeta{Name: "site-s"},
			Spec: api.SiteSpec{
				Address: api.SiteAddress{
					Url: "http://localhost:8888",
				},
			},
		},
	}
	for _, successCase := range successCases {
		errs := ValidateSite(&successCase)
		if len(errs) != 0 {
			t.Errorf("expect success: %v", errs)
		}
	}

	errorCases := map[string]api.Site{
		"missing site address": {
			ObjectMeta: api.ObjectMeta{Name: "site-f"},
		},
		"invalid_label": {
			ObjectMeta: api.ObjectMeta{
				Name: "site-f",
				Labels: map[string]string{
					"NoUppercaseOrSpecialCharsLike=Equals": "bar",
				},
			},
		},
	}
	for testName, errorCase := range errorCases {
		errs := ValidateSite(&errorCase)
		if len(errs) == 0 {
			t.Errorf("expected failur for %s", testName)
		}
	}
}

func TestValidateSiteUpdate(t *testing.T) {
	type siteUpdateTest struct {
		old    api.Site
		update api.Site
	}
	successCases := []siteUpdateTest{
		{
			old: api.Site{
				ObjectMeta: api.ObjectMeta{Name: "site-s"},
				Spec: api.SiteSpec{
					Address: api.SiteAddress{
						Url: "http://localhost:8888",
					},
				},
			},
			update: api.Site{
				ObjectMeta: api.ObjectMeta{Name: "site-s"},
				Spec: api.SiteSpec{
					Address: api.SiteAddress{
						Url: "http://127.0.0.1:8888",
					},
				},
			},
		},
	}
	for _, successCase := range successCases {
		successCase.old.ObjectMeta.ResourceVersion = "1"
		successCase.update.ObjectMeta.ResourceVersion = "1"
		errs := ValidateSiteUpdate(&successCase.update, &successCase.old)
		if len(errs) != 0 {
			t.Errorf("expect success: %v", errs)
		}
	}

	errorCases := map[string]siteUpdateTest{}
	for testName, errorCase := range errorCases {
		errs := ValidateSiteUpdate(&errorCase.update, &errorCase.old)
		if len(errs) == 0 {
			t.Errorf("expected failure: %s", testName)
		}
	}
}
