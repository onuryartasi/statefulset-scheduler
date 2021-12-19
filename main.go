package main

import (
	"context"
	"flag"
	"fmt"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"path/filepath"
	"time"
)

func main(){
	const debug=false
	const schedulerName = "sfs-scheduler"
	var kubeconfig *string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()

	var config *rest.Config
	var err error
	if debug {
		config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
		if err != nil {
			panic(err.Error())
		}
	}else{
		config, err = rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
	}


	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}

	watch,err := clientset.CoreV1().Pods("").Watch(context.Background(),metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.schedulerName=%s,spec.nodeName=",schedulerName),

	})

	if err != nil {
		fmt.Errorf("can't watch pods , %s",err)
	}
	fmt.Println("Scheduler Started")
	for event := range watch.ResultChan() {
		if event.Type != "ADDED" {
			continue
		}
		p := event.Object.(*v1.Pod)

		nodes, err := clientset.CoreV1().Nodes().List(context.Background(),metav1.ListOptions{LabelSelector: fmt.Sprintf("%s=true",p.Name),

		})
		if err != nil {
			fmt.Printf("Cannot list nodes: %s",err)
		}
		var nodeName = ""
		for _,node:= range nodes.Items {
			nodeName = node.Name
		}

		err = clientset.CoreV1().Pods(p.Namespace).Bind(context.Background(),&v1.Binding{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.Name,
				Namespace: p.Namespace,
			},
			Target: v1.ObjectReference{
				APIVersion: "v1",
				Kind:       "Node",
				Name:       nodeName,
			},
		},metav1.CreateOptions{})

		if err != nil {
			fmt.Printf("Error with bind %s\n",err)
		}
		fmt.Println("Pod scheduled", p.Namespace, "/", p.Name,"to",nodeName)
		timestamp := time.Now().UTC()
		clientset.CoreV1().Events(p.Namespace).Create(context.Background(),&v1.Event{
			Count:          1,
			Message:        fmt.Sprintf("Successfully scheduled to %s",nodeName),
			Reason:         "Scheduled",
			LastTimestamp:  metav1.NewTime(timestamp),
			FirstTimestamp: metav1.NewTime(timestamp),
			Type:           "Normal",
			Source: v1.EventSource{
				Component: schedulerName,
			},
			InvolvedObject: v1.ObjectReference{
				Kind:      "Pod",
				Name:      p.Name,
				Namespace: p.Namespace,
				UID:       p.UID,
			},
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: p.Name + "-",
			},
		},metav1.CreateOptions{})
	}


}


