package client

import (
	"fmt"
	"strconv"

	"encoding/json"
	"net/http"

	kapi "k8s.io/kubernetes/pkg/api"
	kerrors "k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/client/restclient"
	"k8s.io/kubernetes/pkg/watch"

	marathon "github.com/jjqquu/go_marathon"
	siteapi "github.com/openshift/origin/pkg/site/api"
	sconst "github.com/openshift/origin/pkg/site/siteagent"
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

	GetApplication(siteId string, projId string, appId string) (result *marathon.Application, err error)
	GetProjectApplicationList(siteId string, projId string) (result *[]marathon.Application, err error)
	GetApplicationList(siteId string) (result *marathon.Applications, err error)
	UpdateApplication(siteId string, projId string, appId string, app *marathon.Application) (result *marathon.DeploymentID, err error)
	ScaleApplication(siteId string, projId string, appId string, replicas int) (result *marathon.DeploymentID, err error)
	RestartApplication(siteId string, projId string, appId string) (result *marathon.DeploymentID, err error)
	CreateApplication(siteId string, projId string, appId string, app *marathon.Application) (result *marathon.Application, err error)
	DeleteApplication(siteId string, projId string, appId string) (result *marathon.DeploymentID, err error)

	GetDeployment(siteId string, projId string, deploymentId string) (result *marathon.Deployment, err error)
	GetDeploymentList(siteId string, projId string) (result *[]marathon.Deployment, err error)
	DeleteDeployment(siteId string, projId string, deploymentId string) (result *marathon.DeploymentID, err error)

	GetProject(siteId string, projId string) (result *marathon.Group, err error)
	GetProjectList(siteId string) (result *marathon.Groups, err error)
	DeleteProject(siteId string, projId string) (result *marathon.DeploymentID, err error)

	GetTaskList(siteId string, projId string, appId string) (result *marathon.Tasks, err error)
	DeleteTasks(siteId string, projId string, appId string, taskIdList []string) error

	GetVersion(siteId string, projId string, appId string, verId string) (result *marathon.Application, err error)
	GetVersionList(siteId string, projId string, appId string) (result *marathon.ApplicationVersions, err error)
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

func handleApi(call restclient.Result, resVal interface{}) (interface{}, error) {
	err := call.Error()
	if err == nil {
		var code int
		call.StatusCode(&code)
		if code == http.StatusOK {
			if resVal != nil {
				body, _ := call.Raw()
				if err := json.Unmarshal(body, resVal); err != nil {
					return nil, err
				}
				return resVal, nil
			} else {
				return nil, nil
			}
		} else {
			return nil, nil
		}
	} else if statusErr, ok := err.(*kerrors.StatusError); ok {
		switch statusErr.ErrStatus.Code {
		case http.StatusNotFound:
			return nil, nil
		case http.StatusNotAcceptable:
			if statusErr.ErrStatus.Details.Causes != nil {
				return nil, fmt.Errorf("site agent return error: %s", statusErr.ErrStatus.Details.Causes[0].Message)
			} else {
				return nil, fmt.Errorf("site agent return not acceptable error: %s", statusErr.ErrStatus.Message)
			}
		default:
			return nil, fmt.Errorf("site agent return unexpected error: %v", statusErr)
		}
	} else {
		return nil, fmt.Errorf("api server internal error: %v", err)
	}
}

// TODO: later we may consider to replace marathon.xxx with golang interface{} type so as to support more types
//       of site: e.g. ECS/ACS etc.
func (c *sites) GetApplication(siteId string, projId string, appId string) (result *marathon.Application, err error) {
	result = new(marathon.Application)
	call := c.r.Get().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Projects, projId).
		Param(sconst.Applications, appId).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*marathon.Application)
	return
}

func (c *sites) GetProjectApplicationList(siteId string, projId string) (result *[]marathon.Application, err error) {
	result = new([]marathon.Application)
	call := c.r.Get().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Projects, projId).
		Param(sconst.Attributes, sconst.Applications).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*[]marathon.Application)
	return
}

func (c *sites) GetApplicationList(siteId string) (result *marathon.Applications, err error) {
	result = new(marathon.Applications)
	call := c.r.Get().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Attributes, sconst.Applications).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*marathon.Applications)
	return
}

func (c *sites) UpdateApplication(siteId string, projId string, appId string, app *marathon.Application) (result *marathon.DeploymentID, err error) {
	appBody, err := json.Marshal(app)
	if err != nil {
		return nil, err
	}
	result = new(marathon.DeploymentID)
	call := c.r.Put().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Projects, projId).
		Param(sconst.Applications, appId).
		Body(appBody).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*marathon.DeploymentID)
	return
}

func (c *sites) ScaleApplication(siteId string, projId string, appId string, replicas int) (result *marathon.DeploymentID, err error) {
	result = new(marathon.DeploymentID)
	call := c.r.Put().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Projects, projId).
		Param(sconst.Applications, appId).
		Param(sconst.AttrReplicas, strconv.Itoa(replicas)).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*marathon.DeploymentID)
	return
}

func (c *sites) RestartApplication(siteId string, projId string, appId string) (result *marathon.DeploymentID, err error) {
	result = new(marathon.DeploymentID)
	call := c.r.Put().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Projects, projId).
		Param(sconst.Applications, appId).
		Param(sconst.Attributes, sconst.AttrRestart).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*marathon.DeploymentID)
	return
}

func (c *sites) CreateApplication(siteId string, projId string, appId string, app *marathon.Application) (result *marathon.Application, err error) {
	appBody, err := json.Marshal(app)
	if err != nil {
		return nil, err
	}
	result = new(marathon.Application)
	call := c.r.Post().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Projects, projId).
		Param(sconst.Attributes, sconst.Applications).
		Body(appBody).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*marathon.Application)
	return
}

func (c *sites) DeleteApplication(siteId string, projId string, appId string) (result *marathon.DeploymentID, err error) {
	result = new(marathon.DeploymentID)
	call := c.r.Delete().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Projects, projId).
		Param(sconst.Applications, appId).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*marathon.DeploymentID)
	return
}

func (c *sites) GetDeployment(siteId string, projId string, deploymentId string) (result *marathon.Deployment, err error) {
	result = new(marathon.Deployment)
	call := c.r.Get().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Projects, projId).
		Param(sconst.Deployments, deploymentId).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*marathon.Deployment)
	return
}

func (c *sites) GetDeploymentList(siteId string, projId string) (result *[]marathon.Deployment, err error) {
	result = new([]marathon.Deployment)
	call := c.r.Get().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Projects, projId).
		Param(sconst.Attributes, sconst.Deployments).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*[]marathon.Deployment)
	return
}

func (c *sites) DeleteDeployment(siteId string, projId string, deploymentId string) (result *marathon.DeploymentID, err error) {
	result = new(marathon.DeploymentID)
	call := c.r.Delete().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Projects, projId).
		Param(sconst.Deployments, deploymentId).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*marathon.DeploymentID)
	return
}

func (c *sites) GetProject(siteId string, projId string) (result *marathon.Group, err error) {
	result = new(marathon.Group)
	call := c.r.Get().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Projects, projId).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*marathon.Group)
	return
}

func (c *sites) GetProjectList(siteId string) (result *marathon.Groups, err error) {
	result = new(marathon.Groups)
	call := c.r.Get().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Attributes, sconst.AttrProjects).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*marathon.Groups)
	return
}

func (c *sites) DeleteProject(siteId string, projId string) (result *marathon.DeploymentID, err error) {
	result = new(marathon.DeploymentID)
	call := c.r.Delete().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Projects, projId).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*marathon.DeploymentID)
	return
}

func (c *sites) GetTaskList(siteId string, projId string, appId string) (result *marathon.Tasks, err error) {
	result = new(marathon.Tasks)
	call := c.r.Get().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Projects, projId).
		Param(sconst.Applications, appId).
		Param(sconst.Attributes, sconst.AttrTasks).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*marathon.Tasks)
	return
}

func (c *sites) DeleteTasks(siteId string, projId string, appId string, taskIdList []string) error {
	var post struct {
		IDs []string `json:"ids"`
	}
	post.IDs = taskIdList
	postBody, err := json.Marshal(post)
	if err != nil {
		return err
	}
	call := c.r.Delete().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Projects, projId).
		Param(sconst.Applications, appId).
		Param(sconst.Attributes, sconst.Tasks).
		Body(postBody).
		Do()
	_, err = handleApi(call, nil)
	return err
}

func (c *sites) GetVersion(siteId string, projId string, appId string, verId string) (result *marathon.Application, err error) {
	result = new(marathon.Application)
	call := c.r.Get().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Projects, projId).
		Param(sconst.Applications, appId).
		Param(sconst.Versions, verId).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*marathon.Application)
	return
}

func (c *sites) GetVersionList(siteId string, projId string, appId string) (result *marathon.ApplicationVersions, err error) {
	result = new(marathon.ApplicationVersions)
	call := c.r.Get().
		Namespace(c.ns).
		Resource("sites").
		Name(siteId).
		SubResource("proxy").
		Param(sconst.Sites, siteId).
		Param(sconst.Projects, projId).
		Param(sconst.Applications, appId).
		Param(sconst.Attributes, sconst.AttrVersions).
		Do()
	ret, err := handleApi(call, result)
	result, _ = ret.(*marathon.ApplicationVersions)
	return
}
