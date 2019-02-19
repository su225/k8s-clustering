// MIT License

// Copyright (c) 2019 Suchith J N

// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// ServiceContext holds all the objects required for this
// service to work properly.
type ServiceContext struct {
	Port uint64
	*http.Server
	*kubernetes.Clientset
}

// Start starts the components of the ServiceContext and if
// there is any error then the node shuts down gracefully
func (ctx *ServiceContext) Start(stopSignal chan<- os.Signal) {
	logrus.Infof("starting kubernetes clustering provider for raft")
	if err := ctx.setupKubernetesClientset(); err != nil {
		stopSignal <- syscall.SIGABRT
		return
	}
	ctx.startHTTPServer(stopSignal)
}

// setupKubernetesClientset sets up kubernetes client set and
// returns error if there is any
func (ctx *ServiceContext) setupKubernetesClientset() error {
	config, err := rest.InClusterConfig()
	if err != nil {
		logrus.Errorf("error while setting up K8S in-cluster client config. Reason=%s", err.Error())
		return err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		logrus.Errorf("error while getting k8s clientset. Reason=%s", err.Error())
		return err
	}
	ctx.Clientset = clientset
	return nil
}

// startHTTPServer starts the HTTP server at the specified port
func (ctx *ServiceContext) startHTTPServer(stopSignal chan<- os.Signal) {
	ctx.Server = &http.Server{
		Addr:    fmt.Sprintf(":%d", ctx.Port),
		Handler: ctx.setupHTTPServerRoutes(),
	}
	go func() {
		logrus.Infof("starting discovery server at port %d", ctx.Port)
		if err := ctx.Server.ListenAndServe(); err != http.ErrServerClosed {
			logrus.Errorf("error while serving. Reason=%s", err.Error())
			stopSignal <- syscall.SIGABRT
		}
	}()
}

// setupHTTPServerRoutes sets up the HTTP server routes
func (ctx *ServiceContext) setupHTTPServerRoutes() *mux.Router {
	r := mux.NewRouter()
	r.HandleFunc("/v1/nodes/{ns}/{label}", ctx.getNodesWithLabel)
	return r
}

// getNodesWithLabel returns all nodes with the given label in the given namespace.
// If successful then the list containing node names is returned with status OK. If
// one of namespace or label is missing then status Bad Request is returned. If there
// is any error while retrieving or marshaling the response then status Internal
// Server Error is returned.
func (ctx *ServiceContext) getNodesWithLabel(w http.ResponseWriter, r *http.Request) {
	ns, nsPresent := mux.Vars(r)["ns"]
	label, labelPresent := mux.Vars(r)["label"]

	if !nsPresent || !labelPresent {
		errMsg := "label and namespace must be present"
		logrus.Errorf("[%s] %s", r.RemoteAddr, errMsg)
		w.Write([]byte(errMsg))
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	logrus.Infof("[%s] received getNodes request. label=%s, ns=%s", r.RemoteAddr, ns, label)
	podlist, err := ctx.Clientset.CoreV1().Pods(ns).List(metav1.ListOptions{LabelSelector: label})
	if err != nil {
		logrus.Errorf("[%s] error while retrieving pods (ns=%s,label=%s). Reason=%s",
			r.RemoteAddr, ns, label, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}

	// Get the podnames and return the same
	// or return marshaling error if it occurs
	podnames := make([]string, 0)
	for _, pod := range podlist.Items {
		podnames = append(podnames, pod.ObjectMeta.GetName())
	}
	marshaled, err := json.Marshal(podnames)
	if err != nil {
		logrus.Errorf("[%s] error while marshaling podnames. Reason=%s", r.RemoteAddr, err.Error())
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(err.Error()))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(marshaled)
}

// Destroy shuts down the components in ServiceContext gracefully
func (ctx *ServiceContext) Destroy() {
	logrus.Infof("destroying the kubernetes clustering provider")
	if ctx.Server != nil {
		shutdownCtx, cancelFunc := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancelFunc()
		if err := ctx.Server.Shutdown(shutdownCtx); err != nil {
			logrus.Errorf("error while shutting down discovery server. Reason=%s", err.Error())
			return
		}
	}
}

func main() {
	port := flag.Uint64("port", 8888, "port in which the service listens")
	flag.Parse()

	svcContext := &ServiceContext{Port: *port}

	gracefulStop := make(chan os.Signal)
	signal.Notify(gracefulStop, syscall.SIGABRT)
	signal.Notify(gracefulStop, syscall.SIGTERM)
	signal.Notify(gracefulStop, syscall.SIGHUP)

	go svcContext.Start(gracefulStop)

	<-gracefulStop
	logrus.Infof("received signal %d", gracefulStop)
	logrus.Infof("initiate graceful shutdown")
	svcContext.Destroy()
}
