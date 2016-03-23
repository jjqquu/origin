/*
Copyright 2016 The Kubernetes Authors All rights reserved.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
    Unless required by applicable law or agreed to in writing, software
    distributed under the License is distributed on an "AS IS" BASIS,
    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
    See the License for the specific language governing permissions and
    limitations under the License.
*/

package site

import (
	"testing"

	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/testapi"
	apitesting "k8s.io/kubernetes/pkg/api/testing"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"reflect"

	siteapi "github.com/openshift/origin/pkg/site/api"
)

func validNewSite() *api.Site {
	return &api.Site{
		ObjectMeta: api.ObjectMeta{
			Name:            "foo",
			ResourceVersion: "4",
			Labels: map[string]string{
				"name": "foo",
			},
		},
		Spec: api.SiteSpec{
			Address: api.SiteAddress{
				Url: "http://localhost:8888",
			},
		},
		Status: api.SiteStatus{
			Phase: api.SiteTerminated,
		},
	}
}

func invalidNewSite() *api.Site {
	return &api.Site{
		ObjectMeta: api.ObjectMeta{
			Name:            "foo",
			ResourceVersion: "5",
		},
		Spec: api.SiteSpec{
			Credential: "bar",
		},
		Status: api.SiteStatus{
			Phase: api.SiteOffline,
		},
	}
}

func TestSiteStrategy(t *testing.T) {
	ctx := api.NewDefaultContext()
	if Strategy.NamespaceScoped() {
		t.Errorf("Site should not be namespace scoped")
	}
	if Strategy.AllowCreateOnUpdate() {
		t.Errorf("Site should not allow create on update")
	}

	site := validNewSite()
	Strategy.PrepareForCreate(site)
	if site.Status.Phase != api.SitePending {
		t.Errorf("Site should not allow setting phase on create")
	}
	errs := Strategy.Validate(ctx, site)
	if len(errs) != 0 {
		t.Errorf("Unexpected error validating %v", errs)
	}

	invalidSite := invalidNewSite()
	Strategy.PrepareForUpdate(invalidSite, site)
	if reflect.DeepEqual(invalidSite.Spec, site.Spec) ||
		!reflect.DeepEqual(invalidSite.Status, site.Status) {
		t.Error("Only spec is expected being changed")
	}
	errs = Strategy.ValidateUpdate(ctx, invalidSite, site)
	if len(errs) == 0 {
		t.Errorf("Expected a validation error")
	}
	if site.ResourceVersion != "4" {
		t.Errorf("Incoming resource version on update should not be mutated")
	}
}

func TestSiteStatusStrategy(t *testing.T) {
	ctx := api.NewDefaultContext()
	if StatusStrategy.NamespaceScoped() {
		t.Errorf("Site should not be namespace scoped")
	}
	if StatusStrategy.AllowCreateOnUpdate() {
		t.Errorf("Site should not allow create on update")
	}

	site := validNewSite()
	invalidSite := invalidNewSite()
	StatusStrategy.PrepareForUpdate(site, invalidSite)
	if !reflect.DeepEqual(invalidSite.Spec, site.Spec) ||
		reflect.DeepEqual(invalidSite.Status, site.Status) {
		t.Error("Only spec is expected being changed")
	}
	errs := Strategy.ValidateUpdate(ctx, invalidSite, site)
	if len(errs) == 0 {
		t.Errorf("Expected a validation error")
	}
	if site.ResourceVersion != "4" {
		t.Errorf("Incoming resource version on update should not be mutated")
	}
}

func TestMatchSite(t *testing.T) {
	testFieldMap := map[bool][]fields.Set{
		true: {
			{"metadata.name": "foo"},
		},
		false: {
			{"foo": "bar"},
		},
	}

	for expectedResult, fieldSet := range testFieldMap {
		for _, field := range fieldSet {
			m := MatchSite(labels.Everything(), field.AsSelector())
			_, matchesSingle := m.MatchesSingle()
			if e, a := expectedResult, matchesSingle; e != a {
				t.Errorf("%+v: expected %v, got %v", fieldSet, e, a)
			}
		}
	}
}

func TestSelectableFieldLabelConversions(t *testing.T) {
	apitesting.TestSelectableFieldLabelConversionsOfKind(t,
		testapi.Controlplane.GroupVersion().String(),
		"Site",
		labels.Set(SiteToSelectableFields(&api.Site{})),
		nil,
	)
}
