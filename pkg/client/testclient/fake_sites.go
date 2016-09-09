package testclient

import (
	kapi "k8s.io/kubernetes/pkg/api"
	ktestclient "k8s.io/kubernetes/pkg/client/unversioned/testclient"
	"k8s.io/kubernetes/pkg/watch"

	siteapi "github.com/openshift/origin/pkg/site/api"

	marathon "github.com/jjqquu/go_marathon"
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

func (c *FakeSites) GetApplication(siteId string, projId string, appId string) (result *marathon.Application, err error) {
	return nil, nil
}

func (c *FakeSites) GetProjectApplicationList(siteId string, projId string) (result *[]marathon.Application, err error) {
	return nil, nil
}

func (c *FakeSites) GetApplicationList(siteId string) (result *marathon.Applications, err error) {
	return nil, nil
}

func (c *FakeSites) UpdateApplication(siteId string, projId string, appId string, app *marathon.Application) (result *marathon.DeploymentID, err error) {
	return nil, nil
}

func (c *FakeSites) ScaleApplication(siteId string, projId string, appId string, replicas int) (result *marathon.DeploymentID, err error) {
	return nil, nil
}

func (c *FakeSites) RestartApplication(siteId string, projId string, appId string) (result *marathon.DeploymentID, err error) {
	return nil, nil
}

func (c *FakeSites) CreateApplication(siteId string, projId string, appId string, app *marathon.Application) (result *marathon.Application, err error) {
	return nil, nil
}

func (c *FakeSites) DeleteApplication(siteId string, projId string, appId string) (result *marathon.DeploymentID, err error) {
	return nil, nil
}

func (c *FakeSites) GetDeployment(siteId string, projId string, deploymentId string) (result *marathon.Deployment, err error) {
	return nil, nil
}

func (c *FakeSites) GetDeploymentList(siteId string, projId string) (result *[]marathon.Deployment, err error) {
	return nil, nil
}

func (c *FakeSites) DeleteDeployment(siteId string, projId string, deploymentId string) (result *marathon.DeploymentID, err error) {
	return nil, nil
}

func (c *FakeSites) GetProject(siteId string, projId string) (result *marathon.Group, err error) {
	return nil, nil
}

func (c *FakeSites) GetProjectList(siteId string) (result *marathon.Groups, err error) {
	return nil, nil
}

func (c *FakeSites) DeleteProject(siteId string, projId string) (result *marathon.DeploymentID, err error) {
	return nil, nil
}

func (c *FakeSites) GetTaskList(siteId string, projId string, appId string) (result *marathon.Tasks, err error) {
	return nil, nil
}

func (c *FakeSites) DeleteTasks(siteId string, projId string, appId string, taskIdList []string) error {
	return nil
}

func (c *FakeSites) GetVersion(siteId string, projId string, appId string, verId string) (result *marathon.Application, err error) {
	return nil, nil
}

func (c *FakeSites) GetVersionList(siteId string, projId string, appId string) (result *marathon.ApplicationVersions, err error) {
	return nil, nil
}
