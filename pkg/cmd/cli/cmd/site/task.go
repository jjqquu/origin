package site

import (
	"bytes"
	"fmt"
	"strings"

	marathon "github.com/jjqquu/go_marathon"
	"github.com/openshift/origin/pkg/client"
)

type TaskList struct {
	client client.Interface
	format Formatter
}

func (t TaskList) Apply(o *SiteOptions) {
	tasks, e := t.client.Sites(o.Namespace).GetTaskList(o.SiteId, o.ProjectId, o.AppId)
	Check(e == nil, "failed to get response: ", e)
	fmt.Println(t.format.Format(tasks, t.Humanize))
}

func (t TaskList) Humanize(body interface{}) string {
	tasks := body.(*marathon.Tasks)
	var b bytes.Buffer
	for _, task := range tasks.Tasks {
		b.WriteString(task.ID)
		b.WriteString(" ")
		b.WriteString(task.Host)
		b.WriteString(" ")
		b.WriteString(task.Version)
		b.WriteString("\n")
		// ports?
	}
	title := "ID HOST VERSION\n"
	text := title + b.String()
	return Columnize(text)
}

type TaskKill struct {
	client client.Interface
	format Formatter
}

func (t TaskKill) Apply(o *SiteOptions) {
	taskIdList := strings.Split(o.TaskId, ",")
	e := t.client.Sites(o.Namespace).DeleteTasks(o.SiteId, o.ProjectId, o.AppId, taskIdList)
	Check(e == nil, "failed to get response: ", e)
	fmt.Printf("Task %s have been killed successfully\n", o.TaskId)
}

func (t TaskKill) Humanize(body interface{}) string {
	return ""
}
