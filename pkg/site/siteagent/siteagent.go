package siteagent

const (
	Ping = "ping"

	Sites        = "sites"
	SiteId       = "site_id"
	Projects     = "projects"
	ProjId       = "proj_id"
	Applications = "applications"
	AppId        = "app_id"
	Tasks        = "tasks"
	TaskId       = "task_id"
	Versions     = "versions"
	VerId        = "ver_id"
	Jobs         = "jobs"
	JobId        = "job_id"
	Deployments  = "deployments"
	DeploymentId = "dp_id"

	// attributes of system resource(site/project/application/job/task/deployment/version etc)
	Attributes       = "attributes"
	AttrQuota        = "quota"
	AttrInfo         = "info"
	AttrLeader       = "leader"
	AttrList         = "list"
	AttrScale        = "scale"
	AttrRestart      = "restart"
	AttrQueue        = "queue"
	AttrJobs         = "Jobs"
	AttrTasks        = "tasks"
	AttrApplications = "applications"
	AttrProjects     = "projects"
	AttrVersions     = "versions"
	AttrReplicas     = "replicas"

	// optional query string parameters
	ParameterForce = "force"
	ParameterScale = "scale"
	ParameterWipe  = "wipe"
	ParameterHost  = "host"

	QuerySiteId       = "{" + SiteId + "}"
	QueryProjId       = "{" + ProjId + "}"
	QueryAppId        = "{" + AppId + "}"
	QueryTaskId       = "{" + TaskId + "}"
	QueryVerId        = "{" + VerId + "}"
	QueryDeploymentId = "{" + DeploymentId + "}"
	QueryJobId        = "{" + JobId + "}"
	QueryReplicas     = "{" + AttrReplicas + "}"
)
