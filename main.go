package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"time"

	"github.com/asdine/storm"
	"github.com/boltdb/bolt"
	"github.com/cncd/pipeline/pipeline/frontend/yaml"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"gitlab.com/sorenmat/ci-server/github"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

const shareddir = "/share"

var (
	kubeCfgFile  = flag.String("kubeconfig", "", "Kubernetes Config File")
	githubSecret = flag.String("githubsecret", "", "Github secret token, needs to match the one on Github ")
)

func main() {
	flag.Parse()
	if *githubSecret == "" {
		*githubSecret = os.Getenv("githubsecret")
	}
	if *githubSecret == "" {
		log.Fatal("githubsecret can't be empty")
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
	fmt.Println("Starting web server...")
	startWeb(config)
}

func startWeb(config *rest.Config) {
	db, err := storm.Open("my.db")
	if err != nil {
		log.Fatal("unable to load db")
	}
	defer db.Close()

	kubectl, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	startUI(db, kubectl, *githubSecret)
}

// generateScript is a helper function that generates a build script and base64 encode it.
func generateScript(commands []string) string {
	var buf bytes.Buffer
	for _, command := range commands {
		buf.WriteString(fmt.Sprintf(`
%s
`, command,
		))
	}
	return base64.StdEncoding.EncodeToString([]byte(buf.String()))
}

func waitForContainerCmd(name string) string {
	command := `while ! test -f "` + shareddir + `/` + name + `.done"; do
	sleep 1
	done
	`
	return command
}

func getConfigfile(build *Build) (*yaml.Config, error) {
	yamldata, err := github.GetConfigFile(build.Owner, build.Repo, build.Commit)
	if err != nil {
		fmt.Println(err)
		log.Fatal(err)
	}

	cfg, err := yaml.ParseBytes(yamldata)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse .ci.yaml file")
	}
	return cfg, nil
}

func createBuildSteps(build *Build, cfg *yaml.Config) ([]v1.Container, error) {
	fmt.Println("constructing build")
	count := 0

	var containers []v1.Container
	for _, cont := range cfg.Pipeline.Containers {

		var cmds []string
		projectName := build.Owner + "/" + build.Repo
		// first command should be the wait for containers+
		cmds = append(cmds, waitForContainerCmd("git"))

		if count > 0 {
			cmds = append(cmds, waitForContainerCmd(fmt.Sprintf("build%v", count-1))) // wait for the previous build step
		}
		cmds = append(cmds, fmt.Sprintf("cd "+shareddir+"/go/src/%v", projectName))
		for _, v := range cont.Commands {
			cmds = append(cmds, v)
		}
		doneCmd := doneCmd(count)
		cmds = append(cmds, doneCmd)

		// Set environment variables
		var buildEnv []v1.EnvVar
		buildEnv = append(buildEnv, v1.EnvVar{Name: "CI_SCRIPT", Value: generateScript(cmds)})
		buildEnv = append(buildEnv, v1.EnvVar{Name: "GOPATH", Value: shareddir + "/go"})
		buildEnv = append(buildEnv, v1.EnvVar{Name: "GIT_REF", Value: build.Commit})

		for key, value := range cont.Environment {
			buildEnv = append(buildEnv, v1.EnvVar{Name: key, Value: value})
		}

		c := v1.Container{
			Name:            cont.Name,
			ImagePullPolicy: v1.PullIfNotPresent,
			Image:           cont.Image,
			VolumeMounts: []v1.VolumeMount{
				{
					Name:      "shared-data",
					MountPath: shareddir,
				},
			},
			Env:     buildEnv,
			Command: []string{"/bin/sh", "-c", "echo $CI_SCRIPT | base64 -d |/bin/sh -e"},
			//WorkingDir:,
			Lifecycle: &v1.Lifecycle{

				PreStop: &v1.Handler{Exec: &v1.ExecAction{Command: []string{"/usr/bin/touch", fmt.Sprintf("%v/build%v.done", shareddir, count)}}},
			},
		}
		if cont.Privileged {
			c.VolumeMounts = append(c.VolumeMounts, v1.VolumeMount{
				Name:      "docker-socket",
				MountPath: "/var/run/docker.sock",
			})
		}
		containers = append(containers, c)
		count++
	}
	return containers, nil
}

func doneCmd(count int) string {
	doneStr := fmt.Sprintf("build%v", count)
	doneCmd := "touch " + shareddir + "/" + doneStr + ".done"
	return doneCmd
}

func createGitContainer(build *Build) v1.Container {

	projectName := build.Owner + "/" + build.Repo
	url := "https://github.com/" + projectName
	workspace := shareddir + "/go/src/" + projectName
	cloneCmd := fmt.Sprintf("git clone %v %v", url, workspace)
	curWDCmd := fmt.Sprintf("cd %v", workspace)
	checkoutCmd := fmt.Sprintf("git checkout %v", build.Commit)
	doneCmd := "touch " + shareddir + "/git.done"
	cmds := []string{cloneCmd, curWDCmd, checkoutCmd, doneCmd}
	return v1.Container{
		Name:            "git",
		Image:           "sorenmat/git:1.0",
		ImagePullPolicy: v1.PullIfNotPresent,
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      "shared-data",
				MountPath: shareddir,
			},
		},
		Command: []string{"/bin/sh", "-c", "echo $CI_SCRIPT | base64 -d |/bin/sh -e"},

		Env: []v1.EnvVar{
			{Name: "CI_SCRIPT", Value: generateScript(cmds)},
		},
		Lifecycle: &v1.Lifecycle{
			PreStop: &v1.Handler{Exec: &v1.ExecAction{Command: []string{"/usr/bin/touch", shareddir + "/git.done"}}},
		},
	}
}

func executeBuild(kubectl *kubernetes.Clientset, build *Build, repo *Repo) error {
	pod := &v1.Pod{Spec: v1.PodSpec{
		RestartPolicy: "Never",
	}}
	buildname := "build-" + uuid.New()

	buildID := -1
	err := repo.db.Bolt.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("builds"))
		if err != nil {
			fmt.Println(err)
		}
		id, _ := b.NextSequence()
		buildID = int(id)
		return nil
	})
	if err != nil || buildID == -1 {
		return errors.Wrap(err, "unable to auto increment build number")
	}
	build.Number = buildID

	err = repo.Save(build)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to save build %v", build))
	}

	fmt.Println("Scheduling build: ", buildname)
	pod.ObjectMeta.Name = buildname
	vol1 := v1.Volume{}
	vol1.Name = "shared-data"
	vol1.EmptyDir = &v1.EmptyDirVolumeSource{}

	dockerSocket := v1.Volume{
		Name: "docker-socket",
	}
	dockerSocket.HostPath = &v1.HostPathVolumeSource{
		Path: "/var/run/docker.sock",
	}

	pod.Spec.Volumes = []v1.Volume{
		vol1,
		dockerSocket,
	}

	// add container to the pod
	cfg, err := getConfigfile(build)
	if err != nil {
		return errors.Wrap(err, "unable to handle buildconfig file")
	}

	buildSteps, err := createBuildSteps(build, cfg)
	if err != nil {
		return errors.Wrap(err, "unable to create build steps")
	}
	pod.Spec.Containers = append(pod.Spec.Containers, createGitContainer(build))
	pod.Spec.Containers = append(pod.Spec.Containers, buildSteps...)

	_, err = kubectl.CoreV1().Pods("default").Create(pod)
	if err != nil {
		log.Fatalf("Error starting build: %v", err)
	}
	build.Status = "Started"
	err = repo.Save(build)
	if err != nil {
		return errors.Wrap(err, "Error while waiting for container ")
	}

	// replace above sleep with a polling of the container ready state
	// perhaps replace with a listen hook
	waitForContainer(kubectl, buildname)
	build.Status = "Running"
	err = repo.Save(build)
	if err != nil {
		log.Fatal("Error while waiting for container ", err)
	}
	for _, cc := range pod.Spec.Containers {
		step := &Step{Name: cc.Name, Status: "Running"}
		build.Steps = append(build.Steps, step)
		repo.Save(build)
		bw := &BucketWriter{build: build, repo: repo, Step: cc.Name}
		err = getLog(kubectl, buildname, cc.Name, bw)
		if err != nil {
			log.Println("Error while getting log ", err)
		}
		//repo = getRepo(repo.db, repo.Name)
		step.Status = "Done"
		repo.Save(build)
	}

	build.Success = true
	repo.Save(build)
	/*
		// clean up
		err = kubectl.Pods("default").Delete(buildname, &meta_v1.DeleteOptions{})
		if err != nil {
			build.Fatal("Error while deleing pod: ", err)
		}
		fmt.Println("Pod deleted!")
	*/
	return nil
}

func waitForContainer(kubectl *kubernetes.Clientset, buildname string) error {
	count := 0
	for {
		pod, err := kubectl.CoreV1().Pods("default").Get(buildname, meta_v1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "unable to get pod, while waiting for it")
		}
		if pod.Status.Phase == "Running" || pod.Status.Phase == "Succeeded" || pod.Status.Phase == "Failed" {
			return nil
		}
		fmt.Println(pod.Status.Reason)
		count++
		time.Sleep(2 * time.Second)
	}
}

// getLog get the build of container in a running pod
func getLog(clientset *kubernetes.Clientset, pod string, container string, bw *BucketWriter) error {
	fmt.Printf("Trying to get log for %v %v\n", pod, container)
	req := clientset.CoreV1().Pods("default").GetLogs(pod, &v1.PodLogOptions{
		Container: container,
		Follow:    true,
		//Timestamps: true,
	})
	readCloser, err := req.Stream()
	if err != nil {
		return errors.Wrap(err, "unable to get stream: ")
	}
	defer readCloser.Close()
	_, err = io.Copy(io.MultiWriter(bw, os.Stdout), readCloser) // Tee something
	if err != nil {
		return errors.Wrap(err, "unable to copy stream to stdout")
	}
	return err
}
