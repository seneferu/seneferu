package main

import (
	"flag"
	"fmt"
	"github.com/pkg/errors"
	"gitlab.com/sorenmat/seneferu/storage/sql"
	"gitlab.com/sorenmat/seneferu/web"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"log"
	"os"
)

var (
	kubeCfgFile  = flag.String("kubeconfig", "", "Kubernetes Config File")
	githubSecret = flag.String("githubsecret", "", "Github secret token, needs to match the one on Github ")
	helmHost     = flag.String("helmhost", "", "Hostname and port of the Helm host / tiller")
	githubToken  = flag.String("githubToken", "", "Github access token, to access the API")
)

func main() {
	flag.Parse()
	if *githubSecret == "" {
		*githubSecret = os.Getenv("githubsecret")
	}
	if *githubSecret == "" {
		log.Fatal("githubsecret can't be empty")
	}
	if *helmHost == "" {
		*helmHost = os.Getenv("helmhost")
	}
	if *helmHost == "" {
		log.Fatal("helmHost can't be empty")
	}
	if *githubToken == "" {
		*githubToken = os.Getenv("githubtoken")
	}
	if *githubToken == "" {
		log.Fatal("githubtoken can't be empty")
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		log.Println("Appears we are not running in a cluster")
		config, err = clientcmd.BuildConfigFromFlags("", *kubeCfgFile)
		if err != nil {
			panic(err.Error())
		}
	} else {
		log.Println("Seems like we are running in a Kubernetes cluster!!")
	}

	service, err := sql.New()

	fmt.Println("Setting up Kubernets access")
	kubectl, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(errors.Wrap(err, "unable create kubectl"))
	}
	/*
		err = builder.CreateConfigmap(kubectl, sshkey)
		if err != nil {
			log.Fatal("Unable to create secret for ssh key: ",err)
		}*/
	fmt.Println("Starting web server...")
	web.StartWebServer(service, kubectl, *githubSecret, *helmHost, *githubToken)
}
