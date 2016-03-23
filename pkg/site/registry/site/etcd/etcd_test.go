package etcd

import (
	"testing"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/fields"
	"k8s.io/kubernetes/pkg/labels"
	"k8s.io/kubernetes/pkg/registry/generic"
	"k8s.io/kubernetes/pkg/registry/registrytest"
	"k8s.io/kubernetes/pkg/runtime"
	etcdtesting "k8s.io/kubernetes/pkg/storage/etcd/testing"

	"github.com/openshift/origin/pkg/site/api"
	_ "github.com/openshift/origin/pkg/site/api/install"
	"github.com/openshift/origin/pkg/site/registry/site"
)

func newStorage(t *testing.T) (*REST, *etcdtesting.EtcdTestServer) {
	etcdStorage, server := registrytest.NewEtcdStorage(t, api.GroupName)
	restOptions := generic.RESTOptions{etcdStorage, generic.UndecoratedStorage, 1}
	storage, _ := NewREST(restOptions)
	return storage, server
}

func validNewSite() *api.Site {
	return &api.Site{
		ObjectMeta: api.ObjectMeta{
			Name: "foo",
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
			Phase: api.SitePending,
		},
	}
}

func TestCreate(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	test := registrytest.New(t, storage.Etcd).SiteScope()
	site := validNewSite()
	site.ObjectMeta = api.ObjectMeta{GenerateName: "foo"}
	test.TestCreate(
		site,
		&api.Site{
			ObjectMeta: api.ObjectMeta{Name: "-a123-a_"},
		},
	)
}

func TestUpdate(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	test := registrytest.New(t, storage.Etcd).SiteScope()
	test.TestUpdate(
		// valid
		validNewSite(),
		// updateFunc
		func(obj runtime.Object) runtime.Object {
			object := obj.(*api.Site)
			object.Spec.Credential = "bar"
			return object
		},
	)
}

func TestDelete(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	test := registrytest.New(t, storage.Etcd).SiteScope().ReturnDeletedObject()
	test.TestDelete(validNewSite())
}

func TestGet(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	test := registrytest.New(t, storage.Etcd).SiteScope()
	test.TestGet(validNewSite())
}

func TestList(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	test := registrytest.New(t, storage.Etcd).SiteScope()
	test.TestList(validNewSite())
}

func TestWatch(t *testing.T) {
	storage, server := newStorage(t)
	defer server.Terminate(t)
	test := registrytest.New(t, storage.Etcd).SiteScope()
	test.TestWatch(
		validNewSite(),
		// matching labels
		[]labels.Set{
			{"name": "foo"},
		},
		// not matching labels
		[]labels.Set{
			{"name": "bar"},
			{"foo": "bar"},
		},
		// matching fields
		[]fields.Set{
			{"metadata.name": "foo"},
		},
		// not matchin fields
		[]fields.Set{
			{"metadata.name": "bar"},
		},
	)
}
