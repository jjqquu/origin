package site

import (
	"fmt"
	//"io"
	//"net/url"
	"os"
	//"os/signal"
	//"syscall"

	//"github.com/docker/docker/pkg/term"
	//"github.com/golang/glog"
	"github.com/spf13/cobra"
	//"k8s.io/kubernetes/pkg/api"
	//"k8s.io/kubernetes/pkg/client/restclient"

	//kerrors "k8s.io/kubernetes/pkg/api/errors"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	kcmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	//"k8s.io/kubernetes/pkg/kubectl/resource"

	"github.com/openshift/origin/pkg/client"
)

const (
	SiteRecommendedCommandName = "site"

	siteLong = `  This command help you to view, add, modify or remove a project/app/task/deployment/marathon in the remote site.
  app
    list                      - list all apps in your current project
    versions [id]             - list all versions of apps of id
    show [id]                 - show config and status of app of id (latest version)
    show [id] [version]       - show config and status of app of id and version
    create [jsonfile]         - deploy application defined in jsonfile
    update [jsonfile]         - update application as defined in jsonfile
    update [id] [jsonfile]    - update application id as defined in jsonfile
    update cpu [id] [cpu%]    - update application id to have cpu% of cpu share
    update memory [id] [MB]   - update application id to have MB of memory
    update instances [id] [N] - update application id to have N instances
    restart [id]              - restart app of id
    delete [id]               - remove all instances of id

  task
    list [id]          - list tasks of app of id
    kill [id]          - kill all tasks of app id
    kill [id] [taskid] - kill task taskid of app id
    queue              - list all queued tasks

  project
    list                          - list all projects
    list [projectid]              - list groups in projectid
    create [jsonfile]             - create a project defined in jsonfile
    update [projectid] [jsonfile] - update project id as defined in jsonfile
    delete [projectid]            - delete project of projectid

  deploy
    list               - list all active deploys
    delete [deployid] - cancel deployment of [deployid]

  marathon
    leader   - get the current Marathon leader
    abdicate - force the current leader to relinquish control
    ping     - ping Marathon master host[s]

  format
    human  - simplified columns, default
    json   - json on one line
    jsonpp - json pretty printed
    raw    - the exact response from Marathon
`

	siteExample = `  # Get task information of specified site_id/project_id/application_id/task_id
  $ %[1]s [site_name] --list --apps 
  $ %[1]s [site_name] --list --apps --format=json
  $ %[1]s [site_name] --show --app=[appid] 
  $ %[1]s [site_name] --show --app=[appid] --format=json
  $ %[1]s [site_name] --create --app=[appid] --file=[jsonfile]
  $ %[1]s [site_name] --update --app=[appid] --file=[jsonfile]
  $ %[1]s [site_name] --update --app=[appid] --cpu=[cpu]
  $ %[1]s [site_name] --update --app=[appid] --mem=[MB]
  $ %[1]s [site_name] --update --app=[appid] --replicas=[N]
  $ %[1]s [site_name] --delete --app=[appid]
  $ %[1]s [site_name] --restart --app=[appid]

  $ %[1]s [site_name] --list --app=[appid] --versions 
  $ %[1]s [site_name] --show --app=[appid] --version=[version] --format=json

  $ %[1]s [site_name] --list --app=[appid] --tasks 
  $ %[1]s [site_name] --list --app=[appid] --tasks --format=json
  $ %[1]s [site_name] --kill --app=[appid] --task=[taskid1,taskid2] 

  $ %[1]s [site_name] --list --projects 
  $ %[1]s [site_name] --list --projects --format=json
  $ %[1]s [site_name] --delete --project 

  $ %[1]s [site_name] --list --deploys 
  $ %[1]s [site_name] --list --deploys --format=jsonpp
  $ %[1]s [site_name] --delete --deploy=[deployment id] 
`
)

const (
	ActionList    = "list"
	ActionShow    = "show"
	ActionCreate  = "create"
	ActionUpdate  = "update"
	ActionDelete  = "delete"
	ActionRestart = "restart"
	ActionKill    = "kill"
)

const (
	ObjectApplication = "application"
	ObjectTask        = "task"
	ObjectProject     = "project"
	ObjectDeployment  = "deployment"
	ObjectVersion     = "version"
)

func NewCmdSite(name, fullName string, f *clientcmd.Factory) *cobra.Command {
	options := &SiteOptions{}

	cmd := &cobra.Command{
		Use:        "site SITENAME --COMMAND [args...]",
		Short:      "view, add, modify or remove a project/app/task/deployment/marathon in the remote site.",
		Long:       siteLong,
		Example:    fmt.Sprintf(siteExample, fullName),
		SuggestFor: []string{"site"},
		Run: func(cmd *cobra.Command, args []string) {
			if err := options.Complete(f, cmd, args); err != nil {
				kcmdutil.CheckErr(err)
			}

			if err := options.Validate(); err != nil {
				kcmdutil.CheckErr(kcmdutil.UsageError(cmd, err.Error()))
			}

			if err := options.RunSiteProxy(); err != nil {
				kcmdutil.CheckErr(err)
			}
		},
	}

	cmd.Flags().BoolVar(&options.IsList, "list", false, "List all the object of projects/application/tasks/versions/deployment information for the specified site.")
	cmd.Flags().BoolVar(&options.IsShow, "show", false, "Show the details information of the object of projects/application/tasks/versions/deployments the specified site.")
	cmd.Flags().BoolVar(&options.IsCreate, "create", false, "Create deployment information of the specified site.")
	cmd.Flags().BoolVar(&options.IsDelete, "delete", false, "Delete information from the specified site.")
	cmd.Flags().BoolVar(&options.IsUpdate, "update", false, "Update deployment information of the specified site.")
	cmd.Flags().BoolVar(&options.IsRestart, "restart", false, "Restart information from the specified site.")
	cmd.Flags().BoolVar(&options.IsKill, "kill", false, "Kill information from the specified site.")

	cmd.Flags().BoolVarP(&options.IsProject, "project", "p", false, "Project")

	cmd.Flags().BoolVar(&options.IsApps, "apps", false, "Project")
	cmd.Flags().BoolVar(&options.IsProjects, "projects", false, "Project")
	cmd.Flags().BoolVar(&options.IsVersions, "versions", false, "Project")
	cmd.Flags().BoolVar(&options.IsTasks, "tasks", false, "Project")
	cmd.Flags().BoolVar(&options.IsDeployments, "deploys", false, "Project")

	cmd.Flags().StringVarP(&options.AppId, "app", "a", "", "Application")
	cmd.Flags().StringVarP(&options.TaskId, "task", "t", "", "Task")
	cmd.Flags().StringVarP(&options.DeploymentId, "deploy", "d", "", "Deployment")
	cmd.Flags().StringVarP(&options.Version, "version", "v", "", "Version")

	cmd.Flags().Float64Var(&options.Cpu, "cpu", -1, "CPU percentage")
	cmd.Flags().Float64Var(&options.Mem, "mem", -1, "Memory")
	cmd.Flags().IntVar(&options.Replicas, "replicas", -1, "Replica number")

	cmd.Flags().StringVarP(&options.FileName, "file", "f", "", "File name")

	cmd.Flags().StringVar(&options.Format, "format", "human", "show information in what format (human/json/jsonpp/raw)")

	return cmd
}

// SiteOptions declare the arguments accepted by the Proxy command
type SiteOptions struct {
	Namespace string
	SiteId    string
	ProjectId string

	AppId        string
	TaskId       string
	Version      string
	DeploymentId string

	Cpu      float64
	Mem      float64
	Replicas int
	FileName string
	Format   string

	Client client.Interface

	Action     string
	ObjectType string

	//action flag
	IsList    bool
	IsShow    bool
	IsCreate  bool
	IsDelete  bool
	IsUpdate  bool
	IsRestart bool
	IsKill    bool

	//object type flag
	IsApps        bool
	IsProject     bool
	IsProjects    bool
	IsVersions    bool
	IsTasks       bool
	IsDeployments bool
}

type Action interface {
	Apply(o *SiteOptions)
}

// Complete verifies command line arguments and loads data from the command environment
func (o *SiteOptions) Complete(f *clientcmd.Factory, cmd *cobra.Command, args []string) error {
	var err error

	if len(args) > 0 {
		o.SiteId = args[0]
	}

	namespace, _, err := f.DefaultNamespace()
	if err != nil {
		return err
	}
	o.Namespace = namespace
	o.ProjectId = namespace

	o.Client, _, err = f.Clients()
	if err != nil {
		return err
	}

	return nil
}

// Validate checks that the provided proxy options are specified.
func (o *SiteOptions) Validate() error {
	if len(o.SiteId) == 0 {
		return fmt.Errorf("site name must be specified")
	}

	numActions := 0
	if o.IsList {
		o.Action = ActionList
		numActions++
	}
	if o.IsShow {
		o.Action = ActionShow
		numActions++
	}
	if o.IsCreate {
		o.Action = ActionCreate
		numActions++
	}
	if o.IsDelete {
		o.Action = ActionDelete
		numActions++
	}
	if o.IsUpdate {
		o.Action = ActionUpdate
		numActions++
	}
	if o.IsRestart {
		o.Action = ActionRestart
		numActions++
	}
	if o.IsKill {
		o.Action = ActionKill
		numActions++
	}
	if numActions == 0 {
		return fmt.Errorf("one of --list, --show, --create, --delete, --update, --restart, --kill must be specified")
	} else if numActions > 1 {
		return fmt.Errorf("only one of --list, --show, --create, --delete, --update, --restart, --kill is allowed")
	}

	numObjects := 0
	if o.IsProject || o.IsProjects {
		o.ObjectType = ObjectProject
		numObjects++
	}
	if len(o.DeploymentId) > 0 || o.IsDeployments {
		o.ObjectType = ObjectDeployment
		numObjects++
	}
	if len(o.AppId) > 0 && (len(o.Version) > 0 || o.IsVersions) {
		o.ObjectType = ObjectVersion
		numObjects++
	}
	if len(o.AppId) > 0 && (len(o.TaskId) > 0 || o.IsTasks) {
		o.ObjectType = ObjectTask
		numObjects++
	}
	if o.IsApps {
		o.ObjectType = ObjectApplication
		numObjects++
	}
	if len(o.AppId) > 0 && len(o.Version) == 0 && len(o.TaskId) == 0 && len(o.DeploymentId) == 0 &&
		!o.IsVersions && !o.IsTasks && !o.IsProject && !o.IsProjects {
		o.ObjectType = ObjectApplication
		numObjects++
	}
	if numObjects == 0 {
		return fmt.Errorf("one type of application, projects, deployment, tasks, version must be specified")
	} else if numObjects > 1 {
		return fmt.Errorf("only one type of application, projects, deployment, tasks, version must be specified")
	}

	return nil
}

// Run executes a validated remote execution against a pod.
func (o *SiteOptions) RunSiteProxy() error {

	// TODO: firstly check if site exists and is running, then invoke

	c := o.Client
	f := NewFormatter(o.Format)
	app := map[string]Action{
		ActionList:    AppList{c, f},
		ActionShow:    AppShow{c, f},
		ActionCreate:  AppCreate{c, f},
		ActionUpdate:  AppUpdate{c, f},
		ActionRestart: AppRestart{c, f},
		ActionDelete:  AppDelete{c, f},
	}
	ver := map[string]Action{
		ActionList: AppVersionsList{c, f},
		ActionShow: AppVersionShow{c, f},
	}
	task := map[string]Action{
		ActionList: TaskList{c, f},
		ActionKill: TaskKill{c, f},
	}
	proj := map[string]Action{
		ActionList:   ProjectList{c, f},
		ActionDelete: ProjectDelete{c, f},
	}
	deploy := map[string]Action{
		ActionList:   DeployList{c, f},
		ActionDelete: DeployDelete{c, f},
	}
	objActs := map[string]map[string]Action{
		ObjectApplication: app,
		ObjectVersion:     ver,
		ObjectTask:        task,
		ObjectProject:     proj,
		ObjectDeployment:  deploy,
	}

	actions, exist := objActs[o.ObjectType]
	if !exist {
		return fmt.Errorf("object type (%v) not supported", o.ObjectType)
	}

	action, exist := actions[o.Action]
	if !exist {
		return fmt.Errorf("action type (%v) not supported", o.Action)
	}

	action.Apply(o)

	return nil
}

func Check(b bool, args ...interface{}) {
	if !b {
		fmt.Fprintln(os.Stderr, args...)
		os.Exit(1)
	}
}
