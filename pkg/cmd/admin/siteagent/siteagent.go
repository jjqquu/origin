package siteagent

import (
	"fmt"
	"io"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/spf13/cobra"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/api/errors"
	"k8s.io/kubernetes/pkg/client/restclient"
	kclient "k8s.io/kubernetes/pkg/client/unversioned"
	kclientcmd "k8s.io/kubernetes/pkg/client/unversioned/clientcmd"
	kcmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"
	"k8s.io/kubernetes/pkg/runtime"
	"k8s.io/kubernetes/pkg/util/intstr"

	authapi "github.com/openshift/origin/pkg/authorization/api"
	cmdutil "github.com/openshift/origin/pkg/cmd/util"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/openshift/origin/pkg/cmd/util/variable"
	configcmd "github.com/openshift/origin/pkg/config/cmd"
	deployapi "github.com/openshift/origin/pkg/deploy/api"
	"github.com/openshift/origin/pkg/generate/app"
	fileutil "github.com/openshift/origin/pkg/util/file"
)

const (
	siteagentLong = `
Install or configure a site agent

This command helps to setup a site agent to talk to the cluster manager for the specified site where
your application runs. With no arguments, the command will check for an existing site agent
service called 'siteagent' and create one if it does not exist. If you want to test whether
a siteagent has already been created add the --dry-run flag and the command will exit with
1 if the registry does not exist.

If a siteagent does not exist with the given name, this command will
create a deployment configuration and service that will run the siteagent. If you are
running your siteagent in production, you should pass --replicas=2 or higher to ensure
you have failover protection.`

	siteagentExample = `  # Check the default siteagent ("siteagent")
  $ %[1]s %[2]s --dry-run

  # See what the siteagent would look like if created
  $ %[1]s %[2]s -o json --credentials=/path/to/openshift-siteagent.kubeconfig --service-account=myserviceaccount

  # Create a siteagent if it does not exist
  $ %[1]s %[2]s siteagent-west --credentials=/path/to/openshift-siteagent.kubeconfig --service-account=myserviceaccount --replicas=2

  # Use a different siteagent image and see the siteagent configuration
  $ %[1]s %[2]s region-west -o yaml --credentials=/path/to/openshift-siteagent.kubeconfig --service-account=myserviceaccount --images=myrepo/somesiteagent:mytag
  `

	secretsVolumeName = "secret-volume"
	secretsPath       = "/etc/secret-volume"

	// this is the official private certificate path on Red Hat distros, and is at least structurally more
	// correct than ubuntu based distributions which don't distinguish between public and private certs.
	// Since Origin is CentOS based this is more likely to work.  Ubuntu images should symlink this directory
	// into /etc/ssl/certs to be compatible.
	defaultCertificateDir = "/etc/pki/tls/private"

	privkeySecretName = "external-host-private-key-secret"
	privkeyVolumeName = "external-host-private-key-volume"
	privkeyName       = "siteagent.pem"
	privkeyPath       = secretsPath + "/" + privkeyName
)

var defaultCertificatePath = path.Join(defaultCertificateDir, "tls.crt")

// SiteAgentConfig contains the configuration parameters necessary to
// launch a siteagent, including general parameters, type of siteagent, and
// type-specific parameters.
type SiteAgentConfig struct {
	// Name is the siteagent name, set as an argument
	Name string

	// Type is the siteagent type, which determines which plugin to use (f5
	// or template).
	Type string

	// ImageTemplate specifies the image from which the siteagent will be created.
	ImageTemplate variable.ImageTemplate

	// Ports specifies the container ports for the siteagent.
	Ports string

	// Replicas specifies the initial replica count for the siteagent.
	Replicas int

	// Labels specifies the label or labels that will be assigned to the siteagent
	// pod.
	Labels string

	// DryRun specifies that the siteagent command should not launch a siteagent but
	// should instead exit with code 1 to indicate if a siteagent is already running
	// or code 0 otherwise.
	DryRun bool

	// SecretsAsEnv sets the credentials as env vars, instead of secrets.
	SecretsAsEnv bool

	// Credentials specifies the path to a .kubeconfig file with the credentials
	// with which the siteagent may contact the master.
	Credentials string

	// DefaultCertificate holds the certificate that will be used if no more
	// specific certificate is found.  This is typically a wildcard certificate.
	DefaultCertificate string

	// Selector specifies a label or set of labels that determines the nodes on
	// which the siteagent pod can be scheduled.
	Selector string

	// ServiceAccount specifies the service account under which the siteagent will
	// run.
	ServiceAccount string

	// ExternalHostUsername specifies the username for authenticating with the
	// external host.
	ExternalHostUsername string

	// ExternalHostPassword specifies the password for authenticating with the
	// external host.
	ExternalHostPassword string

	// ExternalHostHttpsVserver specifies the virtual server for HTTPS connections.
	ExternalHostHttpsVserver string

	// ExternalHostPrivateKey specifies an SSH private key for authenticating with
	// the external host.
	ExternalHostPrivateKey string

	// ExternalHostInsecure specifies that the siteagent should skip strict
	// certificate verification when connecting to the external host.
	ExternalHostInsecure bool
}

var errExit = fmt.Errorf("exit")

const (
	defaultLabel = "siteagent=default"

	// Default port numbers to expose and bind/listen on.
	defaultPort = 80
)

// NewCmdSiteAgent implements the OpenShift CLI siteagent command.
func NewCmdSiteAgent(f *clientcmd.Factory, parentName, name string, out io.Writer) *cobra.Command {
	cfg := &SiteAgentConfig{
		Name:           "siteagent",
		ImageTemplate:  variable.NewDefaultImageTemplate(),
		ServiceAccount: "siteagent",
		Labels:         defaultLabel,
		Ports:          strconv.Itoa(defaultPort),
		Replicas:       1,
	}

	cmd := &cobra.Command{
		Use:     fmt.Sprintf("%s [NAME]", name),
		Short:   "Install a site agent",
		Long:    siteagentLong,
		Example: fmt.Sprintf(siteagentExample, parentName, name),
		Run: func(cmd *cobra.Command, args []string) {
			err := RunCmdSiteAgent(f, cmd, out, cfg, args)
			if err != errExit {
				kcmdutil.CheckErr(err)
			} else {
				os.Exit(1)
			}
		},
	}

	cmd.Flags().StringVar(&cfg.Type, "type", "mesos-marathon", "The type of siteagent to use - if you specify --images this flag may be ignored.")
	cmd.Flags().StringVar(&cfg.ImageTemplate.Format, "images", cfg.ImageTemplate.Format, "The image to base this siteagent on - ${component} will be replaced with --type")
	cmd.Flags().BoolVar(&cfg.ImageTemplate.Latest, "latest-images", cfg.ImageTemplate.Latest, "If true, attempt to use the latest images for the siteagent instead of the latest release.")
	cmd.Flags().StringVar(&cfg.Ports, "ports", cfg.Ports, fmt.Sprintf("A comma delimited list of ports or port pairs to expose on the siteagent pod. Port pairs are applied to the service. The default is set for %d.", defaultPort))
	cmd.Flags().IntVar(&cfg.Replicas, "replicas", cfg.Replicas, "The replication factor of the siteagent; commonly 2 when high availability is desired.")
	cmd.Flags().StringVar(&cfg.Labels, "labels", cfg.Labels, "A set of labels to uniquely identify the siteagent and its components.")
	cmd.Flags().BoolVar(&cfg.DryRun, "dry-run", cfg.DryRun, "Exit with code 1 if the specified siteagent does not exist.")
	cmd.Flags().BoolVar(&cfg.SecretsAsEnv, "secrets-as-env", cfg.SecretsAsEnv, "Use environment variables for master secrets.")
	cmd.Flags().StringVar(&cfg.Credentials, "credentials", "", "Path to a .kubeconfig file that will contain the credentials the siteagent should use to contact the master.")
	cmd.Flags().StringVar(&cfg.DefaultCertificate, "default-cert", cfg.DefaultCertificate, "Optional path to a certificate file that be used as the default certificate.  The file should contain the cert, key, and any CA certs necessary for the siteagent to serve the certificate.")
	cmd.Flags().StringVar(&cfg.Selector, "selector", cfg.Selector, "Selector used to filter nodes on deployment. Used to run siteagents on a specific set of nodes.")
	cmd.Flags().StringVar(&cfg.ServiceAccount, "service-account", cfg.ServiceAccount, "Name of the service account to use to run the siteagent pod.")
	cmd.Flags().StringVar(&cfg.ExternalHostUsername, "external-host-username", cfg.ExternalHostUsername, "If the underlying siteagent implementation connects with an external host, this is the username for authenticating with the external host.")
	cmd.Flags().StringVar(&cfg.ExternalHostPassword, "external-host-password", cfg.ExternalHostPassword, "If the underlying siteagent implementation connects with an external host, this is the password for authenticating with the external host.")
	cmd.Flags().StringVar(&cfg.ExternalHostHttpsVserver, "external-host-https-vserver", cfg.ExternalHostHttpsVserver, "If the underlying siteagent implementation uses virtual servers, this is the name of the virtual server for HTTPS connections.")
	cmd.Flags().StringVar(&cfg.ExternalHostPrivateKey, "external-host-private-key", cfg.ExternalHostPrivateKey, "If the underlying siteagent implementation requires an SSH private key, this is the path to the private key file.")
	cmd.Flags().BoolVar(&cfg.ExternalHostInsecure, "external-host-insecure", cfg.ExternalHostInsecure, "If the underlying siteagent implementation connects with an external host over a secure connection, this causes the siteagent to skip strict certificate verification with the external host.")

	cmd.MarkFlagFilename("credentials", "kubeconfig")

	kcmdutil.AddPrinterFlags(cmd)

	return cmd
}

// generateSecretsConfig generates any Secret and Volume objects, such
// as SSH private keys, that are necessary for the siteagent container.
func generateSecretsConfig(cfg *SiteAgentConfig, kClient *kclient.Client,
	namespace string, defaultCert []byte) ([]*kapi.Secret, []kapi.Volume, []kapi.VolumeMount,
	error) {
	var secrets []*kapi.Secret
	var volumes []kapi.Volume
	var mounts []kapi.VolumeMount

	if len(cfg.ExternalHostPrivateKey) != 0 {
		privkeyData, err := fileutil.LoadData(cfg.ExternalHostPrivateKey)
		if err != nil {
			return secrets, volumes, mounts, fmt.Errorf("error reading private key for external host: %v", err)
		}

		secret := &kapi.Secret{
			ObjectMeta: kapi.ObjectMeta{
				Name: privkeySecretName,
			},
			Data: map[string][]byte{privkeyName: privkeyData},
		}
		secrets = append(secrets, secret)

		volume := kapi.Volume{
			Name: secretsVolumeName,
			VolumeSource: kapi.VolumeSource{
				Secret: &kapi.SecretVolumeSource{
					SecretName: privkeySecretName,
				},
			},
		}
		volumes = append(volumes, volume)

		mount := kapi.VolumeMount{
			Name:      secretsVolumeName,
			ReadOnly:  true,
			MountPath: secretsPath,
		}
		mounts = append(mounts, mount)
	}

	if len(defaultCert) > 0 {
		keys, err := cmdutil.PrivateKeysFromPEM(defaultCert)
		if err != nil {
			return nil, nil, nil, err
		}
		if len(keys) == 0 {
			return nil, nil, nil, fmt.Errorf("the default cert must contain a private key")
		}
		secret := &kapi.Secret{
			ObjectMeta: kapi.ObjectMeta{
				Name: fmt.Sprintf("%s-certs", cfg.Name),
			},
			Type: kapi.SecretTypeTLS,
			Data: map[string][]byte{
				kapi.TLSCertKey:       defaultCert,
				kapi.TLSPrivateKeyKey: keys,
			},
		}
		secrets = append(secrets, secret)
		volume := kapi.Volume{
			Name: "server-certificate",
			VolumeSource: kapi.VolumeSource{
				Secret: &kapi.SecretVolumeSource{
					SecretName: secret.Name,
				},
			},
		}
		volumes = append(volumes, volume)

		mount := kapi.VolumeMount{
			Name:      volume.Name,
			ReadOnly:  true,
			MountPath: defaultCertificateDir,
		}
		mounts = append(mounts, mount)
	}

	return secrets, volumes, mounts, nil
}

func generateProbeConfigForSiteAgent(cfg *SiteAgentConfig, ports []kapi.ContainerPort) *kapi.Probe {
	var probe *kapi.Probe

	probe = &kapi.Probe{}

	healthzPort := defaultPort
	if len(ports) > 0 {
		healthzPort = ports[0].ContainerPort
	}

	probe.Handler.HTTPGet = &kapi.HTTPGetAction{
		Path: "/ping",
		Port: intstr.FromInt(healthzPort),
	}

	return probe
}

func generateLivenessProbeConfig(cfg *SiteAgentConfig, ports []kapi.ContainerPort) *kapi.Probe {
	probe := generateProbeConfigForSiteAgent(cfg, ports)
	if probe != nil {
		probe.InitialDelaySeconds = 10
		probe.TimeoutSeconds = 5
	}
	return probe
}

func generateReadinessProbeConfig(cfg *SiteAgentConfig, ports []kapi.ContainerPort) *kapi.Probe {
	probe := generateProbeConfigForSiteAgent(cfg, ports)
	if probe != nil {
		probe.TimeoutSeconds = 5
	}
	return probe
}

// RunCmdSiteAgent contains all the necessary functionality for the
// OpenShift CLI siteagent command.
func RunCmdSiteAgent(f *clientcmd.Factory, cmd *cobra.Command, out io.Writer, cfg *SiteAgentConfig, args []string) error {
	switch len(args) {
	case 0:
		// uses default value
	case 1:
		cfg.Name = args[0]
	default:
		return kcmdutil.UsageError(cmd, "You may pass zero or one arguments to provide a name for the siteagent")
	}
	name := cfg.Name

	ports, err := app.ContainerPortsFromString(cfg.Ports)
	if err != nil {
		return fmt.Errorf("unable to parse --ports: %v", err)
	}

	label := map[string]string{"siteagent": name}
	if cfg.Labels != defaultLabel {
		valid, remove, err := app.LabelsFromSpec(strings.Split(cfg.Labels, ","))
		if err != nil {
			glog.Fatal(err)
		}
		if len(remove) > 0 {
			return kcmdutil.UsageError(cmd, "You may not pass negative labels in %q", cfg.Labels)
		}
		label = valid
	}

	nodeSelector := map[string]string{}
	if len(cfg.Selector) > 0 {
		valid, remove, err := app.LabelsFromSpec(strings.Split(cfg.Selector, ","))
		if err != nil {
			glog.Fatal(err)
		}
		if len(remove) > 0 {
			return kcmdutil.UsageError(cmd, "You may not pass negative labels in selector %q", cfg.Selector)
		}
		nodeSelector = valid
	}

	image := cfg.ImageTemplate.ExpandOrDie(cfg.Type)

	namespace, _, err := f.OpenShiftClientConfig.Namespace()
	if err != nil {
		return fmt.Errorf("error getting client: %v", err)
	}
	_, kClient, err := f.Clients()
	if err != nil {
		return fmt.Errorf("error getting client: %v", err)
	}

	_, output, err := kcmdutil.PrinterForCommand(cmd)
	if err != nil {
		return fmt.Errorf("unable to configure printer: %v", err)
	}

	generate := output
	if !generate {
		_, err = kClient.Services(namespace).Get(name)
		if err != nil {
			if !errors.IsNotFound(err) {
				return fmt.Errorf("can't check for existing siteagent %q: %v", name, err)
			}
			generate = true
		}
	}
	if !generate {
		fmt.Fprintf(out, "SiteAgent %q service exists\n", name)
		return nil
	}

	if cfg.DryRun && !output {
		return fmt.Errorf("Siteagent %q does not exist (no service)", name)
	}

	if len(cfg.ServiceAccount) == 0 {
		return fmt.Errorf("you must specify a service account for the siteagent with --service-account")
	}

	// create new siteagent
	secretEnv := app.Environment{}
	switch {
	case len(cfg.Credentials) == 0 && len(cfg.ServiceAccount) == 0:
		return fmt.Errorf("siteagent could not be created; you must specify a .kubeconfig file path containing credentials for connecting the siteagent to the master with --credentials")
	case len(cfg.Credentials) > 0:
		clientConfigLoadingRules := &kclientcmd.ClientConfigLoadingRules{ExplicitPath: cfg.Credentials, Precedence: []string{}}
		credentials, err := clientConfigLoadingRules.Load()
		if err != nil {
			return fmt.Errorf("siteagent could not be created; the provided credentials %q could not be loaded: %v", cfg.Credentials, err)
		}
		config, err := kclientcmd.NewDefaultClientConfig(*credentials, &kclientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			return fmt.Errorf("siteagent could not be created; the provided credentials %q could not be used: %v", cfg.Credentials, err)
		}
		if err := restclient.LoadTLSFiles(config); err != nil {
			return fmt.Errorf("siteagent could not be created; the provided credentials %q could not load certificate info: %v", cfg.Credentials, err)
		}
		insecure := "false"
		if config.Insecure {
			insecure = "true"
		}
		secretEnv.Add(app.Environment{
			"OPENSHIFT_MASTER":    config.Host,
			"OPENSHIFT_CA_DATA":   string(config.CAData),
			"OPENSHIFT_KEY_DATA":  string(config.KeyData),
			"OPENSHIFT_CERT_DATA": string(config.CertData),
			"OPENSHIFT_INSECURE":  insecure,
		})
	}
	createServiceAccount := len(cfg.ServiceAccount) > 0 && len(cfg.Credentials) == 0

	defaultCert, err := fileutil.LoadData(cfg.DefaultCertificate)
	if err != nil {
		return fmt.Errorf("siteagent could not be created; error reading default certificate file: %v", err)
	}

	env := app.Environment{
		"SITEAGENT_SERVICE_NAME":                name,
		"SITEAGENT_SERVICE_HTTP_PORT":           fmt.Sprintf(":%d", ports[0].ContainerPort),
		"SITEAGENT_EXTERNAL_HOST_HTTPS_VSERVER": cfg.ExternalHostHttpsVserver,
		"SITEAGENT_EXTERNAL_HOST_USERNAME":      cfg.ExternalHostUsername,
		"SITEAGENT_EXTERNAL_HOST_PASSWORD":      cfg.ExternalHostPassword,
		"SITEAGENT_EXTERNAL_HOST_INSECURE":      strconv.FormatBool(cfg.ExternalHostInsecure),
		"SITEAGENT_EXTERNAL_HOST_PRIVKEY":       privkeyPath,
	}
	env.Add(secretEnv)
	if len(defaultCert) > 0 {
		if cfg.SecretsAsEnv {
			env.Add(app.Environment{"DEFAULT_CERTIFICATE": string(defaultCert)})
		} else {
			// TODO: make --credentials create secrets and bypass service account
			env.Add(app.Environment{"DEFAULT_CERTIFICATE_PATH": defaultCertificatePath})
		}
	}

	secrets, volumes, mounts, err := generateSecretsConfig(cfg, kClient, namespace, defaultCert)
	if err != nil {
		return fmt.Errorf("siteagent could not be created: %v", err)
	}

	livenessProbe := generateLivenessProbeConfig(cfg, ports)
	readinessProbe := generateReadinessProbeConfig(cfg, ports)

	exposedPorts := make([]kapi.ContainerPort, len(ports))
	copy(exposedPorts, ports)
	for i := range exposedPorts {
		exposedPorts[i].HostPort = 0
	}
	containers := []kapi.Container{
		{
			Name:            "siteagent-" + name,
			Image:           image,
			Ports:           exposedPorts,
			Env:             env.List(),
			LivenessProbe:   livenessProbe,
			ReadinessProbe:  readinessProbe,
			ImagePullPolicy: kapi.PullIfNotPresent,
			VolumeMounts:    mounts,
		},
	}

	objects := []runtime.Object{}
	for _, s := range secrets {
		objects = append(objects, s)
	}
	if createServiceAccount {
		objects = append(objects,
			&kapi.ServiceAccount{ObjectMeta: kapi.ObjectMeta{Name: cfg.ServiceAccount}},
			&authapi.ClusterRoleBinding{
				ObjectMeta: kapi.ObjectMeta{Name: fmt.Sprintf("siteagent-%s-role", cfg.Name)},
				Subjects: []kapi.ObjectReference{
					{
						Kind:      "ServiceAccount",
						Name:      cfg.ServiceAccount,
						Namespace: namespace,
					},
				},
				RoleRef: kapi.ObjectReference{
					Kind: "ClusterRole",
					// TODO: we need to define a siteagent role later
					//       Name: "system:siteagent",
					Name: "system:router",
				},
			},
		)
	}
	updatePercent := int(-25)
	objects = append(objects, &deployapi.DeploymentConfig{
		ObjectMeta: kapi.ObjectMeta{
			Name:   name,
			Labels: label,
		},
		Spec: deployapi.DeploymentConfigSpec{
			Strategy: deployapi.DeploymentStrategy{
				Type:          deployapi.DeploymentStrategyTypeRolling,
				RollingParams: &deployapi.RollingDeploymentStrategyParams{UpdatePercent: &updatePercent},
			},
			Replicas: cfg.Replicas,
			Selector: label,
			Triggers: []deployapi.DeploymentTriggerPolicy{
				{Type: deployapi.DeploymentTriggerOnConfigChange},
			},
			Template: &kapi.PodTemplateSpec{
				ObjectMeta: kapi.ObjectMeta{Labels: label},
				Spec: kapi.PodSpec{
					ServiceAccountName: cfg.ServiceAccount,
					NodeSelector:       nodeSelector,
					Containers:         containers,
					Volumes:            volumes,
				},
			},
		},
	})

	objects = app.AddServices(objects, false)
	// set the service port to the provided hostport value
	for i := range objects {
		switch t := objects[i].(type) {
		case *kapi.Service:
			for j, servicePort := range t.Spec.Ports {
				for _, targetPort := range ports {
					if targetPort.ContainerPort == servicePort.Port && targetPort.HostPort != 0 {
						t.Spec.Ports[j].Port = targetPort.HostPort
					}
				}
			}
		}
	}
	// TODO: label all created objects with the same label - siteagent=<name>
	list := &kapi.List{Items: objects}

	if output {
		fn := cmdutil.VersionedPrintObject(f.PrintObject, cmd, out)
		if err := fn(list); err != nil {
			return fmt.Errorf("unable to print object: %v", err)
		}
		return nil
	}

	mapper, typer := f.Factory.Object()
	bulk := configcmd.Bulk{
		Mapper:            mapper,
		Typer:             typer,
		RESTClientFactory: f.Factory.ClientForMapping,

		After: configcmd.NewPrintNameOrErrorAfter(mapper, kcmdutil.GetFlagString(cmd, "output") == "name", "created", out, cmd.Out()),
	}
	if errs := bulk.Create(list, namespace); len(errs) != 0 {
		return errExit
	}
	return nil
}
