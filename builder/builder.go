package builder

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"time"

	"strings"

	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.com/sorenmat/pipeline/pipeline/frontend/yaml"
	"gitlab.com/sorenmat/seneferu/builder/date"
	"gitlab.com/sorenmat/seneferu/github"
	"gitlab.com/sorenmat/seneferu/model"
	"gitlab.com/sorenmat/seneferu/storage"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const shareddir = "/share"

const SSHKEY = "sshkey"

// CreateSSHKeySecret creates and ssh key in the Kubernetes cluster to be used for
// cloning repositories
func CreateSSHKeySecret(kubectl *kubernetes.Clientset, sshkey string, namespace string) error {
	key, err := base64.StdEncoding.DecodeString(string(sshkey))
	if err != nil {
		return errors.Wrap(err, "unable to decode base64 encoded sshkey, is it encoded?")
	}
	cm := v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{Name: SSHKEY},
		Data:       map[string][]byte{"id_rsa": key},
	}
	cm.Namespace = namespace
	_, err = kubectl.CoreV1().Secrets(namespace).Create(&cm)
	if err != nil {
		log.Println("was unable to create ssh key in ", namespace)
	}

	// get docker secret from default NS and copy it to the build namespace
	sd, err := kubectl.CoreV1().Secrets("default").Get("seneferu-docker", meta_v1.GetOptions{})
	if err != nil {
		log.Println("was unable to get seneferu docker key")
	}

	sd.Namespace = namespace
	sd.ResourceVersion = ""
	sd.GenerateName = ""
	sd.Generation = 0
	_, err = kubectl.CoreV1().Secrets(namespace).Create(sd)
	if err != nil {
		log.Println("was unable to create docker secret in ", namespace)
	}
	waitForSecret(kubectl, sd.Name, namespace)
	waitForSecret(kubectl, SSHKEY, namespace)

	return err
}

func volumemounts() []v1.Volume {
	// shared directory for the build
	vol1 := v1.Volume{}
	vol1.Name = "shared-data"
	vol1.EmptyDir = &v1.EmptyDirVolumeSource{}

	volSSHAgent := v1.Volume{}
	volSSHAgent.Name = "ssh-agent"
	volSSHAgent.EmptyDir = &v1.EmptyDirVolumeSource{}

	sshvol := v1.Volume{}
	sshvol.Name = "sshvolume"
	perm := int32(0400)
	sshvol.Secret = &v1.SecretVolumeSource{SecretName: SSHKEY, DefaultMode: &perm}

	// Secret containing the docker registry certificates
	dockerSecrets := v1.Volume{}
	dockerSecrets.Name = "docker-secrets"
	dockerSecrets.Secret = &v1.SecretVolumeSource{SecretName: "seneferu-docker", DefaultMode: &perm}

	return []v1.Volume{
		vol1,
		sshvol,
		volSSHAgent,
		dockerSecrets,
	}
}

func ExecuteBuild(kubectl *kubernetes.Clientset, service storage.Service, build *model.Build, repo *model.Repo, token string, targetURL string, dockerRegHost string, sshkey string) error {
	pod := &v1.Pod{Spec: v1.PodSpec{
		RestartPolicy: "Never",
	},
		ObjectMeta: meta_v1.ObjectMeta{Labels: map[string]string{
			"type": "build",
			"app":  "seneferu-build",
		}},
	}

	buildUUID := "build-" + uuid.New()

	buildNumber, err := service.GetNextBuildNumber(build.Org, build.Name)
	build.Number = buildNumber
	if err != nil {
		return errors.Wrap(err, "unable to get next build number...")
	}
	err = service.SaveBuild(build)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to save build %v", build))
	}

	log.Println("Scheduling build: ", buildUUID)
	pod.ObjectMeta.Name = buildUUID

	pod.Spec.Volumes = volumemounts()
	// add container to the pod
	cfg, err := getConfigfile(build, token)
	if err != nil {
		github.ReportBack(github.GithubStatus{State: "error", Context: "fetching or parsing .ci.yaml"}, build.StatusURL, build.Commit, token)

		return errors.Wrap(err, "unable to handle buildconfig file")
	}
	ns := &v1.Namespace{}
	ns.Name = pod.Name
	ns.Annotations = map[string]string{"type": "build", "managedby": "seneferu"}
	ns.Namespace = pod.Name
	_, err = kubectl.CoreV1().Namespaces().Create(ns)
	if err != nil {
		return errors.Wrapf(err, "Error creating namespace %v: %v", ns, err)
	}
	waitForNamespace(kubectl, ns.Name)
	defer cleanupNamespace(kubectl, ns.Name)

	err = CreateSSHKeySecret(kubectl, sshkey, ns.Name)
	if err != nil {
		log.Fatal("Unable to create or update secret 'sshkey': ", err)
	}

	if cfg.Workspace.Path == "" {
		cfg.Workspace.Path = build.Name
	}
	var prepareSteps []v1.Container
	var buildSteps []v1.Container

	prepareSteps = append(prepareSteps, createSSHAgentContainer())
	prepareSteps = append(prepareSteps, createGitContainer(build, cfg.Workspace.Path))

	x, err := createBuildSteps(build, cfg)
	buildSteps = append(buildSteps, x...)
	if err != nil {
		return errors.Wrap(err, "unable to create build steps")
	}

	services, err := createServiceSteps(cfg)
	// Add the docker containers that writes in shardir to create the socket
	services = append(services, createDockerContainer(dockerRegHost))
	if err != nil {
		return errors.Wrap(err, "unable to create service")
	}
	// call the prepareSteps as init containers
	pod.Spec.InitContainers = prepareSteps
	pod.Spec.Containers = append(pod.Spec.Containers, services...)
	pod.Spec.Containers = append(pod.Spec.Containers, buildSteps...)

	pod.Namespace = ns.Name
	_, err = kubectl.CoreV1().Pods(ns.Name).Create(pod)
	if err != nil {
		return errors.Wrapf(err, "Error starting build: %v", err)
	}
	build.Status = "Started"
	err = service.SaveBuild(build)
	if err != nil {
		return errors.Wrap(err, "unable to save build")
	}
	for _, v := range buildSteps {
		err := github.ReportBack(github.GithubStatus{State: "pending", Context: v.Name}, build.StatusURL, build.Commit, token)
		if err != nil {
			log.Println("unable to report status back to github")
		}
	}

	// replace above sleep with a polling of the container ready state
	// perhaps replace with a listen hook
	err = waitForContainer(kubectl, buildUUID, ns.Name)
	if err != nil {
		for _, v := range buildSteps {
			err := github.ReportBack(github.GithubStatus{State: "error", Context: v.Name}, build.StatusURL, build.Commit, token)
			if err != nil {
				log.Println("unable to report status back to github")
			}
		}

		gerr := github.ReportBack(github.GithubStatus{State: "error", Context: "build failed to start"}, build.StatusURL, build.Commit, token)
		if gerr != nil {
			log.Println("unable to report status back to github about build unable to start")
		}
		return err
	}
	build.Status = "Running"
	err = service.SaveBuild(build)
	if err != nil {
		return errors.Wrap(err, "Error while waiting for container ")
	}

	for _, b := range buildSteps {
		step := &model.Step{StepInfo: model.StepInfo{Name: b.Name, Reponame: build.Name, BuildNumber: build.Number, Org: build.Org, Status: "Running"}}
		build.Steps = append(build.Steps, step)
		err = service.SaveBuild(build)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to save build %v", build))
		}
		err = service.SaveStep(step)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to save build step %v", b.Name))
		}

		go registerLog(service, repo.Org, repo.Name, step, buildUUID, b.Name, build, kubectl, ns.Name)
	}
	for _, b := range services {
		s := &model.Service{Name: b.Name}
		build.Services = append(build.Services, s)
		err := service.SaveBuild(build)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to save build %v", build))
		}
		go registerLogForService(service, repo.Org, repo.Name, buildUUID, b.Name, build, kubectl, ns.Name)
	}

	//perhaps wait for pod to be in Completed or Error state

	// wait for all the build steps to finish
	log.Println("Waiting for build steps...")
	for _, b := range buildSteps {
		waitForContainerTermination(kubectl, b, buildUUID, ns.Name)
	}

	build.Success = true
	for _, b := range buildSteps {
		for _, step := range build.Steps {
			if step.Name == b.Name {
				exitCode, _ := waitForContainerTermination(kubectl, b, buildUUID, ns.Name)
				if exitCode > 0 {
					build.Status = "Failed"
					build.Success = false
					step.Status = "Failed"
				}
				step.ExitCode = exitCode
				var state string
				if exitCode == 0 {
					state = "success"
				} else {
					state = "error"
				}

				callbackURL := fmt.Sprintf("%v/repo/%v/%v/build/%v/step/%v", targetURL, build.Org, build.Name, build.Number, step.Name)
				err := github.ReportBack(github.GithubStatus{State: state, Context: step.Name, TargetURL: callbackURL}, build.StatusURL, build.Commit, token)
				if err != nil {
					log.Println("unable to report status back to github")
				}

			}
		}
	}

	log.Println("All build steps done...")

	// TODO fix this
	build.Status = "Done"
	err = service.SaveBuild(build)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to save build %v", build))
	}

	// Fetch coverage configuration from settings
	var testCoverage string
	for _, c := range cfg.Pipeline.Containers {
		if c.Coverage != "" {
			testCoverage = c.Coverage
			break
		}
	}

	// Add coverage to build
	coverage := getCoverageFromLogs(build, buildNumber, testCoverage)
	build.Coverage = coverage

	// calculate the time the build took
	t := time.Now().Sub(build.Timestamp)
	build.Duration = format.Duration(t)

	err = service.SaveBuild(build)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to save build %v", build))
	}

	return nil
}

func cleanupNamespace(kubectl *kubernetes.Clientset, namespace string) {
	log.Printf("clean up of namespace %v started", namespace)
	err := kubectl.CoreV1().Namespaces().Delete(namespace, &meta_v1.DeleteOptions{})
	if err != nil {
		log.Println("Error while deleing namespace: ", err)
	}
	log.Printf("Namespace %v deleted!", namespace)
}

// generateScript is a helper function that generates a build script and base64 encode it.
func generateScript(commands []string) string {
	var buf bytes.Buffer
	for _, command := range commands {
		buf.WriteString(fmt.Sprintf("echo '%s'\n%s\n", command, command))
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

func getConfigfile(build *model.Build, token string) (*yaml.Config, error) {
	yamldata, err := github.GetConfigFile(build.TreesURL, build.Commit, token)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch config file from github")
	}

	cfg, err := yaml.ParseBytes(yamldata)
	if err != nil {
		return nil, errors.Wrap(err, "unable to parse .ci.yaml file")
	}
	return cfg, nil
}

func doneCmd(count int) string {
	doneStr := fmt.Sprintf("build%v", count)
	touchStr := "touch " + shareddir + "/" + doneStr + ".done;"
	doneCmd := `clean() { rc=$?; ` + touchStr + ` exit $rc; }; trap clean EXIT`
	return doneCmd
}

func getCoverageFromLogs(build *model.Build, buildNumber int, testCoverage string) string {
	// Search for the coverage regex in the logs, and add it to the build if found
	for _, s := range build.Steps {
		r, err := regexp.Compile(testCoverage)
		if err != nil {
			log.Println(err)
			return ""
		}
		if result := r.FindString(s.Log); result != "" {
			return result
		}
	}
	return ""
}

func createBuildSteps(build *model.Build, cfg *yaml.Config) ([]v1.Container, error) {
	log.Println("constructing build")
	count := 0

	var containers []v1.Container
	for _, cont := range cfg.Pipeline.Containers {
		// strip "refs/heads/"
		branch := strings.Replace(build.Ref, "refs/heads/", "", -1)
		if !cont.Constraints.Branch.Match(branch) {
			log.Printf("Branch %v didn't meet condition of %v\n", build.Ref, cont.Constraints.Branch)
			continue
		}
		var cmds []string
		// first command should be the wait for containers+
		cmds = append(cmds, waitForContainerCmd("git"))

		if count > 0 {
			cmds = append(cmds, waitForContainerCmd(fmt.Sprintf("build%v", count-1))) // wait for the previous build step
		}

		workspace := cfg.Workspace.Path
		if workspace == "" {
			workspace = build.Name
		}
		workspace = shareddir + "/" + workspace

		doneCmd := doneCmd(count)
		cmds = append(cmds, doneCmd)

		for _, v := range cont.Commands {
			cmds = append(cmds, v)
		}

		// Set environment variables
		var buildEnv []v1.EnvVar
		buildEnv = append(buildEnv, v1.EnvVar{Name: "CI_SCRIPT", Value: generateScript(cmds)})
		buildEnv = append(buildEnv, v1.EnvVar{Name: "GOPATH", Value: shareddir + "/go"})
		buildEnv = append(buildEnv, v1.EnvVar{Name: "GIT_REF", Value: build.Commit})

		buildEnv = append(buildEnv, v1.EnvVar{Name: "DOCKER_HOST", Value: fmt.Sprintf("unix:///%v/docker.sock", shareddir)})

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
				{
					Name:      "sshvolume",
					MountPath: "/root/.ssh", //TODO this needs to be fixed

				},
			},
			Env:        buildEnv,
			WorkingDir: workspace,
			Lifecycle: &v1.Lifecycle{
				PreStop: &v1.Handler{Exec: &v1.ExecAction{Command: []string{fmt.Sprintf("/usr/bin/touch %v/build%v.done", shareddir, count)}}},
			},
		}

		if len(cont.Args) > 0 {
			c.Args = cont.Args
		}
		// There we explicit commands defined in the build configuration file
		if len(cont.Commands) > 0 {
			c.Command = []string{"/bin/sh", "-c", "echo $CI_SCRIPT | base64 -d |/bin/sh -e"}
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
		// handle docker in th services..
		containers = append(containers, c)
	}
	return containers, nil
}

func createGitContainer(build *model.Build, workspace string) v1.Container {

	projectName := build.Org + "/" + build.Name
	url := "git@github.com:" + projectName

	workspace = shareddir + "/" + workspace

	sshTrustCmd := "ssh-keyscan -t rsa github.com > ~/.ssh/known_hosts"
	cloneCmd := fmt.Sprintf("git clone %v %v", url, workspace)
	curWDCmd := fmt.Sprintf("cd %v", workspace)

	// TODO take the last element of the refs/head/init this is most likely not a good idea
	branch := ""
	if strings.Contains(build.Ref, "/") {
		branch = strings.Split(build.Ref, "/")[2]
	} else {
		// on a pullrequestevent we only get the branchname
		branch = build.Ref
	}
	checkoutCmd := fmt.Sprintf("git checkout %v", branch)
	log.Println(checkoutCmd)

	exportSSH := "SSH_AUTH_SOCK=/share/socket; export SSH_AUTH_SOCK"
	doneCmd := "touch " + shareddir + "/git.done"

	sshcmds := []string{
		"echo starting ssh-agent",
		"eval $(ssh-agent -a /share/socket1)",
		"mkdir ~/.ssh",
		"echo trying ssh-add",
		"cp /ssh/id_rsa ~/.ssh/id_rsa",
		"ssh-add ~/.ssh/id_rsa",
	}
	cmds := append(sshcmds, exportSSH, sshTrustCmd, cloneCmd, curWDCmd, checkoutCmd, doneCmd)

	return v1.Container{
		Name:            "git",
		Image:           "sorenmat/git:1.0",
		ImagePullPolicy: v1.PullIfNotPresent,
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      "shared-data",
				MountPath: shareddir,
			},
			{
				Name:      "ssh-agent",
				MountPath: "/.ssh-agent",
			},
			{
				Name:      "sshvolume",
				MountPath: "/ssh",
				ReadOnly:  false,
			},
		},

		Command: []string{"/bin/sh", "-c", "echo $CI_SCRIPT | base64 -d |/bin/sh -e"},

		Env: []v1.EnvVar{
			{Name: "SSH_AUTH_SOCK", Value: "/.ssh-agent/socket"},
			{Name: "CI_SCRIPT", Value: generateScript(cmds)},
		},
	}
}

func createSSHAgentContainer() v1.Container {
	doneCmd := "touch " + shareddir + "/ssh-key-add.done"
	cmds := []string{
		"echo starting ssh-agent",
		"eval $(ssh-agent -a /share/socket)",
		"echo trying ssh-add",
		"cp /root/.ssh/id_rsa /share/id_rsa",
		"ssh-add /share/id_rsa",
		"echo 'marking step as done'",
		doneCmd,
	}

	return v1.Container{
		Name:            "ssh-agent",
		Image:           "nardeas/ssh-agent",
		ImagePullPolicy: v1.PullIfNotPresent,
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      "shared-data",
				MountPath: shareddir,
			},
			{
				Name:      "sshvolume",
				MountPath: "/root/.ssh",
				ReadOnly:  false,
			},
			{
				Name:      "ssh-agent",
				MountPath: "/.ssh-agent",
			},
		},

		Command: []string{"/bin/sh", "-c", "echo $CI_SCRIPT | base64 -d |/bin/sh -e"},

		Env: []v1.EnvVar{
			{Name: "CI_SCRIPT", Value: generateScript(cmds)},
		},
	}
}

func createDockerContainer(dockerRegHost string) v1.Container {
	priv := true
	return v1.Container{
		Name:            "docker",
		Image:           "docker:17-dind",
		ImagePullPolicy: v1.PullIfNotPresent,
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      "shared-data",
				MountPath: "/var/run",
			},
			{
				Name:      "docker-secrets",
				MountPath: "/etc/docker/certs.d/" + dockerRegHost,
			},
		},
		SecurityContext: &v1.SecurityContext{
			Privileged: &priv,
		},
	}
}

func waitForContainerTermination(kubectl *kubernetes.Clientset, b v1.Container, buildUUID string, namespace string) (int32, error) {
	for {
		pod, err := kubectl.CoreV1().Pods(namespace).Get(buildUUID, meta_v1.GetOptions{})
		if err != nil {
			return -1, errors.Wrap(err, "unable to get pod, while waiting for it")
		}
		reason, err := printPod(pod)
		if err != nil {
			log.Println("print pod error", err)
		}
		log.Println("REASON: ", reason)

		for _, v := range pod.Status.ContainerStatuses {
			if v.Name == b.Name {
				if v.State.Terminated != nil && v.State.Terminated.Reason != "" {
					return v.State.Terminated.ExitCode, nil
				}
				time.Sleep(2 * time.Second)
			}
		}
	}
}

func registerLog(service storage.Service, org string, reponame string, step *model.Step, buildUUID string, name string, build *model.Build, kubectl *kubernetes.Clientset, namespace string) error {
	// start watching the logs in a separate go routine
	err := saveLog(kubectl, buildUUID, name, step, namespace)
	if err != nil {
		log.Println("Error while getting log ", err)
	}
	step.Status = "Done"
	err = service.SaveStep(step)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to save build step %v", step))
	}
	return nil
}

func registerLogForService(service storage.Service, org string, reponame string, buildUUID string, name string, build *model.Build, kubectl *kubernetes.Clientset, namespace string) error {
	// get the log without waiting, since its a service and it should be running for ever...
	err := saveLog(kubectl, buildUUID, name, nil, namespace)
	if err != nil {
		log.Println("Error while getting log ", err)
	}
	err = service.SaveBuild(build)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to save build %v", build))
	}
	return nil
}

func waitForContainer(kubectl *kubernetes.Clientset, buildname string, namespace string) error {
	for {
		pod, err := kubectl.CoreV1().Pods(namespace).Get(buildname, meta_v1.GetOptions{})
		if err != nil {
			return errors.Wrap(err, "unable to get pod, while waiting for it")
		}
		reason, err := printPod(pod)
		if err != nil {
			log.Println("print pod error", err)
			return err
		}
		if reason == "Init:Error" {
			return fmt.Errorf(reason)
		}
		if reason == "Running" {
			return nil
		}
		if reason == "Failed" {
			return fmt.Errorf(reason)
		}
		log.Println("unknown reason state", reason)
		if pod.Status.Phase == "Running" || pod.Status.Phase == "Succeeded" || pod.Status.Phase == "Failed" {
			return nil
		}
		log.Println("Waitting for " + buildname + " " + pod.Status.Reason)
		time.Sleep(2 * time.Second)
	}
}

// saveLog get the build of container in a running pod
func saveLog(kubectl *kubernetes.Clientset, pod string, container string, step *model.Step, namespace string) error {
	log.Printf("Trying to get log for %v %v\n", pod, container)
	req := kubectl.CoreV1().Pods(namespace).GetLogs(pod, &v1.PodLogOptions{
		Container: container,
		Follow:    true,
		//Timestamps: true,
	})
	readCloser, err := req.Stream()
	if err != nil {
		return errors.Wrap(err, "unable to get stream: ")
	}
	defer readCloser.Close()
	dbw := DBLogWriter{step: step}
	_, err = io.Copy(io.MultiWriter(os.Stdout, dbw), readCloser)
	if err != nil {
		return errors.Wrap(err, "unable to copy stream to stdout")
	}
	return err
}

type DBLogWriter struct {
	step *model.Step
}

func (d DBLogWriter) Write(p []byte) (n int, err error) {
	if d.step == nil {
		return 0, nil
	}
	d.step.Log = d.step.Log + string(p)
	return len(p), nil
}

func waitForSecret(kubectl *kubernetes.Clientset, name, namespace string) bool {
	counter := 0
	for {
		sd, err := kubectl.CoreV1().Secrets(namespace).Get(name, meta_v1.GetOptions{})
		if err != nil {
			log.Printf("was unable to get %v in %v\n", name, namespace)
		}
		if sd.Name == name {
			return true
		}
		time.Sleep(1 * time.Second)
		counter++
		if counter == 120 {
			return false
		}
	}
}

func waitForNamespace(kubectl *kubernetes.Clientset, name string) bool {
	counter := 0
	for {
		ns, err := kubectl.CoreV1().Namespaces().Get(name, meta_v1.GetOptions{})
		if err != nil {
			log.Printf("was unable to get namespace %v\n", name)
		}
		if ns.Name == name {
			return true
		}
		time.Sleep(1 * time.Second)
		counter++
		if counter == 120 {
			return false
		}

	}
}

func printPod(pod *v1.Pod) (string, error) {
	restarts := 0
	readyContainers := 0

	reason := string(pod.Status.Phase)
	if pod.Status.Reason != "" {
		reason = pod.Status.Reason
	}

	initializing := false

	for i := range pod.Status.InitContainerStatuses {
		container := pod.Status.InitContainerStatuses[i]
		restarts += int(container.RestartCount)
		switch {
		case container.State.Terminated != nil && container.State.Terminated.ExitCode == 0:
			continue
		case container.State.Terminated != nil:
			// initialization is failed
			if len(container.State.Terminated.Reason) == 0 {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Init:Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("Init:ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else {
				reason = "Init:" + container.State.Terminated.Reason
			}
			initializing = true
		case container.State.Waiting != nil && len(container.State.Waiting.Reason) > 0 && container.State.Waiting.Reason != "PodInitializing":
			reason = "Init:" + container.State.Waiting.Reason
			initializing = true
		default:
			reason = fmt.Sprintf("Init:%d/%d", i, len(pod.Spec.InitContainers))
			initializing = true
		}
		break
	}

	if !initializing {
		restarts = 0
		hasRunning := false
		for i := len(pod.Status.ContainerStatuses) - 1; i >= 0; i-- {
			container := pod.Status.ContainerStatuses[i]

			restarts += int(container.RestartCount)
			if container.State.Waiting != nil && container.State.Waiting.Reason != "" {
				reason = container.State.Waiting.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason != "" {
				reason = container.State.Terminated.Reason
			} else if container.State.Terminated != nil && container.State.Terminated.Reason == "" {
				if container.State.Terminated.Signal != 0 {
					reason = fmt.Sprintf("Signal:%d", container.State.Terminated.Signal)
				} else {
					reason = fmt.Sprintf("ExitCode:%d", container.State.Terminated.ExitCode)
				}
			} else if container.Ready && container.State.Running != nil {
				hasRunning = true
				readyContainers++
			}
		}

		// change pod status back to "Running" if there is at least one container still reporting as "Running" status
		if reason == "Completed" && hasRunning {
			reason = "Running"
		}
	}
	if pod.DeletionTimestamp != nil {
		reason = "Terminating"
	}

	return reason, nil
}
