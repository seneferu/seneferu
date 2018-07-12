package main

import (
	"context"
	"log"

	"github.com/heetch/confita"
	"github.com/heetch/confita/backend/env"
	"github.com/heetch/confita/backend/flags"
	"github.com/pkg/errors"
	"gitlab.com/sorenmat/seneferu/storage/sql"
	"gitlab.com/sorenmat/seneferu/web"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Config struct {
	KubeCfgFile   string `config:"kubecfgFile"`
	GithubSecret  string `config:"githubSecret,required"`
	GithubToken   string `config:"githubToken,required"`
	Sshkey        string `config:"sshkey,required"`
	TargetURL     string `config:"targetURL,required"`
	DockerRegHost string `config:"dockerRegHost"`
}

/*
var (
	kubeCfgFile   = kingpin.Flag("kubeconfig", "Kubernetes Config File").Envar("KUBE_CONFIG").String()
	githubSecret  = kingpin.Flag("githubsecret", "Github secret token, needs to match the one on Github ").Envar("GITHUB_SECRET").Required().String()
	githubToken   = kingpin.Flag("githubToken", "Github access token, to access the API").Envar("GITHUB_TOKEN").Required().String()
	sshkey        = kingpin.Flag("sshkey", "Github ssh key, used for cloning the repositories").Envar("SSH_KEY").Required().String()
	targetURL     = kingpin.Flag("targetURL", "Base URL to use for reporting status to Github").Envar("TARGET_URL").Required().String()
	dockerRegHost = kingpin.Flag("dockerhost", "Host name of a private docker registry").Envar("DOCKER_REGISTRY_HOST").String()
)
*/
func main() {
	//kingpin.Parse()

	loader := confita.NewLoader(
		env.NewBackend(),
		//file.NewBackend("config.json"),
		//file.NewBackend("config.yaml"),
		flags.NewBackend())
	cfg := Config{
		KubeCfgFile: "~/.kube/config"}
	err := loader.Load(context.Background(), &cfg)
	if err != nil {
		log.Fatal(err)
	}
	config, err := rest.InClusterConfig()
	if err != nil {
		log.Println("Appears we are not running in a cluster")
		config, err = clientcmd.BuildConfigFromFlags("", cfg.KubeCfgFile)
		if err != nil {
			log.Fatal(err)
		}
	} else {
		log.Println("Seems like we are running in a Kubernetes cluster!!")
	}

	log.Println("Trying to connect to database")
	service, err := sql.New()
	if err != nil {
		log.Fatal(errors.Wrap(err, "unable to create database connection"))
	}
	log.Println("... connected")

	log.Println("Setting up Kubernets access")
	kubectl, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(errors.Wrap(err, "unable create kubectl"))
	}

	log.Println("Starting web server...")
	web.StartWebServer(service, kubectl, cfg.GithubSecret, cfg.TargetURL, cfg.GithubToken, cfg.DockerRegHost, cfg.Sshkey)
}
