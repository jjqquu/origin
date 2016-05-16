package site

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"

	marathon "github.com/jjqquu/go_marathon"
	"github.com/openshift/origin/pkg/client"
)

type AppList struct {
	client client.Interface
	format Formatter
}

func (a AppList) Apply(o *SiteOptions) {
	apps, e := a.client.Sites(o.Namespace).GetProjectApplicationList(o.SiteId, o.ProjectId)
	Check(e == nil, "failed to get response: ", e)
	fmt.Println(a.format.Format(apps, a.Humanize))
}

func (a AppList) Humanize(body interface{}) string {
	applications := body.(*[]marathon.Application)
	title := "APP VERSION USER\n"
	var b bytes.Buffer
	for _, app := range *applications {
		b.WriteString(app.ID)
		b.WriteString(" ")
		b.WriteString(app.Version)
		b.WriteString(" ")
		b.WriteString(app.User)
		b.WriteString("\n")
	}
	text := title + b.String()
	return Columnize(text)
}

type AppVersionsList struct {
	client client.Interface
	format Formatter
}

func (a AppVersionsList) Apply(o *SiteOptions) {
	appVers, e := a.client.Sites(o.Namespace).GetVersionList(o.SiteId, o.ProjectId, o.AppId)
	Check(e == nil, "failed to get response: ", e)
	fmt.Println(a.format.Format(appVers, a.Humanize))
}

func (a AppVersionsList) Humanize(body interface{}) string {
	versions := body.(*marathon.ApplicationVersions)
	var b bytes.Buffer
	b.WriteString("VERSIONS\n")
	for _, version := range versions.Versions {
		b.WriteString(version)
		b.WriteString("\n")
	}
	return b.String()
}

type AppVersionShow struct {
	client client.Interface
	format Formatter
}

func (a AppVersionShow) Apply(o *SiteOptions) {
	app, e := a.client.Sites(o.Namespace).GetVersion(o.SiteId, o.ProjectId, o.AppId, o.Version)
	Check(e == nil, "failed to get response: ", e)
	fmt.Println(a.format.Format(app, a.Humanize))
}

func (a AppVersionShow) Humanize(body interface{}) string {
	app := body.(*marathon.Application)
	title := "INSTANCES CPU MEM DISK CMD\n"
	var b bytes.Buffer
	b.WriteString(strconv.Itoa(*app.Instances))
	b.WriteString(" ")
	cpu := fmt.Sprintf("%.2f", app.CPUs)
	b.WriteString(cpu)
	b.WriteString(" ")
	mem := fmt.Sprintf("%.2f", *app.Mem)
	b.WriteString(mem)
	b.WriteString(" ")
	disk := fmt.Sprintf("%.2f", *app.Disk)
	b.WriteString(disk)
	b.WriteString(" ")
	if app.Cmd != nil {
		b.WriteString(*app.Cmd)
	} else {
		b.WriteString("<empty>")
	}
	text := title + b.String()
	return Columnize(text)
}

type AppShow struct {
	client client.Interface
	format Formatter
}

func (a AppShow) Apply(o *SiteOptions) {
	app, e := a.client.Sites(o.Namespace).GetApplication(o.SiteId, o.ProjectId, o.AppId)
	Check(e == nil, "failed to get response: ", e)
	if app == nil {
		fmt.Printf("the application %s doesn't exist\n", o.AppId)
	} else {
		fmt.Println(a.format.Format(app, a.Humanize))
	}
}

func (a AppShow) Humanize(body interface{}) string {
	app := body.(*marathon.Application)
	title := "INSTANCES CPU MEM DISK CMD\n"
	var b bytes.Buffer
	b.WriteString(strconv.Itoa(*app.Instances))
	b.WriteString(" ")
	cpu := fmt.Sprintf("%.2f", app.CPUs)
	b.WriteString(cpu)
	b.WriteString(" ")
	mem := fmt.Sprintf("%.2f", *app.Mem)
	b.WriteString(mem)
	b.WriteString(" ")
	disk := fmt.Sprintf("%.2f", *app.Disk)
	b.WriteString(disk)
	b.WriteString(" ")
	if app.Cmd != nil {
		b.WriteString(*app.Cmd)
	} else {
		b.WriteString("<empty>")
	}
	text := title + b.String()
	return Columnize(text)
}

type AppCreate struct {
	client client.Interface
	format Formatter
}

func (a AppCreate) Apply(o *SiteOptions) {
	body, e := ioutil.ReadFile(o.FileName)
	Check(e == nil, "failed to read json file: ", e)
	newApp := new(marathon.Application)
	e = json.Unmarshal(body, newApp)
	Check(e == nil, "failed to decode json file "+o.FileName+": ", e)

	appId := "/" + o.ProjectId + "/" + o.AppId
	newApp.ID = appId //command argument overrides file content
	app, e := a.client.Sites(o.Namespace).CreateApplication(o.SiteId, o.ProjectId, o.AppId, newApp)
	Check(e == nil, "failed to get response: ", e)
	fmt.Println(a.format.Format(app, a.Humanize))
}

func (a AppCreate) Humanize(body interface{}) string {
	application := body.(*marathon.Application)
	var b bytes.Buffer
	b.WriteString("APPID VERSION\n")
	b.WriteString(application.ID)
	b.WriteString(" ")
	b.WriteString(application.Version)
	return Columnize(b.String())
}

type AppUpdate struct {
	client client.Interface
	format Formatter
}

func (a AppUpdate) Apply(o *SiteOptions) {
	switch {
	case len(o.FileName) > 0:
		a.fromJsonFile(o)
	case o.Cpu > 0 || o.Mem > 0 || o.Replicas > 0:
		a.fromCLI(o)
	default:
		Check(false, "update arguments required")
	}
}

func (a AppUpdate) fromJsonFile(o *SiteOptions) {
	body, e := ioutil.ReadFile(o.FileName)
	Check(e == nil, "failed to read json file: ", e)
	newApp := new(marathon.Application)
	e = json.Unmarshal(body, newApp)
	Check(e == nil, "failed to decode json file "+o.FileName+": ", e)

	appId := "/" + o.ProjectId + "/" + o.AppId
	newApp.ID = appId //command argument overrides file content
	updatedApp, e := a.client.Sites(o.Namespace).UpdateApplication(o.SiteId, o.ProjectId, o.AppId, newApp, false)
	Check(e == nil, "failed to get response: ", e)
	fmt.Println(a.format.Format(updatedApp, a.Humanize))
}

func (a AppUpdate) fromCLI(o *SiteOptions) {
	newApp := new(marathon.Application)
	appId := "/" + o.ProjectId + "/" + o.AppId
	newApp.ID = appId
	switch {
	case o.Replicas > 0:
		newApp.Instances = &o.Replicas
	case o.Mem > 0:
		newApp.Mem = &o.Mem
	case o.Cpu > 0:
		newApp.CPUs = o.Cpu
	}

	updatedApp, e := a.client.Sites(o.Namespace).UpdateApplication(o.SiteId, o.ProjectId, o.AppId, newApp, false)
	Check(e == nil, "failed to get response: ", e)
	fmt.Println(a.format.Format(updatedApp, a.Humanize))
}

func (a AppUpdate) Humanize(body interface{}) string {
	update := body.(*marathon.DeploymentID)
	title := "DEPLOYID VERSION\n"
	text := title + update.DeploymentID + " " + update.Version
	return Columnize(text)
}

type AppRestart struct {
	client client.Interface
	format Formatter
}

func (a AppRestart) Apply(o *SiteOptions) {
	deployId, e := a.client.Sites(o.Namespace).RestartApplication(o.SiteId, o.ProjectId, o.AppId)
	Check(e == nil, "failed to get response: ", e)
	fmt.Println(a.format.Format(deployId, a.Humanize))
}

func (a AppRestart) Humanize(body interface{}) string {
	update := body.(*marathon.DeploymentID)
	title := "DEPLOYID VERSION\n"
	text := title + update.DeploymentID + " " + update.Version
	return Columnize(text)
}

type AppDelete struct {
	client client.Interface
	format Formatter
}

func (a AppDelete) Apply(o *SiteOptions) {
	deployId, e := a.client.Sites(o.Namespace).DeleteApplication(o.SiteId, o.ProjectId, o.AppId)
	Check(e == nil, "failed to get response: ", e)
	fmt.Println(a.format.Format(deployId, a.Humanize))
}

func (a AppDelete) Humanize(body interface{}) string {
	update := body.(*marathon.DeploymentID)
	title := "DEPLOYID VERSION\n"
	text := title + update.DeploymentID + " " + update.Version
	return Columnize(text)
}
