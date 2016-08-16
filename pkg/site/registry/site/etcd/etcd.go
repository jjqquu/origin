package etcd

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/registry/generic/registry"
	"k8s.io/kubernetes/pkg/runtime"

	siteapi "github.com/openshift/origin/pkg/site/api"
	"github.com/openshift/origin/pkg/site/registry/site"
	"github.com/openshift/origin/pkg/util/restoptions"
)

type REST struct {
	*registry.Store
}

type StatusREST struct {
	store *registry.Store
}

func (r *StatusREST) New() runtime.Object {
	return &siteapi.Site{}
}

// Update alters the status subset of an object.
func (r *StatusREST) Update(ctx api.Context, name string, objInfo rest.UpdatedObjectInfo) (runtime.Object, bool, error) {
	return r.store.Update(ctx, name, objInfo)
}

// NewREST returns a RESTStorage object that will work against sites.
func NewREST(optsGetter restoptions.Getter) (*REST, *StatusREST, error) {
	prefix := "/sites"

	store := &registry.Store{
		NewFunc:     func() runtime.Object { return &siteapi.Site{} },
		NewListFunc: func() runtime.Object { return &siteapi.SiteList{} },
		KeyRootFunc: func(ctx api.Context) string {
			return registry.NamespaceKeyRootFunc(ctx, prefix)
		},
		KeyFunc: func(ctx api.Context, name string) (string, error) {
			return registry.NamespaceKeyFunc(ctx, prefix, name)
		},
		ObjectNameFunc: func(obj runtime.Object) (string, error) {
			return obj.(*siteapi.Site).Name, nil
		},
		PredicateFunc:     site.MatchSite,
		QualifiedResource: api.Resource("sites"),

		CreateStrategy: site.Strategy,
		UpdateStrategy: site.Strategy,
	}

	if err := restoptions.ApplyOptions(optsGetter, store, prefix); err != nil {
		return nil, nil, err
	}

	statusStore := *store
	statusStore.UpdateStrategy = site.StatusStrategy

	return &REST{store}, &StatusREST{&statusStore}, nil
}
