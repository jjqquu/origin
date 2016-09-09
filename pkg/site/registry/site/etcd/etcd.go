package etcd

import (
	"k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/rest"
	"k8s.io/kubernetes/pkg/capabilities"
	"k8s.io/kubernetes/pkg/registry/generic/registry"
	genericrest "k8s.io/kubernetes/pkg/registry/generic/rest"
	"k8s.io/kubernetes/pkg/runtime"

	"fmt"
	siteapi "github.com/openshift/origin/pkg/site/api"
	"github.com/openshift/origin/pkg/site/registry/site"
	"github.com/openshift/origin/pkg/util/restoptions"

	"net/http"
	"net/url"
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
func NewREST(optsGetter restoptions.Getter, proxyTransport http.RoundTripper, insecureProxyTransport http.RoundTripper) (*REST, *StatusREST, *ProxyREST, error) {
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
		return nil, nil, nil, err
	}

	statusStore := *store
	statusStore.UpdateStrategy = site.StatusStrategy

	return &REST{store}, &StatusREST{&statusStore}, &ProxyREST{store: store, proxyTransport: proxyTransport, insecureProxyTransport: insecureProxyTransport}, nil
}

// ProxyREST implements the proxy subresource for a Pod
// TODO: move me into pod/rest - I'm generic to store type via ResourceGetter
type ProxyREST struct {
	store                  *registry.Store
	proxyTransport         http.RoundTripper
	insecureProxyTransport http.RoundTripper
}

// Implement Connecter
var _ = rest.Connecter(&ProxyREST{})

var proxyMethods = []string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"}

// New returns an empty pod resource
func (r *ProxyREST) New() runtime.Object {
	return &siteapi.Site{}
}

// ConnectMethods returns the list of HTTP methods that can be proxied
func (r *ProxyREST) ConnectMethods() []string {
	return proxyMethods
}

// NewConnectOptions returns versioned resource that represents proxy parameters
func (r *ProxyREST) NewConnectOptions() (runtime.Object, bool, string) {
	return nil, true, ""
}

// Connect returns a handler for the site agent proxy
func (r *ProxyREST) Connect(ctx api.Context, id string, opts runtime.Object, responder rest.Responder) (http.Handler, error) {
	location, err := getSiteAgentLocation(r.store, ctx, id)
	if err != nil {
		return nil, err
	}
	location.Path = "proxy"

	// Return a proxy handler that uses the desired transport,
	// wrapped with additional proxy handling (to get URL rewriting, X-Forwarded-* headers, etc)
	// upgrade is NOT required
	handler := genericrest.NewUpgradeAwareProxyHandler(location, r.insecureProxyTransport, true, false, responder)
	handler.MaxBytesPerSec = capabilities.Get().PerConnectionBandwidthLimitBytesPerSec
	return handler, nil
}

func getSiteAgentLocation(siteStore *registry.Store, ctx api.Context, id string) (*url.URL, error) {
	obj, err := siteStore.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	site := obj.(*siteapi.Site)
	if site == nil {
		return nil, fmt.Errorf("Unexpected object type: %#v", site)
	}
	siteAgentLoc := &url.URL{
		Scheme: "http",
		Host:   site.Status.SiteAgentAddress,
	}
	return siteAgentLoc, nil
}
