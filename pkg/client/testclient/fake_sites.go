package testclient

import (
	kapi "k8s.io/kubernetes/pkg/api"
	ktestclient "k8s.io/kubernetes/pkg/client/unversioned/testclient"
	"k8s.io/kubernetes/pkg/watch"

	siteapi "github.com/openshift/origin/pkg/site/api"
)

// FakeSites implements SiteInterface. Meant to be embedded into a struct to get a default
// implementation. This makes faking out just the methods you want to test easier.
type FakeSites struct {
	Fake      *Fake
	Namespace string
}

func (c *FakeSites) Get(name string) (*siteapi.Site, error) {
	obj, err := c.Fake.Invokes(ktestclient.NewGetAction("sites", c.Namespace, name), &siteapi.Site{})
	if obj == nil {
		return nil, err
	}

	return obj.(*siteapi.Site), err
}

func (c *FakeSites) List(opts kapi.ListOptions) (*siteapi.SiteList, error) {
	obj, err := c.Fake.Invokes(ktestclient.NewListAction("sites", c.Namespace, opts), &siteapi.SiteList{})
	if obj == nil {
		return nil, err
	}

	return obj.(*siteapi.SiteList), err
}

func (c *FakeSites) Create(inObj *siteapi.Site) (*siteapi.Site, error) {
	obj, err := c.Fake.Invokes(ktestclient.NewCreateAction("sites", c.Namespace, inObj), inObj)
	if obj == nil {
		return nil, err
	}

	return obj.(*siteapi.Site), err
}

func (c *FakeSites) Update(inObj *siteapi.Site) (*siteapi.Site, error) {
	obj, err := c.Fake.Invokes(ktestclient.NewUpdateAction("sites", c.Namespace, inObj), inObj)
	if obj == nil {
		return nil, err
	}

	return obj.(*siteapi.Site), err
}

func (c *FakeSites) UpdateStatus(inObj *siteapi.Site) (*siteapi.Site, error) {
	action := ktestclient.NewUpdateAction("sites", c.Namespace, inObj)
	action.Subresource = "status"
	obj, err := c.Fake.Invokes(action, inObj)
	if obj == nil {
		return nil, err
	}

	return obj.(*siteapi.Site), err
}

func (c *FakeSites) Delete(name string) error {
	_, err := c.Fake.Invokes(ktestclient.NewDeleteAction("sites", c.Namespace, name), &siteapi.Site{})
	return err
}

func (c *FakeSites) Watch(opts kapi.ListOptions) (watch.Interface, error) {
	return c.Fake.InvokesWatch(ktestclient.NewWatchAction("sites", c.Namespace, opts))
}
