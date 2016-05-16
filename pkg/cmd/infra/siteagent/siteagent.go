package siteagent

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	cmdutil "k8s.io/kubernetes/pkg/kubectl/cmd/util"

	"github.com/openshift/origin/pkg/cmd/util"
	"github.com/openshift/origin/pkg/cmd/util/clientcmd"
	"github.com/openshift/origin/pkg/version"
)

const (
	siteAgentLong = `
Start an mesos-marathon site agent

This command launches a process that will talk to a site managed by a mesos-marathon cluster manager. 

You may restrict the set of DeploymentConfigs exposed to a single project (with --namespace), projects your client has
access to with a set of labels (--project-labels), namespaces matching a label (--namespace-labels), or all
namespaces (no argument). You can limit the sites to those matching a --labels or --fields selector. Note
that you must have a cluster-wide administrative role to view all namespaces.`
)

// MarathonSiteAgentOptions represent the complete structure needed to start an mesos-marathon site agent
type MarathonSiteAgentOptions struct {
	Config *clientcmd.Config

	MarathonSiteAgent
}

// MarathonSiteAgent is the config necessary to start a site agent.
type MarathonSiteAgent struct {
	// Name of the site agent
	Name string

	// Port exposed by site agent to provide http restful service
	Port string

	// Username specifies the username with which the plugin should authenticate
	// with the Mesos Marathon scheduler.
	Username string

	// Password specifies the password with which the plugin should authenticate
	// with the Mesos Marathon scheduler.
	Password string

	// HttpVserver specifies the name of the server in Mesos Marathon scheduler
	// that the plugin will configure for HTTP connections.
	HttpVserver string

	// HttpsVserver specifies the name of the server in Mesos Marathon scheduler
	// that the plugin will configure for HTTPS connections.
	HttpsVserver string

	// PrivateKey specifies the filename of an private key for
	// authenticating with Mesos Marathon scheduler.  This key is required to copy
	// certificates to the Mesos Marathon scheduler.
	PrivateKey string

	// Insecure specifies whether the Mesos Marathon scheduler plugin should perform
	// strict certificate validation for connections to the Mesos Marathon scheduler.
	Insecure bool
}

// Bind binds MarathonSiteAgent arguments to flags
func (o *MarathonSiteAgent) Bind(flag *pflag.FlagSet) {
	flag.StringVar(&o.Name, "name", util.Env("SITEAGENT_SERVICE_NAME", "public"), "The name the siteagent will identify itself with")
	flag.StringVar(&o.Port, "port", util.Env("SITEAGENT_SERVICE_HTTP_PORT", ":80"), "The port of Mesos marathon site agent")
	flag.StringVar(&o.HttpsVserver, "marathon-https-vserver", util.Env("SITEAGENT_EXTERNAL_HOST_HTTPS_VSERVER", "https-ose-vserver"), "The endpoint url of Mesos marathon scheduler for HTTPS connections")
	flag.StringVar(&o.Username, "marathon-username", util.Env("SITEAGENT_EXTERNAL_HOST_USERNAME", ""), "The username for mesos marathon scheduler's management utility")
	flag.StringVar(&o.Password, "marathon-password", util.Env("SITEAGENT_EXTERNAL_HOST_PASSWORD", ""), "The password for mesos marathon scheduler's management utility")
	flag.StringVar(&o.PrivateKey, "marathon-private-key", util.Env("SITEAGENT_EXTERNAL_HOST_PRIVKEY", ""), "The path to the Mesos Marathon scheduler private key file")
	flag.BoolVar(&o.Insecure, "marathon-insecure", util.Env("SITEAGENT_EXTERNAL_HOST_INSECURE", "") == "true", "Skip strict certificate verification")
}

// Validate verifies the required Marathon flags are present
func (o *MarathonSiteAgent) Validate() error {
	if o.Port == "" {
		return errors.New("Mesos-marathon site agent must be specified with port to listen to")
	}

	if o.Username == "" {
		return errors.New("Marathon username must be specified")
	}

	if o.Password == "" {
		return errors.New("Marathon password must be specified")
	}

	if len(o.HttpsVserver) == 0 {
		return errors.New("Marathon HTTPS vservers cannot be blank")
	}

	return nil
}

// NewCommandSiteAgent provides CLI handler for the siteagent.
func NewCommandSiteAgent(name string) *cobra.Command {
	options := &MarathonSiteAgentOptions{
		Config: clientcmd.NewConfig(),
	}
	options.Config.FromFile = true

	cmd := &cobra.Command{
		Use:   fmt.Sprintf("%s%s", name, clientcmd.ConfigSyntax),
		Short: "Start an Marathon site agent",
		Long:  siteAgentLong,
		Run: func(c *cobra.Command, args []string) {
			cmdutil.CheckErr(options.Complete())
			cmdutil.CheckErr(options.Validate())
			cmdutil.CheckErr(options.Run())
		},
	}

	cmd.AddCommand(version.NewVersionCommand(name, false))

	flag := cmd.Flags()
	options.Config.Bind(flag)
	options.MarathonSiteAgent.Bind(flag)

	return cmd
}

func (o *MarathonSiteAgentOptions) Complete() error {
	return nil
}

func (o *MarathonSiteAgentOptions) Validate() error {
	return o.MarathonSiteAgent.Validate()
}

// Run launches a site agent process using the provided options. It never exits.
func (o *MarathonSiteAgentOptions) Run() error {
	cfg := &ServerConfig{
		SiteName: o.Name,
		Logging:  true,

		MarathonAuthEnabled: true,
		MarathonURL:         o.HttpsVserver,
		MarathonUsername:    o.Username,
		MarathonPassword:    o.Password,
	}
	var ports []string

	ports = append(ports, o.Port)
	svr := New(cfg)
	svr.ServeApi(ports)

	return nil
}
