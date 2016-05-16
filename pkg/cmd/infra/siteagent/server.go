package siteagent

import (
	"crypto/tls"
	"encoding/json"
	"expvar"
	"fmt"
	"io/ioutil"
	"mime"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"

	"github.com/golang/glog"
	"github.com/gorilla/mux"
	marathon "github.com/jjqquu/go_marathon"
	sconst "github.com/openshift/origin/pkg/site/siteagent"
)

/*
 * In order to make mesos/marathon multi-tenant, The server basically implements as proxy of underlying mesos
 * marathon scheduler with following enhancement:
 *  i) use marathon grouping as project isolation mechanism (namespace for resource visibility)
 *  ii) assume that marathon scheduler will implement quota (restriction for resource consumption)
 *  iii) improve security of only allowing project to touch its own application/tasks/deployments
 *
 * the restful server's API design:
 *  i) return HTTP_STATUS_OK
 *     if everything works well
 *  ii) return HTTP_STATUS_NOT_ACCEPTABLE
 *     if some failure happened in response to the invocation
 *  iii) return HTTP_STATUS_NOT_FOUND
 *     if the wanted resource object doesn't exist (only returned by getXXX, e.g getApplication/getProject/getDeployment/getTasks)
 * TODO:
 *  i) we need to make one single server support multi-site simutaneously
 *  ii) we need to check parity between centrolized control plane and underlying site: e.g. application created
 *      by site manually, rather than created by control plance here
 *      basic idea:
 *         i) add label to the app so that we can use the label to judage if it's create by control plane
 *         ii) the lable can be generated and hold by deploymentConfig for later correlation purpose
 *  iii) we need to add event subscription and use broadcast machanism to propogate it.
 *  iv) we need to add mesos api support to retrieve mesos master/slave information
 *
 */
type ServerConfig struct {
	SiteName string
	Logging  bool

	MarathonAuthEnabled bool
	MarathonURL         string
	MarathonUsername    string
	MarathonPassword    string

	TLSConfig *tls.Config
}

type MesosBackend struct {
	// TODO: add mesos client
	MarathonClient marathon.Marathon
}

type Server struct {
	cfg *ServerConfig

	backend *MesosBackend

	router  *mux.Router
	start   chan struct{}
	servers []serverCloser
}

func New(cfg *ServerConfig) *Server {
	srv := &Server{
		cfg:     cfg,
		backend: nil,
		start:   make(chan struct{}),
	}

	mClient := initMarathonClient(cfg)
	if mClient != nil {
		srv.backend = &MesosBackend{
			MarathonClient: mClient,
		}
	}

	r := createRouter(srv)
	srv.router = r
	return srv
}

func initMarathonClient(cfg *ServerConfig) marathon.Marathon {
	config := marathon.NewDefaultConfig()
	config.URL = cfg.MarathonURL

	//if os.Getenv("DEBUG") != "" {
	{
		tmpLogFile, _ := ioutil.TempFile("/tmp", "marathon.data")
		config.LogOutput = tmpLogFile
	}
	if cfg.MarathonAuthEnabled {
		config.HTTPBasicAuthUser = cfg.MarathonUsername
		config.HTTPBasicPassword = cfg.MarathonPassword
	}
	client, err := marathon.NewClient(config)
	if err != nil {
		glog.Fatalf("Failed to create a client for marathon, error: %s", err)
	}
	return client
}

func (s *Server) Close() {
	for _, srv := range s.servers {
		if err := srv.Close(); err != nil {
			glog.Error(err)
		}
	}
}

type serverCloser interface {
	Serve() error
	Close() error
}

// ServeApi loops through all of the protocols sent in to siteagent and spawns
// off a go routine to setup a serving http.Server for each.
func (s *Server) ServeApi(protoAddrs []string) error {
	var chErrors = make(chan error, len(protoAddrs))

	for _, protoAddr := range protoAddrs {
		srv, err := s.newServer(protoAddr)
		if err != nil {
			return err
		}
		s.servers = append(s.servers, srv)

		glog.Infof("Listening for HTTP on %s", protoAddr)
		go func(s serverCloser) {
			if err := s.Serve(); err != nil && strings.Contains(err.Error(), "use of closed network connection") {
				err = nil
			}
			chErrors <- err
		}(srv)
	}

	glog.Info("channel begins to work")
	for i := 0; i < len(protoAddrs); i++ {
		err := <-chErrors
		if err != nil {
			glog.Infof("channel failed to work because of %v", err)
			return err
		}
	}
	glog.Info("channel finishes working")

	return nil
}

type HttpServer struct {
	srv *http.Server
	l   net.Listener
}

func (s *HttpServer) Serve() error {
	glog.Info("HTTP is serving")
	return s.srv.Serve(s.l)
}
func (s *HttpServer) Close() error {
	return s.l.Close()
}

type HttpApiFunc func(w http.ResponseWriter, r *http.Request, vars map[string]string) error

func (s *Server) newServer(addr string) (serverCloser, error) {
	l, err := net.Listen("tcp", addr)
	if err != nil {
		glog.Errorf("failed to listen on a port: %v", err)
		return nil, err
	}
	res := &HttpServer{
		&http.Server{
			Addr:    addr,
			Handler: s.router,
		},
		l,
	}
	return res, nil
}

func matchesContentType(contentType, expectedType string) bool {
	mimetype, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		glog.Errorf("Error parsing media type: %s error: %v", contentType, err)
	}
	return err == nil && mimetype == expectedType
}

func checkForJson(r *http.Request) error {
	ct := r.Header.Get("Content-Type")

	// No Content-Type header is ok as long as there's no Body
	if ct == "" {
		if r.Body == nil || r.ContentLength == 0 {
			return nil
		}
	}

	// Otherwise it better be json
	if matchesContentType(ct, "application/json") {
		return nil
	}
	return fmt.Errorf("Content-Type specified (%s) must be 'application/json'", ct)
}

//If we don't do this, POST method without Content-type (even with empty body) will fail
func parseForm(r *http.Request) error {
	if r == nil {
		return nil
	}
	if err := r.ParseForm(); err != nil && !strings.HasPrefix(err.Error(), "mime:") {
		return err
	}
	return nil
}

func parseMultipartForm(r *http.Request) error {
	if err := r.ParseMultipartForm(4096); err != nil && !strings.HasPrefix(err.Error(), "mime:") {
		return err
	}
	return nil
}

func httpError(w http.ResponseWriter, err error) {
	if err == nil || w == nil {
		glog.Error("unexpected HTTP error handling")
		return
	}
	statusCode := http.StatusInternalServerError
	// FIXME: this is brittle and should not be necessary.
	// If we need to differentiate between different possible error types, we should
	// create appropriate error types with clearly defined meaning.
	errStr := strings.ToLower(err.Error())
	for keyword, status := range map[string]int{
		"not found":             http.StatusNotFound,
		"no such":               http.StatusNotFound,
		"bad parameter":         http.StatusBadRequest,
		"conflict":              http.StatusConflict,
		"impossible":            http.StatusNotAcceptable,
		"wrong login/password":  http.StatusUnauthorized,
		"hasn't been activated": http.StatusForbidden,
	} {
		if strings.Contains(errStr, keyword) {
			statusCode = status
			break
		}
	}

	glog.Error("HTTP Error")
	http.Error(w, err.Error(), statusCode)
}

// writeJSON writes the value v to the http response stream as json with standard
// json encoding.
func writeText(w http.ResponseWriter, code int, text string) error {
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(code)
	w.Write([]byte(text))
	return nil
}

// writeJSON writes the value v to the http response stream as json with standard
// json encoding.
func writeJSON(w http.ResponseWriter, code int, v interface{}) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	return json.NewEncoder(w).Encode(v)
}

func (s *Server) optionsHandler(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	w.WriteHeader(http.StatusOK)
	return nil
}

// "/ping":
func (s *Server) ping(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	w.Header().Set("Cache-Control", "no-cache")
	w.WriteHeader(http.StatusOK)
	return nil
}

// "/sites/":
func (s *Server) getSiteList(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	//TODO: we return sites list hereafter we listen to site
	err := fmt.Errorf("NOT implemented")
	return handleError(w, "getSiteList of site agent", err)
}

// "/sites/{site_id:.*}/info":
func (s *Server) getSiteInfo(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("getSiteInfo with(%v)", vars)
	if vars == nil {
		err := fmt.Errorf("parameter of siteid is required")
		return handleError(w, "postApplication of site agent", err)
	}

	info, err := s.backend.MarathonClient.Info()

	if err == nil {
		glog.Infof("s.backend.MarathonClient.Info() return info=%v)", *info)
		return writeJSON(w, http.StatusOK, *info)
	} else {
		return handleError(w, "getSiteInfo of site agent: failed to call remote Info()", err)
	}
}

// "/sites/{site_id:.*}/leader":
func (s *Server) getSiteLeader(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("getSiteLeader with(%v)", vars)
	if vars == nil {
		err := fmt.Errorf("parameter of siteid is required")
		return handleError(w, "postApplication of site agent", err)
	}

	leader, err := s.backend.MarathonClient.Leader()

	if err == nil {
		glog.Infof("s.backend.MarathonClient.Leader() return leader=%s)", leader)
		return writeJSON(w, http.StatusOK, leader)
	} else {
		return handleError(w, "getSiteLeader of site agent: failed to call remote Leader()", err)
	}
}

//"/sites/{site_id:.*}/projects/":
func (s *Server) getProjectList(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("getProjectList with(%v)", vars)

	if vars == nil {
		err := fmt.Errorf("parameter of siteid is required")
		return handleError(w, "getProjectList of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "getProjectList of site agent", err)
	}

	projects, err := s.backend.MarathonClient.Groups()

	if err == nil {
		glog.Infof("s.backend.MarathonClient.Group() return projects=%v)", *projects)
		return writeJSON(w, http.StatusOK, *projects)
	} else {
		return handleError(w, "getProjectList of site agent: failed to call remote Groups()", err)
	}
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/":
func (s *Server) getProject(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("getProject with(%v)", vars)

	if vars == nil {
		err := fmt.Errorf("parameter of siteid/projid are required")
		return handleError(w, "getProject of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "getProject of site agent", err)
	}

	projId := vars[sconst.ProjId]

	project, err := s.backend.MarathonClient.Group(projId)

	if err == nil {
		glog.Infof("s.backend.MarathonClient.Group(%s) return app=%v)", projId, *project)
		return writeJSON(w, http.StatusOK, *project)
	} else if apiErr, ok := err.(*marathon.APIError); ok && apiErr.ErrCode == marathon.ErrCodeNotFound {
		glog.Infof("getProject of site agent: not found")
		w.WriteHeader(http.StatusNotFound)
		return nil
	} else {
		return handleError(w, "getProject of site agent: failed to call remote Group()", err)
	}
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/quota":
func (s *Server) getProjectQuota(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	err := fmt.Errorf("NOT implemented")
	return handleError(w, "getProjectQuota of site agent", err)
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/applications/":
func (s *Server) getProjectApplicationList(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("getProjectApplicationList with(%v)", vars)

	if vars == nil {
		err := fmt.Errorf("parameter of siteid/projid are required")
		return handleError(w, "getProjectApplicationList of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "getProjectApplicationList of site agent", err)
	}

	projId := vars[sconst.ProjId]

	project, err := s.backend.MarathonClient.Group(projId)
	applications := project.Apps

	if err == nil {
		glog.Infof("s.backend.MarathonClient.Group(%s) return project with applications=%v)", projId, applications)
		return writeJSON(w, http.StatusOK, applications)
	} else if apiErr, ok := err.(*marathon.APIError); ok && apiErr.ErrCode == marathon.ErrCodeNotFound {
		glog.Infof("getProjectApplicationList of site agent: not found")
		w.WriteHeader(http.StatusNotFound)
		return nil
	} else {
		return handleError(w, "getProjectApplicationList of site agent: failed to call remote Group()", err)
	}
}

// "/sites/{site_id:.*}/applications/":
func (s *Server) getApplicationList(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("getApplicationList with(%v)", vars)

	if vars == nil {
		err := fmt.Errorf("parameter of siteid are required")
		return handleError(w, "getApplicationList of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "getApplicationList of site agent", err)
	}

	values := r.URL.Query()
	applications, err := s.backend.MarathonClient.Applications(values)

	if err == nil {
		glog.Infof("s.backend.MarathonClient.Applications(%v) return project with applications=%v)", values, applications)
		return writeJSON(w, http.StatusOK, applications)
	} else {
		return handleError(w, "getApplicationList of site agent: failed to call remote Applications()", err)
	}
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/applications/{app_id:.*}":
func (s *Server) getApplication(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("getApplication with(%v)", vars)
	if vars == nil {
		err := fmt.Errorf("parameter of siteid/projid/appid are required")
		return handleError(w, "getApplication of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "getApplication of site agent", err)
	}

	projId := vars[sconst.ProjId]
	appId := vars[sconst.AppId]

	glog.Infof("getApplication with(site=%s, proj=%s, app=%s)", siteId, projId, appId)

	appName := "/" + projId + "/" + appId

	app, err := s.backend.MarathonClient.Application(appName)
	if err == nil {
		glog.Infof("s.backend.MarathonClient.Application(%s) return app=%v)", appName, *app)
		return writeJSON(w, http.StatusOK, *app)
	} else if apiErr, ok := err.(*marathon.APIError); ok && apiErr.ErrCode == marathon.ErrCodeNotFound {
		glog.Infof("getApplication of site agent: not found")
		w.WriteHeader(http.StatusNotFound)
		return nil
	} else {
		return handleError(w, "getApplication of site agent: failed to call remote Application()", err)
	}
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/applications/{app_id:.*}/versions/{ver_id:.*}":
func (s *Server) getVersion(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("getVersion with(%v)", vars)
	if vars == nil {
		err := fmt.Errorf("parameter of siteid/projid/appid/versionid are required")
		return handleError(w, "getVersion of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "getVersion of site agent", err)
	}

	projId := vars[sconst.ProjId]
	appId := vars[sconst.AppId]
	verId := vars[sconst.VerId]

	appName := "/" + projId + "/" + appId

	appVersion, err := s.backend.MarathonClient.ApplicationByVersion(appName, verId)
	if err == nil {
		glog.Infof("s.backend.MarathonClient.ApplicationByVersion(%s) return appVersion=%v)", appName, *appVersion)
		return writeJSON(w, http.StatusOK, *appVersion)
	} else if apiErr, ok := err.(*marathon.APIError); ok && apiErr.ErrCode == marathon.ErrCodeNotFound {
		glog.Infof("getVersion of site agent: not found")
		w.WriteHeader(http.StatusNotFound)
		return nil
	} else {
		return handleError(w, "getVersion of site agent: failed to call remote ApplicationByVersion()", err)
	}
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/applications/{app_id:.*}/versions/":
func (s *Server) getVersionList(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("getVersionList with(%v)", vars)
	if vars == nil {
		err := fmt.Errorf("parameter of siteid/projid/appid are required")
		return handleError(w, "getVersionList of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "getVersionList of site agent", err)
	}

	projId := vars[sconst.ProjId]
	appId := vars[sconst.AppId]

	appName := "/" + projId + "/" + appId

	appVersions, err := s.backend.MarathonClient.ApplicationVersions(appName)
	if err == nil {
		glog.Infof("s.backend.MarathonClient.ApplicationVersions(%s) return appVersions=%v)", appName, *appVersions)
		return writeJSON(w, http.StatusOK, *appVersions)
	} else {
		return handleError(w, "getVersionList of site agent: failed to call remote ApplicationVersion()", err)
	}
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/applications/{app_id:.*}/tasks/":
func (s *Server) getTaskList(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("getTaskList with(%v)", vars)
	if vars == nil {
		err := fmt.Errorf("parameter of siteid/projid/appid are required")
		return handleError(w, "getTaskList of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "getTaskList of site agent", err)
	}

	projId := vars[sconst.ProjId]
	appId := vars[sconst.AppId]

	appName := "/" + projId + "/" + appId

	tasks, err := s.backend.MarathonClient.Tasks(appName)
	if err == nil {
		glog.Infof("s.backend.MarathonClient.Tasks(%s) return tasks=%v)", appName, *tasks)
		return writeJSON(w, http.StatusOK, *tasks)
	} else {
		return handleError(w, "getTaskList of site agent: failed to call remote Tasks()", err)
	}
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/deployments/":
func (s *Server) getProjectDeploymentList(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("getDeploymentList with(%v)", vars)
	if vars == nil {
		err := fmt.Errorf("parameter of siteid/projid are required")
		return handleError(w, "getProjectDeploymentList of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "getProjectDeploymentList of site agent", err)
	}

	projId := vars[sconst.ProjId]

	deployments, err := s.backend.MarathonClient.Deployments()
	if err != nil {
		return handleError(w, "getProjectDeploymentList of site agent: failed to call remote Deployments()", err)
	}

	glog.Infof("s.backend.MarathonClient.Deployment() return dps=%v)", deployments)
	projIdPrefix := "/" + projId + "/"
	var result []*marathon.Deployment
	for _, dp := range deployments {
		var isAppsDeployment = true
		for _, appId := range dp.AffectedApps {
			// scan to see if all affected app belongs to the specified project
			if !strings.HasPrefix(appId, projIdPrefix) {
				isAppsDeployment = false
				break
			}
		}
		if isAppsDeployment {
			result = append(result, dp)
		}
	}
	return writeJSON(w, http.StatusOK, result)
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/deployments/{dp_id:.*}":
func (s *Server) getDeployment(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("getDeploymentList with(%v)", vars)
	if vars == nil {
		err := fmt.Errorf("parameter of siteid/projid/deployment id are required")
		return handleError(w, "getDeployment of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "getDeployment of site agent", err)
	}

	projId := vars[sconst.ProjId]
	dpId := vars[sconst.DeploymentId]

	deployments, err := s.backend.MarathonClient.Deployments()
	if err != nil {
		return handleError(w, "getDeployment of site agent: failed to call remote Deployments()", err)
	}
	projIdPrefix := "/" + projId + "/"
	for _, deployment := range deployments {
		if deployment.ID == dpId {
			for _, appId := range deployment.AffectedApps {
				if strings.HasPrefix(appId, projIdPrefix) {
					glog.Infof("s.backend.MarathonClient.Deployments(%s) return deployment=%v)", projId, *deployment)
					return writeJSON(w, http.StatusOK, *deployment)
				}
			}
		}
	}

	// no qualified deployment found
	glog.Infof("getDeployment of site agent: not found")
	w.WriteHeader(http.StatusNotFound)
	return nil
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/jobs/":
func (s *Server) getProjectJobList(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	// TODO: add it when we introduce chronous
	err := fmt.Errorf("NOT implemented")
	return handleError(w, "getProjectJobList of site agent", err)
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/jobs/{job_id:.*}":
func (s *Server) getJob(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	// TODO: add it when we introduce chronous
	err := fmt.Errorf("NOT implemented")
	return handleError(w, "getJob of site agent", err)
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/quota":
func (s *Server) putProjectQuota(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	// TODO: add it when we introduce quota&project mechanism into marathon
	err := fmt.Errorf("NOT implemented")
	return handleError(w, "putProjectQuota of site agent", err)
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/applications/{app_id:.*}":
func (s *Server) putApplication(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("putApplication with(%v)", vars)
	if vars == nil {
		err := fmt.Errorf("parameter of siteid/projid/applicationid/replica are required")
		return handleError(w, "putApplication of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "putApplication of site agent", err)
	}

	projId := vars[sconst.ProjId]
	appId := vars[sconst.AppId]
	fullAppName := "/" + projId + "/" + appId

	force := false
	paraForce := r.URL.Query().Get(sconst.ParameterForce)
	if paraForce == "true" {
		force = true
	}

	glog.Infof("putApplication with(site=%s, proj=%s, app=%s)", siteId, projId, appId)

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		err := fmt.Errorf("failed to get the request http body of application, error: %s\n", err)
		return handleError(w, "putApplication of site agent", err)
	}

	app := new(marathon.Application)
	if err := json.Unmarshal(reqBody, app); err != nil {
		err := fmt.Errorf("failed to unmarshall the request http body of application, error: %s\n", err)
		return handleError(w, "putApplication of site agent", err)
	}

	if !strings.EqualFold(app.ID, fullAppName) {
		err := fmt.Errorf("failed to update the application, error: app(%s) does NOT match the specified full name (%s)\n", app.ID, fullAppName)
		return handleError(w, "putApplication of site agent", err)
	}

	glog.Infof("createing application with(%v)", *app)

	deploymentID, err := s.backend.MarathonClient.UpdateApplication(app, force)
	if err == nil {
		glog.Infof("s.backend.MarathonClient.UpdateApplication(%s) succeeds with deploymentid=%v)", fullAppName, *deploymentID)
		return writeJSON(w, http.StatusOK, *deploymentID)
	} else {
		return handleError(w, "putApplication of site agent: failed to call remote UpdateApplication()", err)
	}
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/applications/{app_id:.*}/restart":
func (s *Server) putApplicationRestart(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("putApplicationRestart with(%v)", vars)
	if vars == nil {
		err := fmt.Errorf("parameter of siteid/projid/applicationid are required")
		return handleError(w, "putApplicationRestart of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "putApplicationRestart of site agent", err)
	}

	projId := vars[sconst.ProjId]
	appId := vars[sconst.AppId]
	fullAppName := "/" + projId + "/" + appId

	force := false
	paraForce := r.URL.Query().Get(sconst.ParameterForce)
	if paraForce == "true" {
		force = true
	}

	glog.Infof("putApplicationRestart with(site=%s, proj=%s, app=%s)", siteId, projId, appId)

	deploymentID, err := s.backend.MarathonClient.RestartApplication(fullAppName, force)
	if err == nil {
		glog.Infof("s.backend.MarathonClient.RestartApplication(%s) succeeds with deploymentid=%v)", fullAppName, *deploymentID)
		return writeJSON(w, http.StatusOK, *deploymentID)
	} else {
		return handleError(w, "putApplicationRestart of site agent: failed to call remote RestartApplication()", err)
	}
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/applications/{app_id:.*}/replicas/{replicas:.*}":
func (s *Server) putApplicationScale(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("putApplicationScale with(%v)", vars)
	if vars == nil {
		err := fmt.Errorf("parameter of siteid/projid/applicationid/replica are required")
		return handleError(w, "putApplicationScale of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "putApplicationScale of site agent", err)
	}

	projId := vars[sconst.ProjId]
	appId := vars[sconst.AppId]
	replicas := vars[sconst.AttrReplicas]
	fullAppName := "/" + projId + "/" + appId

	replicasNum, _ := strconv.Atoi(replicas)

	force := false
	paraForce := r.URL.Query().Get(sconst.ParameterForce)
	if paraForce == "true" {
		force = true
	}

	glog.Infof("putApplicationScale with(site=%s, proj=%s, app=%s, replicas=%d)", siteId, projId, appId, replicasNum)

	deploymentID, err := s.backend.MarathonClient.ScaleApplicationInstances(fullAppName, replicasNum, force)
	if err == nil {
		glog.Infof("s.backend.MarathonClient.ScaleApplicationInstances(%s) succeeds with deploymentid=%v)", fullAppName, *deploymentID)
		return writeJSON(w, http.StatusOK, *deploymentID)
	} else {
		glog.Errorf("Could not scale application %s, error: %s", fullAppName, err)
		return handleError(w, "putApplicationScale of site agent: failed to call remote ScaleApplicationInstances()", err)
	}
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/applications":
func (s *Server) postApplication(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("postApplication with(%v)", vars)
	if vars == nil {
		err := fmt.Errorf("parameter of siteid/projid/appid are required")
		return handleError(w, "postApplication of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("parameter of siteid/projid/application id are required")
		return handleError(w, "postApplication of site agent", err)
	}

	projId := vars[sconst.ProjId]
	fullProjName := "/" + projId + "/"

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		err = fmt.Errorf("failed to get the request http body, error: %s\n", err)
		return handleError(w, "postApplication of site agent", err)
	}

	app := new(marathon.Application)
	if err = json.Unmarshal(reqBody, app); err != nil {
		err = fmt.Errorf("failed to unmarshall the request http body, error: %s\n", err)
		return handleError(w, "postApplication of site agent", err)
	}

	if !strings.HasPrefix(app.ID, fullProjName) {
		err = fmt.Errorf("app(%s) does NOT belong to project(/%s)\n", app.ID, projId)
		return handleError(w, "postApplication of site agent", err)
	}

	glog.Infof("creating application with(%v)", *app)
	appCreated, err := s.backend.MarathonClient.CreateApplication(app)
	if err == nil {
		glog.Infof("s.backend.MarathonClient.createApplication() return the app being created=%v)", *appCreated)
		return writeJSON(w, http.StatusOK, *appCreated)
	} else {
		return handleError(w, "postApplication of site agent: failed to call remote CreateApplication()", err)
	}
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/jobs/{job_id:.*}":
func (s *Server) postJob(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	// TODO: add it when we introduce chronous
	err := fmt.Errorf("NOT implemented")
	return handleError(w, "postJob of site agent", err)
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/applications/{app_id:.*}":
func (s *Server) deleteApplication(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("deleteApplication with(%v)", vars)
	if vars == nil {
		err := fmt.Errorf("parameter of siteid/projid/application id are required")
		return handleError(w, "deleteApplication of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "deleteApplication of site agent", err)
	}

	projId := vars[sconst.ProjId]
	appId := vars[sconst.AppId]

	//TODO: we need to revisit enhancement of marathon client for supporting force parameter of DeleteApplication
	fullAppName := "/" + projId + "/" + appId
	deploymentId, err := s.backend.MarathonClient.DeleteApplication(fullAppName)
	if err == nil {
		glog.Infof("s.backend.MarathonClient.DeleteApplication(%s) return deployment_id=%v)", fullAppName, *deploymentId)
		return writeJSON(w, http.StatusOK, *deploymentId)
	} else {
		return handleError(w, "deleteApplication of site agent: failed to call remote DeleteApplication()", err)
	}
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}":
func (s *Server) deleteProject(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("deleteProject with(%v)", vars)
	if vars == nil {
		err := fmt.Errorf("parameter of siteid/projid/ are required")
		return handleError(w, "deleteProject of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "deleteProject of site agent", err)
	}

	projId := vars[sconst.ProjId]

	//TODO: we need to revisit enhancement of marathon client for supporting force parameter of DeleteGroup
	fullProjName := "/" + projId
	deploymentId, err := s.backend.MarathonClient.DeleteGroup(fullProjName)
	if err == nil {
		glog.Infof("s.backend.MarathonClient.DeleteGroup(%s) return deployment_id=%v)", projId, *deploymentId)
		return writeJSON(w, http.StatusOK, *deploymentId)
	} else {
		return handleError(w, "deleteProject of site agent: failed to call remote DeleteGroup()", err)
	}
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/jobs/{job_id:.*}":
func (s *Server) deleteJob(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	// TODO: add it when we introduce chronous
	err := fmt.Errorf("NOT implemented")
	return handleError(w, "deleteJob of site agent", err)
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/deployments/{dp_id:.*}":
func (s *Server) deleteDeployment(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("deleteDeployment with(%v)", vars)
	if vars == nil {
		err := fmt.Errorf("parameter of siteid/projid/deployment id are required")
		return handleError(w, "deleteDeployment of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "deleteDeployment of site agent", err)
	}

	projId := vars[sconst.ProjId]
	dpId := vars[sconst.DeploymentId]

	projIdPrefix := "/" + projId + "/"
	deployments, err := s.backend.MarathonClient.Deployments()
	if err != nil {
		return handleError(w, "deleteDeployment of site agent: failed to call remote Deployments()", err)
	}
	for _, deployment := range deployments {
		if deployment.ID == dpId {
			for _, affectedAppId := range deployment.AffectedApps {
				if strings.HasPrefix(affectedAppId, projIdPrefix) {
					// TODO:
					//   i) we need to change marathon client to support force parameter of DeleteDeployment
					//   ii) we need to revisit enhancing marather server api for namespacing of deployment
					//       so as to save round trip for efficience
					// we delete the deploymnet with the same id as the specified project
					deletedDpId, err := s.backend.MarathonClient.DeleteDeployment(dpId, true)
					if err == nil {
						glog.Infof("s.backend.MarathonClient.DeleteDeployments(%s) succeeded and return with deploymentId=%v)", dpId, *deletedDpId)
						return writeJSON(w, http.StatusOK, *deletedDpId)
					} else {
						return handleError(w, "deleteDeployment of site agent: failed to call remote DeleteDeployment()", err)
					}
				}
			}
		}
	}

	// no matched deployment, do nothing
	w.WriteHeader(http.StatusOK)
	return nil
}

// "/sites/{site_id:.*}/projects/{proj_id:.*}/applications/{app_id:.*}/tasks":
// request body like below
// {
//  "ids": [
//          "task.25ab260e-b5ec-11e4-a4f4-685b35c8a22e",
//          "task.5e7b39d4-b5f0-11e4-8021-685b35c8a22e",
//          "task.a21cb64a-b5eb-11e4-a4f4-685b35c8a22e"
//         ]
// }
func (s *Server) deleteTasks(w http.ResponseWriter, r *http.Request, vars map[string]string) error {
	glog.Infof("deleteTasks with(%v)", vars)
	if vars == nil {
		err := fmt.Errorf("parameter of siteid/projid/appid/taskid are required")
		return handleError(w, "deleteTasks of site agent", err)
	}

	siteId := vars[sconst.SiteId]

	if siteId != s.cfg.SiteName {
		err := fmt.Errorf("the specified site(%s) does NOT match the configured site(%s)", siteId, s.cfg.SiteName)
		return handleError(w, "deleteTasks of site agent", err)
	}

	projId := vars[sconst.ProjId]
	appId := vars[sconst.AppId]

	reqBody, err := ioutil.ReadAll(r.Body)
	if err != nil {
		err = fmt.Errorf("failed to read request http body, error: %s", err)
		return handleError(w, "deleteTasks of site agent", err)
	}

	var post struct {
		IDs []string `json:"ids"`
	}
	if err := json.Unmarshal(reqBody, &post); err != nil {
		err = fmt.Errorf("failed to unmarshall the request http body, error: %s", err)
		return handleError(w, "deleteTasks of site agent", err)
	}
	taskIds := post.IDs

	killOpts := &marathon.KillTaskOpts{}
	paraForce := r.URL.Query().Get(sconst.ParameterForce)
	if paraForce == "true" {
		killOpts.Scale = true
	}

	glog.Infof("deleteTasks with(taskIds=%v , opts=%v)", taskIds, killOpts)

	// check if those tasks belongs to the specified app
	// if not, reject this request
	// for the purpose of efficience, we check the task id format according to the Marathon internals
	//   class mesosphere.marathon.tasks.TaskIdUtil  & mesosphere.marathon.state.PathId
	//
	//   taskid:           appId.safePath + appDelimiter + uuid
	//   appDelimiter:     "_"
	//   appId.safePath:   if appId is "/p1/p2/p3" and appDelimiter is "_", then appId.safePath P1_p2_p3
	//
	safeAppId := projId + "_" + appId + "."
	for _, taskId := range taskIds {
		if !strings.HasPrefix(taskId, safeAppId) {
			// found a task does NOT belong to the app, we reject the request
			err = fmt.Errorf("rejected to kill taskid (%s) because it does NOT belong to application (/%s/%s)", taskId, projId, appId)
			return handleError(w, "deleteTasks of site agent", err)
		}
	}

	// TODO: enhance marathon client KillTasks() with scale = true and return deploymentid
	err = s.backend.MarathonClient.KillTasks(taskIds, killOpts)
	if err == nil {
		glog.Infof("s.backend.MarathonClient.KillTask(%s)", taskIds)
		w.WriteHeader(http.StatusOK)
		return nil
	} else {
		return handleError(w, "deleteTasks of site agent: failed to call remote KillTasks()", err)
	}
}

func handleError(w http.ResponseWriter, errContext string, err error) error {
	errRes := fmt.Errorf("%s: %v", errContext, err)
	glog.Errorf("%v", errRes)
	writeText(w, http.StatusNotAcceptable, errRes.Error())
	return nil
}

func makeHttpHandler(logging bool, localMethod string, localRoute string, handlerFunc HttpApiFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if logging {
			glog.Infof("%s %s", r.Method, r.RequestURI)
		}

		w.Header().Set("Server", "siteagent"+" ("+runtime.GOOS+")")

		vars := make(map[string]string, len(mux.Vars(r)))
		for key, value := range mux.Vars(r) {
			orgVal, err := url.QueryUnescape(value)
			if err != nil {
				err = fmt.Errorf("failed to url decode query parameter(%s) in request(%s): %v", value, r.RequestURI, err)
				glog.Errorf("%s", err)
				httpError(w, err)
			} else {
				vars[key] = orgVal
			}
		}
		if err := handlerFunc(w, r, vars); err != nil {
			err = fmt.Errorf("Site agent failed to handle request(%s): %v", r.RequestURI, err)
			glog.Errorf("%s", err)
			httpError(w, err)
		}
	}
}

func profilerSetup(mainRouter *mux.Router, path string) {
	var r = mainRouter.PathPrefix(path).Subrouter()
	r.HandleFunc("/vars", expVars)
	r.HandleFunc("/pprof/", pprof.Index)
	r.HandleFunc("/pprof/cmdline", pprof.Cmdline)
	r.HandleFunc("/pprof/profile", pprof.Profile)
	r.HandleFunc("/pprof/symbol", pprof.Symbol)
	r.HandleFunc("/pprof/block", pprof.Handler("block").ServeHTTP)
	r.HandleFunc("/pprof/heap", pprof.Handler("heap").ServeHTTP)
	r.HandleFunc("/pprof/goroutine", pprof.Handler("goroutine").ServeHTTP)
	r.HandleFunc("/pprof/threadcreate", pprof.Handler("threadcreate").ServeHTTP)
}

// Replicated from expvar.go as not public.
func expVars(w http.ResponseWriter, r *http.Request) {
	first := true
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	fmt.Fprintf(w, "{\n")
	expvar.Do(func(kv expvar.KeyValue) {
		if !first {
			fmt.Fprintf(w, ",\n")
		}
		first = false
		fmt.Fprintf(w, "%q: %s", kv.Key, kv.Value)
	})
	fmt.Fprintf(w, "\n}\n")
}

type QueriesHttpApiFunc struct {
	queries []string
	api     HttpApiFunc
}

type QueriesHttpApiFuncList []QueriesHttpApiFunc

func createRouter(s *Server) *mux.Router {
	r := mux.NewRouter()
	if os.Getenv("DEBUG") != "" {
		profilerSetup(r, "/debug/")
	}

	// !!! Attention: the queries definition order matters: first match, first serve
	m := map[string]map[string]QueriesHttpApiFuncList{
		"GET": {
			"/" + sconst.Ping: []QueriesHttpApiFunc{
				{queries: []string{}, api: s.ping},
			},
			"/proxy": []QueriesHttpApiFunc{
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Applications, sconst.QueryAppId, sconst.Attributes, sconst.AttrVersions},
					api:     s.getVersionList,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Applications, sconst.QueryAppId, sconst.Versions, sconst.QueryVerId},
					api:     s.getVersion,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Applications, sconst.QueryAppId, sconst.Attributes, sconst.AttrTasks},
					api:     s.getTaskList,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Applications, sconst.QueryAppId},
					api:     s.getApplication,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Attributes, sconst.Applications},
					api:     s.getProjectApplicationList,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Attributes, sconst.AttrApplications},
					api:     s.getApplicationList,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Jobs, sconst.QueryJobId},
					api:     s.getJob,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Attributes, sconst.Jobs},
					api:     s.getProjectJobList,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Deployments, sconst.QueryDeploymentId},
					api:     s.getDeployment,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Attributes, sconst.Deployments},
					api:     s.getProjectDeploymentList,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Attributes, sconst.AttrProjects},
					api:     s.getProjectList,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId},
					api:     s.getProject,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Attributes, sconst.AttrQuota},
					api:     s.getProjectQuota,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Attributes, sconst.AttrInfo},
					api:     s.getSiteInfo,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Attributes, sconst.AttrLeader},
					api:     s.getSiteLeader,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Attributes, sconst.AttrList},
					api:     s.getSiteList,
				},
			},
		},
		"PUT": {
			"/proxy": []QueriesHttpApiFunc{
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Attributes, sconst.AttrQuota},
					api:     s.putProjectQuota,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Applications, sconst.QueryAppId, sconst.Attributes, sconst.AttrRestart},
					api:     s.putApplicationRestart,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Applications, sconst.QueryAppId, sconst.AttrReplicas, sconst.QueryReplicas},
					api:     s.putApplicationScale,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Applications, sconst.QueryAppId},
					api:     s.putApplication,
				},
			},
		},
		"POST": {
			"/proxy": []QueriesHttpApiFunc{
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Attributes, sconst.Applications},
					api:     s.postApplication,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Attributes, sconst.Jobs},
					api:     s.postJob,
				},
			},
		},
		"DELETE": {
			"/proxy": []QueriesHttpApiFunc{
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Applications, sconst.QueryAppId, sconst.Attributes, sconst.Tasks},
					api:     s.deleteTasks,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Applications, sconst.QueryAppId},
					api:     s.deleteApplication,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Jobs, sconst.QueryJobId},
					api:     s.deleteJob,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId, sconst.Deployments, sconst.QueryDeploymentId},
					api:     s.deleteDeployment,
				},
				{
					queries: []string{sconst.Sites, sconst.QuerySiteId, sconst.Projects, sconst.QueryProjId},
					api:     s.deleteProject,
				},
			},
		},
	}

	for method, routes := range m {
		for route, queries := range routes {
			for _, hdlr := range queries {
				queryString := strings.Join(hdlr.queries, ",")
				glog.Infof("Registering %s, %s ? %s", method, route, queryString)
				localMethod := method
				localRoute := route
				localQueries := hdlr.queries
				localFct := hdlr.api

				// build the handler function
				f := makeHttpHandler(s.cfg.Logging, localMethod, localRoute, localFct)

				if localRoute == "" {
					if len(localQueries) > 0 {
						r.Methods(localMethod).Queries(localQueries...).HandlerFunc(f)
					} else {
						r.Methods(localMethod).HandlerFunc(f)
					}
				} else {
					if len(localQueries) > 0 {
						r.Path(localRoute).Methods(localMethod).Queries(localQueries...).HandlerFunc(f)
					} else {
						r.Path(localRoute).Methods(localMethod).HandlerFunc(f)
					}
				}
			}
		}
	}

	return r
}
