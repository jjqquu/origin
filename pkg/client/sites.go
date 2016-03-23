package client

import (
	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/watch"

	siteapi "github.com/openshift/origin/pkg/site/api"
)

// SitesNamespacer has methods to work with Site resources in a namespace
type SitesNamespacer interface {
	Sites(namespace string) SiteInterface
}

// SiteInterface exposes methods on Site resources
type SiteInterface interface {
	List(opts kapi.ListOptions) (*siteapi.SiteList, error)
	Get(name string) (*siteapi.Site, error)
	Create(site *siteapi.Site) (*siteapi.Site, error)
	Update(site *siteapi.Site) (*siteapi.Site, error)
	UpdateStatus(site *siteapi.Site) (*siteapi.Site, error)
	Delete(name string) error
	Watch(opts kapi.ListOptions) (watch.Interface, error)
}

// sites implements SiteInterface interface
type sites struct {
	r  *Client
	ns string
}

// newSites returns a sites
func newSites(c *Client, namespace string) *sites {
	return &sites{
		r:  c,
		ns: namespace,
	}
}

// List takes a label and field selector, and returns the list of sites that match that selectors
func (c *sites) List(opts kapi.ListOptions) (result *siteapi.SiteList, err error) {
	result = &siteapi.SiteList{}
	err = c.r.Get().
		Namespace(c.ns).
		Resource("sites").
		VersionedParams(&opts, kapi.ParameterCodec).
		Do().
		Into(result)
	return
}

// Get takes the name of the site, and returns the corresponding Site object, and an error if it occurs
func (c *sites) Get(name string) (result *siteapi.Site, err error) {
	result = &siteapi.Site{}
	err = c.r.Get().Namespace(c.ns).Resource("sites").Name(name).Do().Into(result)
	return
}

// Delete takes the name of the site, and returns an error if one occurs
func (c *sites) Delete(name string) error {
	return c.r.Delete().Namespace(c.ns).Resource("sites").Name(name).Do().Error()
}

// Create takes the representation of a site.  Returns the server's representation of the site, and an error, if it occurs
func (c *sites) Create(site *siteapi.Site) (result *siteapi.Site, err error) {
	result = &siteapi.Site{}
	err = c.r.Post().Namespace(c.ns).Resource("sites").Body(site).Do().Into(result)
	return
}

// Update takes the representation of a site to update.  Returns the server's representation of the site, and an error, if it occurs
func (c *sites) Update(site *siteapi.Site) (result *siteapi.Site, err error) {
	result = &siteapi.Site{}
	err = c.r.Put().Namespace(c.ns).Resource("sites").Name(site.Name).Body(site).Do().Into(result)
	return
}

// UpdateStatus takes the site with altered status.  Returns the server's representation of the site, and an error, if it occurs.
func (c *sites) UpdateStatus(site *siteapi.Site) (result *siteapi.Site, err error) {
	result = &siteapi.Site{}
	err = c.r.Put().Namespace(c.ns).Resource("sites").Name(site.Name).SubResource("status").Body(site).Do().Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested sites.
func (c *sites) Watch(opts kapi.ListOptions) (watch.Interface, error) {
	return c.r.Get().
		Prefix("watch").
		Namespace(c.ns).
		Resource("sites").
		VersionedParams(&opts, kapi.ParameterCodec).
		Watch()
}
