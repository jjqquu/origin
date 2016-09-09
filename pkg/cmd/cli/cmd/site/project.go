package site

import (
	"bytes"
	"fmt"
	"strconv"

	marathon "github.com/jjqquu/go_marathon"
	"github.com/openshift/origin/pkg/client"
)

type ProjectList struct {
	client client.Interface
	format Formatter
}

func (p ProjectList) Apply(o *SiteOptions) {
	projects, e := p.client.Sites(o.Namespace).GetProjectList(o.SiteId)
	Check(e == nil, "failed to get response: ", e)
	fmt.Println(p.format.Format(projects, p.Humanize))
}

func (g ProjectList) Humanize(body interface{}) string {
	root := body.(*marathon.Groups)
	return columnizeGroup(root)
}

func columnizeGroup(root *marathon.Groups) string {
	title := "PROJECTID APPS\n"
	var b bytes.Buffer
	for _, group := range root.Groups {
		gatherGroup(group, &b)
	}
	text := title + b.String()
	return Columnize(text)
}

func gatherGroup(g *marathon.Group, b *bytes.Buffer) {
	b.WriteString(g.ID)
	b.WriteString(" ")
	b.WriteString(strconv.Itoa(len(g.Apps)))
	b.WriteString("\n")
}

type ProjectDelete struct {
	client client.Interface
	format Formatter
}

func (p ProjectDelete) Apply(o *SiteOptions) {
	deployid, e := p.client.Sites(o.Namespace).DeleteProject(o.SiteId, o.ProjectId)
	Check(e == nil, "failed to get response: ", e)
	fmt.Println(p.format.Format(deployid, p.Humanize))
}

func (g ProjectDelete) Humanize(body interface{}) string {
	deployId := body.(*marathon.DeploymentID)
	title := "DEPLOYID VERSION\n"
	text := title + deployId.DeploymentID + " " + deployId.Version
	return Columnize(text)
}
