package site

import (
	"bytes"
	"fmt"
	"strconv"

	marathon "github.com/jjqquu/go_marathon"
	"github.com/openshift/origin/pkg/client"
)

type DeployList struct {
	client client.Interface
	format Formatter
}

func (d DeployList) Apply(o *SiteOptions) {
	deploys, e := d.client.Sites(o.Namespace).GetDeploymentList(o.SiteId, o.ProjectId)
	Check(e == nil, "failed to get response: ", e)
	fmt.Println(d.format.Format(deploys, d.Humanize))
}

func (d DeployList) Humanize(body interface{}) string {
	deploys := body.(*[]marathon.Deployment)
	title := "DEPLOYID VERSION PROGRESS APPS\n"
	var b bytes.Buffer
	for _, deploy := range *deploys {
		b.WriteString(deploy.ID)
		b.WriteString(" ")
		b.WriteString(deploy.Version)
		b.WriteString(" ")
		b.WriteString(strconv.Itoa(deploy.CurrentStep))
		b.WriteString("/")
		b.WriteString(strconv.Itoa(deploy.TotalSteps))
		b.WriteString(" ")
		for _, app := range deploy.AffectedApps {
			b.WriteString(app)
		}
		b.UnreadRune()
		b.WriteString("\n")
	}
	text := title + b.String()
	return Columnize(text)
}

type DeployDelete struct {
	client client.Interface
	format Formatter
}

func (d DeployDelete) Apply(o *SiteOptions) {
	deployId, e := d.client.Sites(o.Namespace).DeleteDeployment(o.SiteId, o.ProjectId, o.DeploymentId)
	Check(e == nil, "failed to get response: ", e)
	fmt.Println(d.format.Format(deployId, d.Humanize))
}

func (d DeployDelete) Humanize(body interface{}) string {
	rollback := body.(*marathon.DeploymentID)
	title := "DEPLOYID VERSION\n"
	text := title + rollback.DeploymentID + " " + rollback.Version
	return Columnize(text)
}
