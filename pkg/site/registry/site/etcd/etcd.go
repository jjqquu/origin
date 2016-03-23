package etcd

import (
	"k8s.io/kubernetes/pkg/api"
	etcdgeneric "k8s.io/kubernetes/pkg/registry/generic/etcd"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/storage"

	siteapi "github.com/openshift/origin/pkg/site/api"
	"github.com/openshift/origin/pkg/site/registry/site"
)

type REST struct {
	*etcdgeneric.Etcd
}

type StatusREST struct {
	store *etcdgeneric.Etcd
}

func (r *StatusREST) New() runtime.Object {
	return &siteapi.Site{}
}

// Update alters the status subset of an object.
func (r *StatusREST) Update(ctx api.Context, obj runtime.Object) (runtime.Object, bool, error) {
	return r.store.Update(ctx, obj)
}

// NewREST returns a RESTStorage object that will work against sites.
func NewREST(s storage.Interface) (*REST, *StatusREST) {
	prefix := "/sites"

	store := &etcdgeneric.Etcd{
		NewFunc:     func() runtime.Object { return &siteapi.Site{} },
		NewListFunc: func() runtime.Object { return &siteapi.SiteList{} },
		KeyRootFunc: func(ctx api.Context) string {
			return prefix
		},
		KeyFunc: func(ctx api.Context, name string) (string, error) {
			return etcdgeneric.NoNamespaceKeyFunc(ctx, prefix, name)
		},
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*siteapi.Site).Name, nil
		},
		PredicateFunc:     site.MatchSite,
		QualifiedResource: api.Resource("sites"),

		CreateStrategy: site.Strategy,
		UpdateStrategy: site.Strategy,

		Storage: s,
	}

	statusStore := *store
	statusStore.UpdateStrategy = site.StatusStrategy

	return &REST{store}, &StatusREST{store: &statusStore}
}
