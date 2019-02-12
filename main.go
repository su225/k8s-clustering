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
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// MemberPod represents the pod which is the member
// of the cluster to which this node belongs
type MemberPod struct {
	PodName      string            `json:"pod_name"`
	PodNamespace string            `json:"pod_ns"`
	Labels       map[string]string `json:"labels"`
}

func main() {
	go setupKubernetesListener()
	launchHTTPServer()
}

func launchHTTPServer() {
	r := mux.NewRouter()
	r.HandleFunc("/peers", getPeers).Methods("GET")
	r.HandleFunc("/peers/reachable", getReachableNodes).Methods("GET")
	if err := http.ListenAndServe(":8080", r); err != nil {
		logrus.Errorf("error while launching HTTP server (reason=%s)", err.Error())
		return
	}
}

func setupKubernetesListener() {
	k8sconfig, err := rest.InClusterConfig()
	if err != nil {
		logrus.Errorf("error while obtaining kubernetes-configuration (reason=%s)", err)
		return
	}
	clientset, err := kubernetes.NewForConfig(k8sconfig)
	if err != nil {
		logrus.Errorf("error while creating clientset (reason=%s)", clientset)
	}
	for {
		pods, err := clientset.CoreV1().Pods("").List(metav1.ListOptions{})
		if err != nil {
			logrus.Errorf("error while querying pods (reason=%s)", err.Error())
			continue
		}
		logrus.Infof("there are %d pods in the cluster", len(pods.Items))
		for _, pod := range pods.Items {
			logrus.Infof("pod %s/%s has labels %v",
				pod.GetNamespace(),
				pod.GetName(),
				pod.GetLabels())
		}
		<-time.After(10 * time.Second)
	}
}

// getPeers returns all the peers known to this node in the cluster
func getPeers(w http.ResponseWriter, r *http.Request) {
	logrus.Infof("received GET /peers request from %s", r.RemoteAddr)
}

func getReachableNodes(w http.ResponseWriter, r *http.Request) {
	logrus.Infof("received GET /peers/reachable request from %s", r.RemoteAddr)
}
