package deployer

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/client/restclient"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"

	"github.com/openshift/origin/pkg/client"
	ocmd "github.com/openshift/origin/pkg/cmd/cli/cmd"
	"github.com/openshift/origin/pkg/cmd/util"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	deployapi "github.com/openshift/origin/pkg/deploy/api"
	deployutil "github.com/openshift/origin/pkg/deploy/util"

	marathon "github.com/jjqquu/go_marathon"
)

const (
	defaultTimeout = time.Duration(360) * time.Second

	mdeployerLong = `
Perform a deployment to mesos marathon site

This command launches a deployment as described by a deployment configuration.`
)

type Deployment struct {
	name, namespace, label string

	kclient *kclient.Client
	oclient *client.Client
	sites   client.SiteInterface

	siteId, projId, appId string

	to     *kapi.ReplicationController
	config *deployapi.DeploymentConfig

	desiredReplicas       int
	desiredApp, actualApp *marathon.Application
}

// NewCommandMarathonDeployer provides a CLI handler for deploy.
func NewCommandMarathonDeployer(name string) *cobra.Command {
	deployment := &Deployment{}

	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s%s", name, clientcmd.ConfigSyntax),
		Short: "Run the marathon deployer",
		Long:  mdeployerLong,
		Run: func(c *cobra.Command, args []string) {
			if len(deployment.name) == 0 {
				glog.Fatal("name of deployment is required")
			}
			if len(deployment.namespace) == 0 {
				glog.Fatal("namespace is required")
			}

			kcfg, err := restclient.InClusterConfig()
			if err != nil {
				glog.Fatal(err)
			}
			deployment.kclient, err = kclient.New(kcfg)
			if err != nil {
				glog.Fatal(err)
			}
			deployment.oclient, err = client.New(kcfg)
			if err != nil {
				glog.Fatal(err)
			}

			if err = deployment.execute(); err != nil {
				glog.Fatal(err)
			}
		},
	}

	cmd.AddCommand(ocmd.NewCmdVersion(name, nil, os.Stdout, ocmd.VersionOptions{}))

	flag := cmd.Flags()
	flag.StringVar(&deployment.name, "deployment", util.Env("OPENSHIFT_DEPLOYMENT_NAME", ""), "The deployment name to start")
	flag.StringVar(&deployment.namespace, "namespace", util.Env("OPENSHIFT_DEPLOYMENT_NAMESPACE", ""), "The deployment namespace")

	return cmd
}

func (d *Deployment) execute() error {
	glog.Infof("Deployment %s/%s is being started", d.namespace, d.name)

	var err error

	// Look up the new deployment.
	d.to, err = d.kclient.ReplicationControllers(d.namespace).Get(d.name)
	if err != nil {
		return fmt.Errorf("couldn't get deployment %s/%s: %v", d.namespace, d.name, err)
	}

	// Decode the config from the deployment.
	d.config, err = deployutil.DecodeDeploymentConfig(d.to, kapi.Codecs.UniversalDecoder())
	if err != nil {
		return fmt.Errorf("couldn't decode deployment config from deployment %s/%s: %v", d.to.Namespace, d.to.Name, err)
	}

	d.label = deployutil.LabelForDeployment(d.to)

	// Validation: e.g. new deployment has a desired replica count
	var hasDesired bool
	var desiredReplicas int32
	desiredReplicas, hasDesired = deployutil.DeploymentDesiredReplicas(d.to)
	if !hasDesired {
		return fmt.Errorf("deployment %s has no desired replica count", d.label)
	}
	d.desiredReplicas = int(desiredReplicas)

	d.desiredApp = convertToMarathonApplication(d.config.Spec.MarathonAppTemplate)
	if d.desiredApp == nil {
		return fmt.Errorf("deployment %s has no specified marathon application template in deployment config", d.label)
	}
	d.desiredApp.Instances = &d.desiredReplicas

	d.siteId = d.config.Spec.Site
	if len(d.siteId) == 0 {
		return fmt.Errorf("deployment %s has no specified site in deployment config", d.label)
	}

	// check app ID
	d.projId = d.namespace
	d.appId = d.desiredApp.ID
	if strings.HasPrefix(d.appId, "/") {
		// appId is an absolute path like "/projectid/appid"
		res := strings.Split(d.appId, "/")
		if len(res) != 3 {
			return fmt.Errorf("deployment %s aborted because invalid app id (%s) is specified in template of deployment config:len=%d,%v",
				d.label,
				d.appId,
				len(res),
				res)
		}
		if !strings.EqualFold(res[1], d.projId) {
			return fmt.Errorf("deployment %s aborted because project id (%s) does NOT match app id (%s) specified in template of deployment config",
				d.label,
				d.projId,
				d.appId)
		}

		d.appId = res[2]
	} else {
		// replace app's id with absolute path
		d.desiredApp.ID = "/" + d.projId + "/" + d.appId
	}

	d.sites = d.oclient.Sites(d.namespace)

	// use app ID to get existing application if deployed
	d.actualApp, err = d.sites.GetApplication(d.siteId, d.projId, d.appId)
	if err != nil {
		return fmt.Errorf("deployment %s aborted upon failure of checking existence of app (@%s:%s): %v", d.label, d.siteId, d.appId, err)
	}

	version, isScale := deployutil.DeploymentVersionOfMarathonScale(d.to)
	if isScale && version == d.config.Status.LatestVersion {
		return d.scale()
	}

	version, isReconcile := deployutil.DeploymentVersionOfMarathonReconcile(d.to)
	if isReconcile && version == d.config.Status.LatestVersion {
		return d.reconcile()
	}

	version, isRetry := deployutil.DeploymentVersionOfMarathonRetry(d.to)
	if isRetry && version == d.config.Status.LatestVersion {
		return d.retry()
	}

	return d.deploy(false)
}

func (d *Deployment) scale() error {
	glog.Infof("Scaling deployment %s gets started for app %s", d.label, d.appId)

	if d.actualApp == nil {
		return fmt.Errorf("deployment %s aborted because the application %s doesn't exist",
			d.label,
			d.appId)
	}

	if *d.actualApp.Instances == d.desiredReplicas {
		glog.Infof("Scaling deployment %s succeeded without doing anything because there're %d instances for app %s",
			d.label,
			d.desiredReplicas,
			d.appId)
		return nil
	}

	if d.actualApp.Deployments != nil && len(d.actualApp.Deployments) > 0 {
		return fmt.Errorf("deployment %s aborted because the application %s currently is in deployment: %v",
			d.label,
			d.appId,
			d.actualApp.Deployments)
	}

	deploy, err := d.sites.ScaleApplication(d.siteId, d.projId, d.appId, d.desiredReplicas)
	if err != nil {
		return fmt.Errorf("Scaling deployment %s failed to scale app %s and site return error as %v",
			d.label,
			d.appId,
			err)
	} else {
		glog.Infof("Scaling deployment %s has been conducted on site %s with deployment as (id, version, replicas) as (%s, %s, %d)",
			d.label,
			d.siteId,
			deploy.DeploymentID,
			deploy.Version,
			d.desiredReplicas)
	}

	// we wait until deployment gets done
	err = d.postMarathonDeploymentProcessing(deploy, "Scaling")

	if err != nil {
		return err
	} else {
		glog.Infof("Scaling deployment %s of app %s has been done on site %s successfully, and %d instances launched ",
			d.label,
			d.appId,
			d.siteId,
			d.desiredReplicas)
		return nil
	}
}

func (d *Deployment) retry() error {
	if kapi.Semantic.DeepDerivative(d.desiredApp, d.actualApp) {
		// the spec of desired app is the same as actual app
		glog.Infof("Retry of deployment %s continues for app %s ", d.label, d.appId)

		var deploy *marathon.DeploymentID
		if len(d.actualApp.Deployments) > 0 {
			deploymentToBeDeleted := d.actualApp.Deployments[0]
			deploymentIdToBeDeleted := deploymentToBeDeleted["id"]
			deploy = &marathon.DeploymentID{
				DeploymentID: deploymentIdToBeDeleted,
			}
		}
		return d.postMarathonDeploymentProcessing(deploy, "Retry")
	} else {
		// the spec of desired app is different from actual app, just start a new deployment
		glog.Infof("Retry of deployment %s gets started for app %s", d.label, d.appId)
		return d.deploy(false)
	}
}

func (d *Deployment) reconcile() error {

	glog.Infof("Reconcilation deployment %s gets started for app %s", d.label, d.appId)

	// Find all deployments for the config.
	unsortedDeployments, err := d.kclient.ReplicationControllers(d.namespace).List(kapi.ListOptions{LabelSelector: deployutil.ConfigSelector(d.config.Name)})
	if err != nil {
		return fmt.Errorf("couldn't get controllers in namespace %s: %v", d.namespace, err)
	}
	deployments := unsortedDeployments.Items

	// Sort all the deployments by version.
	sort.Sort(deployutil.ByLatestVersionDesc(deployments))

	// Find any last completed deployment.
	var from *kapi.ReplicationController
	for _, candidate := range deployments {
		if candidate.Name == d.to.Name {
			continue
		}
		if deployutil.DeploymentStatusFor(&candidate) == deployapi.DeploymentStatusComplete {
			from = &candidate
			break
		}
	}

	// if there's a completed app deployment, then rollback to it which is the current deployment
	// config set by deployment config controller
	// otherwise, just delete the existing marathon app deployment
	if from == nil {
		if d.actualApp == nil {
			glog.Infof("Reconcilation deployment %s needs to do nothing because the  app %s doesn't exist",
				d.label,
				d.appId)
		} else if len(d.actualApp.Deployments) > 0 {
			var deploy *marathon.DeploymentID
			deploymentToBeDeleted := d.actualApp.Deployments[0]
			deploymentIdToBeDeleted := deploymentToBeDeleted["id"]
			deploy, err = d.sites.DeleteDeployment(d.siteId, d.projId, deploymentIdToBeDeleted)
			if err != nil {
				return fmt.Errorf("Reconcilation deployment %s failed because of failure of rollback: %v",
					d.label,
					err)
			} else {
				d.postMarathonDeploymentProcessing(deploy, "Reconciling")
			}
			glog.Infof("Reconcilation deployment %s failed but those running marathon deployment of the app %s got deleted",
				d.label,
				d.appId)
		}

		// no matter whether deletion succeeds or not, alway failed this deployment because of nowhere to rollback
		return fmt.Errorf("Reconcilation deployment %s failed because the app %s has nowhere to rollback",
			d.label,
			d.appId)
	} else {
		return d.deploy(true)
	}
}

//
// TODO: mesos marathon scheduler currently support rolling update by default
//       we need to support recreating, A/B deployment and Blue/green deploymnet later
//
func (d *Deployment) deploy(force bool) error {
	var err error

	if d.actualApp == nil {
		// if the application doesn't exist, we call CreateApplication
		glog.Infof("Deploying %s for the first time (replicas: %d)", d.label, d.desiredReplicas)
		deployApp, err := d.sites.CreateApplication(d.siteId, d.projId, d.appId, d.desiredApp)
		if err != nil {
			return fmt.Errorf("deployment %s aborted upon failure of app creation request to marathon scheduler of : %v", d.label, err)
		} else {
			glog.Infof("Deployment %s has been conducted on site %s with creation of app as (id, version, replicas) as (%s, %s, %d)",
				d.label,
				d.siteId,
				deployApp.ID,
				deployApp.Version,
				d.desiredReplicas)
		}

		err = d.postMarathonDeploymentProcessing(nil, "Creating")
	} else {
		glog.Infof("Deployment %s for the existing app %s starts", d.label, d.appId)

		// if it exists,
		// i) check if the app is currently is in deployment, if yes, abort, otherwise continue
		// ii) call UpdateApplication according to the specified config
		if !force && len(d.actualApp.Deployments) > 0 {
			return fmt.Errorf("deployment %s aborted because the application currently is in deployment: %v",
				d.label,
				d.actualApp.Deployments)
		}

		deploy, err := d.sites.UpdateApplication(d.siteId, d.projId, d.appId, d.desiredApp, force)
		if err != nil {
			return fmt.Errorf("deployment %s failed  to update application %s and site return error as %v",
				d.label,
				d.appId,
				err)
		} else {
			glog.Infof("Deployment %s is updating app %s on site %s with marathon deployment as (id, version, replicas) as (%s, %s, %d)",
				d.label,
				d.appId,
				d.siteId,
				deploy.DeploymentID,
				deploy.Version,
				d.desiredReplicas)
		}
		err = d.postMarathonDeploymentProcessing(deploy, "Updating")
	}

	// we wait until deployment gets done
	if err != nil {
		return err
	} else {
		glog.Infof("Deployment %s of app %s has been done on site %s successfully, and %d instances launched ",
			d.label,
			d.appId,
			d.siteId,
			d.desiredReplicas)
		return nil
	}
}

func (d *Deployment) postMarathonDeploymentProcessing(deploy *marathon.DeploymentID, deploymentAction string) error {
	var err error

	if deploy != nil {
		// we wait until deployment gets done
		err = d.waitForDeployment(deploy.DeploymentID, determineTimeout(d.desiredApp))
		if err != nil {
			return fmt.Errorf("Deployment(%s) %s failed to wait until marathon deployment(%s) of application (/%s/%s) ok: %v",
				deploymentAction,
				d.label,
				deploy.DeploymentID,
				d.projId,
				d.appId,
				err)
		} else {
			glog.Infof("Deployment(%s) %s waits util marathon deployment (id, version, replicas) (%s, %s, %d) returns",
				deploymentAction,
				d.label,
				deploy.DeploymentID,
				deploy.Version,
				d.desiredReplicas)
		}
	}

	// we double check if there're desired number of marathon tasks running
	err = d.waitForApplication(determineTimeout(d.desiredApp))
	if err != nil {
		return fmt.Errorf("Deployment(%s) %s failed to wait until marathon application(/%s/%s) gets ready: %v",
			deploymentAction,
			d.label,
			d.projId,
			d.appId,
			err)
	} else {
		glog.Infof("Deployment(%s) %s succeeded waiting until marathon application(/%s/%s) get ready",
			deploymentAction,
			d.label,
			d.projId,
			d.appId)
	}
	return nil
}

func (d *Deployment) waitForApplication(timeout time.Duration) error {
	now := time.Now()
	stopTime := now.Add(timeout)

	for {
		if time.Now().After(stopTime) {
			return fmt.Errorf("timeout(%v) of waiting for application", timeout)
		}

		updatedActualApp, err := d.sites.GetApplication(d.siteId, d.projId, d.appId)
		if err == nil {
			if updatedActualApp == nil || isApplicationOk(updatedActualApp) {
				// if updatedActualApp is nil, it means that app got rollbacked to non-creation status
				// otherwise, we wait until application got to ok status
				elapsedStr := fmt.Sprintf("%0.2f sec(s)", time.Since(now).Seconds())
				glog.Infof("In deployment of %s, after elapsed time %s, app (/%s/%s) is good on site %s",
					d.label,
					elapsedStr,
					d.projId,
					d.appId,
					d.siteId)

				return nil
			}
		}
		time.Sleep(time.Duration(2) * time.Second)
	}
}

func isApplicationOk(app *marathon.Application) bool {
	// step: check if all the tasks are running?
	if !app.AllTaskRunning() {
		return false
	}

	// step: if the app has not health checks, just return true
	if app.HealthChecks == nil || len(*app.HealthChecks) == 0 {
		return true
	}

	// step: iterate the app checks and look for false
	for _, task := range app.Tasks {
		for _, check := range task.HealthCheckResults {
			//When a task is flapping in Marathon, this is sometimes nil
			if check == nil || !check.Alive {
				return false
			}
		}
	}

	return true
}

func (d *Deployment) waitForDeployment(deploymentId string, timeout time.Duration) error {

	now := time.Now()
	stopTime := now.Add(timeout)

	for {
		if time.Now().After(stopTime) {
			return fmt.Errorf("Deployment %s timeout(%v) while waiting for deployment", d.label, timeout)
		}

		found, err := d.sites.GetDeployment(d.siteId, d.projId, deploymentId)

		if err == nil && found == nil {
			// no error , and deployment has not been found, so it has been completed
			elapsedStr := fmt.Sprintf("%0.2f sec(s)", time.Since(now).Seconds())
			glog.Infof("Deployment %s, after elapsed time %s, gets the marathon deployment(%s) good on site %s",
				d.label,
				elapsedStr,
				deploymentId,
				d.siteId)

			return nil
		}
		time.Sleep(time.Duration(2) * time.Second)
	}
}

func determineTimeout(app *marathon.Application) time.Duration {
	if app == nil {
		return defaultTimeout
	}

	max := defaultTimeout

	if app.HealthChecks != nil && len(*app.HealthChecks) > 0 {
		for _, h := range *app.HealthChecks {
			grace := time.Duration(h.GracePeriodSeconds) * time.Second
			if grace > max {
				max = grace
			}
		}
		return max
	}
	return defaultTimeout
}

func convertToMarathonApplication(app *deployapi.MarathonApplication) *marathon.Application {
	result := new(marathon.Application)

	result.ID = app.ID
	result.Cmd = app.Cmd
	result.Args = &app.Args
	result.Constraints = convertToMarathonConstraints(app.Constraints)

	result.Container = convertToMarathonContainer(app.Container)

	result.CPUs = *app.CPUs
	result.Disk = app.Disk
	result.Env = &app.Env
	result.Executor = app.Executor

	result.HealthChecks = convertToMarathonHealthChecks(app.HealthChecks)

	result.Mem = app.Mem
	for _, p := range app.Ports {
		result.Ports = append(result.Ports, int(p))
	}
	result.RequirePorts = app.RequirePorts
	result.BackoffSeconds = app.BackoffSeconds
	result.BackoffFactor = app.BackoffFactor
	result.MaxLaunchDelaySeconds = app.MaxLaunchDelaySeconds
	result.Dependencies = app.Dependencies
	result.User = app.User

	result.UpgradeStrategy = convertToMarathonUpgradeStrategy(app.UpgradeStrategy)

	result.Uris = &app.Uris
	result.Labels = &app.Labels
	result.AcceptedResourceRoles = app.AcceptedResourceRoles

	result.Fetch = convertToMarathonFetch(app.Fetch)

	return result
}

func convertToMarathonConstraints(constraints []deployapi.MarathonConstraint) *[][]string {
	if len(constraints) == 0 {
		return nil
	}

	var result [][]string
	for _, c := range constraints {
		var cons []string
		for _, con := range c.Constraint {
			cons = append(cons, con)
		}
		result = append(result, cons)
	}

	return &result
}

func convertToMarathonContainer(container *deployapi.MarathonContainer) *marathon.Container {
	if container == nil {
		return nil
	}

	result := new(marathon.Container)

	result.Type = container.Type
	result.Docker = convertToMarathonDocker(container.Docker)
	result.Volumes = convertToMarathonVolumes(container.Volumes)

	return result
}

func convertToMarathonHealthChecks(healthChecks []deployapi.MarathonHealthCheck) *[]marathon.HealthCheck {
	if len(healthChecks) == 0 {
		return nil
	}

	var result []marathon.HealthCheck
	for _, check := range healthChecks {
		var command *marathon.Command = nil
		if check.Command != nil {
			command = &marathon.Command{
				Value: *check.Command,
			}
		}
		var portIndex *int
		if check.PortIndex != nil {
			t := int(*check.PortIndex)
			portIndex = &t
		}
		var maxConsecutiveFailures *int
		if check.MaxConsecutiveFailures != nil {
			t := int(*check.MaxConsecutiveFailures)
			maxConsecutiveFailures = &t
		}
		resCheck := marathon.HealthCheck{
			Command:   command,
			PortIndex: portIndex,
			Path:      check.Path,
			MaxConsecutiveFailures: maxConsecutiveFailures,
			Protocol:               check.Protocol,
			GracePeriodSeconds:     int(check.GracePeriodSeconds),
			IntervalSeconds:        int(check.IntervalSeconds),
			TimeoutSeconds:         int(check.TimeoutSeconds),
		}
		result = append(result, resCheck)
	}
	return &result
}

func convertToMarathonUpgradeStrategy(strategy *deployapi.MarathonUpgradeStrategy) *marathon.UpgradeStrategy {
	if strategy == nil {
		return nil
	}

	result := &marathon.UpgradeStrategy{
		MinimumHealthCapacity: strategy.MinimumHealthCapacity,
		MaximumOverCapacity:   strategy.MaximumOverCapacity,
	}
	return result
}

func convertToMarathonFetch(fetchs []deployapi.MarathonFetch) []marathon.Fetch {
	var result []marathon.Fetch

	for _, f := range fetchs {
		resFetch := marathon.Fetch{
			URI:        f.URI,
			Executable: f.Executable,
			Extract:    f.Extract,
			Cache:      f.Cache,
		}
		result = append(result, resFetch)
	}
	return result
}

func convertToMarathonDocker(docker *deployapi.MarathonDocker) *marathon.Docker {
	if docker == nil {
		return nil
	}

	result := new(marathon.Docker)

	result.ForcePullImage = docker.ForcePullImage
	result.Image = docker.Image
	result.Network = docker.Network
	result.Parameters = convertToMarathonParameters(docker.Parameters)
	result.PortMappings = convertToMarathonPortMapping(docker.PortMappings)
	result.Privileged = docker.Privileged
	return result
}

func convertToMarathonParameters(paras []deployapi.MarathonParameters) *[]marathon.Parameters {
	if len(paras) == 0 {
		return nil
	}

	var result []marathon.Parameters

	for _, p := range paras {
		resPara := marathon.Parameters{
			Key:   p.Key,
			Value: p.Value,
		}
		result = append(result, resPara)
	}
	return &result
}

func convertToMarathonVolumes(volumes []deployapi.MarathonVolume) *[]marathon.Volume {
	if len(volumes) == 0 {
		return nil
	}

	var result []marathon.Volume

	for _, v := range volumes {
		resVol := marathon.Volume{
			ContainerPath: v.ContainerPath,
			HostPath:      v.HostPath,
			Mode:          v.Mode,
		}
		result = append(result, resVol)
	}
	return &result
}

func convertToMarathonPortMapping(portMaps []deployapi.MarathonPortMapping) *[]marathon.PortMapping {
	if len(portMaps) == 0 {
		return nil
	}

	var result []marathon.PortMapping

	for _, pm := range portMaps {
		resPortMapping := marathon.PortMapping{
			ContainerPort: int(pm.ContainerPort),
			HostPort:      int(pm.HostPort),
			ServicePort:   int(pm.ServicePort),
			Protocol:      pm.Protocol,
		}
		result = append(result, resPortMapping)
	}
	return &result
}
