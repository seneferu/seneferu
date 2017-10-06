package main

import (
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"time"

	"github.com/asdine/storm"
	"github.com/boltdb/bolt"
	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.com/sorenmat/pipeline/pipeline/frontend/yaml"
	"gitlab.com/sorenmat/ci-server/github"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"strconv"
	"strings"
)

const shareddir = "/share"

var (
	kubeCfgFile  = flag.String("kubeconfig", "", "Kubernetes Config File")
	githubSecret = flag.String("githubsecret", "", "Github secret token, needs to match the one on Github ")
	helmHost     = flag.String("helmhost", "", "Hostname and port of the Helm host / tiller")
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
		*helmHost = os.Getenv("githubsecret")
	}
	if *helmHost == "" {
		log.Fatal("helmHost can't be empty")
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
	startWebServer(db, kubectl, *githubSecret, *helmHost)
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

func createServiceSteps(cfg *yaml.Config) ([]v1.Container, error) {

	var containers []v1.Container
	for _, serv := range cfg.Services.Containers {
		c := v1.Container{
			Name:            serv.Name,
			ImagePullPolicy: v1.PullIfNotPresent,
			Image:           serv.Image,
		}
		if serv.Privileged {
			c.VolumeMounts = append(c.VolumeMounts, v1.VolumeMount{
				Name:      "docker-socket",
				MountPath: "/var/run/docker.sock",
			})
		}
		containers = append(containers, c)
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
	buildUUID := "build-" + uuid.New()

	buildNumber := -1
	err := repo.db.Bolt.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists([]byte("builds"))
		if err != nil {
			fmt.Println(err)
		}
		id, _ := b.NextSequence()
		buildNumber = int(id)
		return nil
	})
	if err != nil || buildNumber == -1 {
		return errors.Wrap(err, "unable to auto increment build number")
	}
	build.Number = buildNumber

	err = repo.Save(build)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to save build %v", build))
	}

	fmt.Println("Scheduling build: ", buildUUID)
	pod.ObjectMeta.Name = buildUUID
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

	var buildSteps []v1.Container
	buildSteps = append(buildSteps, createGitContainer(build))
	x, err := createBuildSteps(build, cfg)
	buildSteps = append(buildSteps, x...)
	if err != nil {
		return errors.Wrap(err, "unable to create build steps")
	}

	services, err := createServiceSteps(cfg)
	if err != nil {
		return errors.Wrap(err, "unable to create service")
	}

	pod.Spec.Containers = append(pod.Spec.Containers, services...)
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
	waitForContainer(kubectl, buildUUID)
	build.Status = "Running"
	err = repo.Save(build)
	if err != nil {
		return errors.Wrap(err, "Error while waiting for container ")
	}

	for _, b := range buildSteps {
		step := &Step{Name: b.Name, Status: "Running"}
		build.Steps = append(build.Steps, step)
		repo.Save(build)
		go registerLog(repo, step, buildUUID, b.Name, build, kubectl)
	}
	for _, b := range services {
		service := &Service{Name: b.Name}
		build.Services = append(build.Services, service)
		err := repo.Save(build)
		if err != nil {
			log.Println("unable to save service...")
		}
		go registerLogForService(repo, buildUUID, b.Name, build, kubectl)
	}

	// wait for all the build steps to finish
	fmt.Println("Waiting for build steps...")
	for _, b := range buildSteps {
		waitForContainerTermination(kubectl, b, buildUUID)
	}

	build.Success = true
	for _, b := range buildSteps {
		for _, step := range build.Steps {
			if step.Name == b.Name {
				exitCode, _ := waitForContainerTermination(kubectl, b, buildUUID)
				if exitCode > 0 {
					build.Status = "Failed"
					build.Success = false
					step.Status = "Failed"
				}
				step.ExitCode = exitCode
			}
		}
	}

	log.Println("All build steps done...")

	// TODO fix this
	build.Status = "Done"
	repo.Save(build)

	// Fetch coverage configuration from settings
	var testCoverage string
	for _, c := range cfg.Pipeline.Containers {
		if c.Coverage != "" {
			testCoverage = c.Coverage
			break
		}
	}

	// Add coverage to build
	coverage := getCoverageFromLogs(repo, buildNumber, testCoverage)
	build.Coverage = coverage

	// calculate the time the build took
	t := time.Now().Sub(build.Timestamp)
	build.Took = formatDuration(t)

	repo.Save(build)

	// clean up

	err = kubectl.CoreV1().Pods("default").Delete(buildUUID, &meta_v1.DeleteOptions{})
	if err != nil {
		log.Println("Error while deleing pod: ", err)
		return errors.Wrap(err, "Error while deleing pod")
	}

	log.Println("Pod deleted!")
	log.Println("*****************************************")
	log.Println("Delcaring build done")
	log.Println("*****************************************")
	return nil
}

func formatDuration(d time.Duration) string {
	// taken from github.com/hako/durafmt
	var (
		units = []string{"days", "hours", "minutes", "seconds"}
	)

	var duration string
	input := d.String()

	// Convert duration.
	seconds := int(d.Seconds()) % 60
	minutes := int(d.Minutes()) % 60
	hours := int(d.Hours()) % 24
	days := int(d/(24*time.Hour)) % 365 % 7
	// Create a map of the converted duration time.
	durationMap := map[string]int{
		"seconds": seconds,
		"minutes": minutes,
		"hours":   hours,
		"days":    days,
	}

	// Construct duration string.
	for _, u := range units {
		v := durationMap[u]
		strval := strconv.Itoa(v)
		switch {
		// add to the duration string if v > 1.
		case v > 1:
			duration += strval + " " + u + " "
			// remove the plural 's', if v is 1.
		case v == 1:
			duration += strval + " " + strings.TrimRight(u, "s") + " "
			// omit any value with 0s or 0.
		case d.String() == "0" || d.String() == "0s":
			// note: milliseconds and minutes have the same suffix (m)
			// so we have to check if the units match with the suffix.

			// check for a suffix that is NOT the milliseconds suffix.
			if strings.HasSuffix(input, string(u[0])) && !strings.Contains(input, "ms") {
				// if it happens that the units are milliseconds, skip.
				if u == "milliseconds" {
					continue
				}
				duration += strval + " " + u
			}
			break
			// omit any value with 0.
		case v == 0:
			continue
		}
	}
	// trim any remaining spaces.
	duration = strings.TrimSpace(duration)
	return duration
}

func waitForContainerTermination(kubectl *kubernetes.Clientset, b v1.Container, buildUUID string) (int32, error) {
	for {
		pod, err := kubectl.CoreV1().Pods("default").Get(buildUUID, meta_v1.GetOptions{})
		if err != nil {
			return -1, errors.Wrap(err, "unable to get pod, while waiting for it")
		}
		for _, v := range pod.Status.ContainerStatuses {
			if v.Name == b.Name {
				if v.State.Terminated != nil && v.State.Terminated.Reason != "" {
					return v.State.Terminated.ExitCode, nil
				} else {
					time.Sleep(2 * time.Second)
				}
			}
		}
	}
}

func registerLog(repo *Repo, step *Step, buildUUID string, name string, build *Build, kubectl *kubernetes.Clientset) {

	bw := &BucketWriter{build: build, repo: repo, Step: name}
	// start watching the logs in a separate go routine
	err := saveLog(kubectl, buildUUID, name, bw)
	if err != nil {
		log.Println("Error while getting log ", err)
	}
	step.Status = "Done"
	repo.Save(build)
}

func registerLogForService(repo *Repo, buildUUID string, name string, build *Build, kubectl *kubernetes.Clientset) {
	bw := &BucketWriter{build: build, repo: repo, Step: name}
	// get the log without waiting, since its a service and it should be running for ever...
	err := saveLog(kubectl, buildUUID, name, bw)
	if err != nil {
		log.Println("Error while getting log ", err)
	}
	repo.Save(build)
}

func getCoverageFromLogs(repo *Repo, buildNumber int, testCoverage string) string {
	// Search for the coverage regex in the logs, and add it to the build if found
	for _, b := range repo.Build {
		if b.Number == buildNumber {
			for _, s := range b.Steps {
				r, err := regexp.Compile(testCoverage)
				if err != nil {
					log.Println(err)
					return ""
				}
				if result := r.FindString(s.Log); result != "" {
					return result
				}
			}
		}
	}
	return ""
}

func waitForContainer(kubectl *kubernetes.Clientset, buildname string) error {
	for {
		pod, err := kubectl.CoreV1().Pods("default").Get(buildname, meta_v1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "unable to get pod, while waiting for it")
		}
		if pod.Status.Phase == "Running" || pod.Status.Phase == "Succeeded" || pod.Status.Phase == "Failed" {
			return nil
		}
		log.Println(pod.Status.Reason)
		time.Sleep(2 * time.Second)
	}
}

// saveLog get the build of container in a running pod
func saveLog(clientset *kubernetes.Clientset, pod string, container string, bw *BucketWriter) error {
	log.Printf("Trying to get log for %v %v\n", pod, container)
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
