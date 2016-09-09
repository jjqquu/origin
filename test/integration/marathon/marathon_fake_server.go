package main

import (
	"flag"
	"fmt"
	"github.com/donovanhide/eventsource"
	"github.com/golang/glog"
	yaml "gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"runtime"
)

const (
	fakeAPIFilename = "./methods.yml"
	fakeAPIPort     = 3000
)

type restMethod struct {
	// the uri of the method
	URI string `yaml:"uri,omitempty"`
	// the http method type (GET|PUT etc)
	Method string `yaml:"method,omitempty"`
	// the content i.e. response
	Content string `yaml:"content,omitempty"`
}

type fakeServer struct {
	io.Closer

	eventSrv *eventsource.Server
	httpSrv  *http.Server
	listener net.Listener
}

type fakeEvent struct {
	data string
}

func (t fakeEvent) Id() string {
	return "0"
}

func (t fakeEvent) Event() string {
	return "MarathonEvent"
}

func (t fakeEvent) Data() string {
	return t.data
}

var uris map[string]*string
var apiPort = flag.Uint("port", fakeAPIPort, "tcp port to listen to")
var apiFilename = flag.String("file", fakeAPIFilename, "yml file to define api req/resp")

func newFakeServer(port uint, reqresp string) (*fakeServer, error) {
	// step: open and read in the methods yaml
	contents, err := ioutil.ReadFile(reqresp)
	if err != nil {
		glog.Fatalf("unable to read in the methods yaml file: %s", reqresp)
	}
	// step: unmarshal the yaml
	var methods []*restMethod
	err = yaml.Unmarshal([]byte(contents), &methods)
	if err != nil {
		glog.Fatalf("Unable to unmarshal the methods yaml, error: %v", err)
		return nil, err
	}

	// step: construct a hash from the methods
	uris = make(map[string]*string, 0)
	for _, method := range methods {
		uris[fmt.Sprintf("%s:%s", method.Method, method.URI)] = &method.Content
	}

	eventSrv := eventsource.NewServer()

	mux := http.NewServeMux()
	mux.HandleFunc("/v2/events", eventSrv.Handler("event"))
	mux.HandleFunc("/publish", func(writer http.ResponseWriter, reader *http.Request) {
		event := "fakeevent"
		eventSrv.Publish([]string{"event"}, fakeEvent{event})
	})
	mux.HandleFunc("/", func(writer http.ResponseWriter, reader *http.Request) {
		key := fmt.Sprintf("%s:%s", reader.Method, reader.RequestURI)
		content, found := uris[key]

		glog.Infof("get request = %s", key)

		if found {
			glog.Infof("send response = \n%s", *content)

			writer.Header().Add("Content-Type", "application/json")
			writer.Write([]byte(*content))
			return
		}

		glog.Infof("send response = message not found")

		http.Error(writer, `{"message": "not found"}`, 404)
	})

	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)

	if err != nil {
		return nil, err
	}

	res := &fakeServer{
		eventSrv: eventSrv,
		httpSrv: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
		listener: listener,
	}
	return res, nil
}

func (s *fakeServer) startServing() error {
	var chError = make(chan error, 1)

	glog.Infof("Started, serving at %v\n", s.httpSrv.Addr)

	go func() {
		err := http.Serve(s.listener, s.httpSrv.Handler)

		if err != nil {
			glog.Info("Server message: %v", err)
		}
		chError <- err
	}()

	glog.Info("channel begins to work")
	err := <-chError
	if err != nil {
		glog.Infof("channel failed to work because of %v", err)
		return err
	}
	glog.Info("channel finishes working")

	return nil
}

func (s *fakeServer) Close() error {
	s.eventSrv.Close()
	s.listener.Close()
	return nil
}

func main() {
	if len(os.Getenv("GOMAXPROCS")) == 0 {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	flag.Parse()

	svr, err := newFakeServer(*apiPort, *apiFilename)
	if err != nil {
		glog.Fatalf("failed to new fake marathon server with port as %d", *apiPort)
	} else {
		svr.startServing()
	}
}
