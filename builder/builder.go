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

	"strconv"
	"strings"

	"github.com/pborman/uuid"
	"github.com/pkg/errors"
	"github.com/sorenmat/pipeline/pipeline/frontend/yaml"
	"gitlab.com/sorenmat/seneferu/github"
	"gitlab.com/sorenmat/seneferu/model"
	"gitlab.com/sorenmat/seneferu/storage"
	"gitlab.com/sorenmat/seneferu/webstream"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const shareddir = "/share"

const SSHKEY = "sshkey"

// CreateSSHKeySecret creates and ssh key in the Kubernetes cluster to be used for
// cloning repositories
func CreateSSHKeySecret(kubectl *kubernetes.Clientset, sshkey string) error {
	key, err := base64.StdEncoding.DecodeString(string(sshkey))
	if err != nil {
		return errors.Wrap(err, "unable to decode base64 encoded sshkey, is it encoded?")
	}
	cm := v1.Secret{
		ObjectMeta: meta_v1.ObjectMeta{Name: SSHKEY},
		Data:       map[string][]byte{"id_rsa": key},
	}
	exsistingKey, _ := kubectl.CoreV1().Secrets("default").Get(SSHKEY, meta_v1.GetOptions{})
	if exsistingKey != nil {
		log.Println("sshkey exsists, will try to update it")
		_, err := kubectl.CoreV1().Secrets("default").Update(&cm)
		return err
	} else {
		log.Println("sshkey mssing, will try to create it")
		_, err := kubectl.CoreV1().Secrets("default").Create(&cm)
		return err
	}
}

func ExecuteBuild(kubectl *kubernetes.Clientset, service storage.Service, build *model.Build, repo *model.Repo, token string) error {
	pod := &v1.Pod{Spec: v1.PodSpec{
		RestartPolicy: "Never",
	}}
	buildUUID := "build-" + uuid.New()

	buildNumber, err := service.GetNextBuildNumber() // fix me
	build.Number = buildNumber
	if err != nil {
		return errors.Wrap(err, "unable to get next build number...")
	}
	err = service.SaveBuild(build)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to save build %v", build))
	}

	fmt.Println("Scheduling build: ", buildUUID)
	pod.ObjectMeta.Name = buildUUID

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

	dockerSocket := v1.Volume{
		Name: "docker-socket",
	}
	dockerSocket.HostPath = &v1.HostPathVolumeSource{
		Path: "/var/run/docker.sock",
	}

	pod.Spec.Volumes = []v1.Volume{
		vol1,
		sshvol,
		dockerSocket,
		volSSHAgent,
	}

	// add container to the pod
	cfg, err := getConfigfile(build, token)
	if err != nil {
		return errors.Wrap(err, "unable to handle buildconfig file")
	}
	var prepareSteps []v1.Container
	var buildSteps []v1.Container
	prepareSteps = append(prepareSteps, createSSHAgentContainer())
	prepareSteps = append(prepareSteps, createSSHKeyAdd())
	prepareSteps = append(prepareSteps, createGitContainer(build))
	x, err := createBuildSteps(build, cfg)
	buildSteps = append(buildSteps, x...)
	if err != nil {
		return errors.Wrap(err, "unable to create build steps")
	}

	services, err := createServiceSteps(cfg)
	if err != nil {
		return errors.Wrap(err, "unable to create service")
	}
	pod.Spec.Containers = append(pod.Spec.Containers, prepareSteps...)
	pod.Spec.Containers = append(pod.Spec.Containers, services...)
	pod.Spec.Containers = append(pod.Spec.Containers, buildSteps...)

	for _, v := range buildSteps {
		err := github.ReportBack(github.GithubStatus{State: "pending", Context: v.Name}, build.Org, build.Name, build.Commit, token)
		if err != nil {
			log.Println("unable to report status back to github")
		}
	}

	_, err = kubectl.CoreV1().Pods("default").Create(pod)
	if err != nil {
		return errors.Wrapf(err, "Error starting build: %v", err)
	}
	build.Status = "Started"
	err = service.SaveBuild(build)
	if err != nil {
		return errors.Wrap(err, "Error while waiting for container ")
	}

	// replace above sleep with a polling of the container ready state
	// perhaps replace with a listen hook
	waitForContainer(kubectl, buildUUID)
	build.Status = "Running"
	err = service.SaveBuild(build)
	if err != nil {
		return errors.Wrap(err, "Error while waiting for container ")
	}

	for _, b := range buildSteps {
		step := &model.Step{StepInfo: model.StepInfo{Name: b.Name, Status: "Running"}}
		build.Steps = append(build.Steps, step)
		err = service.SaveBuild(build)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to save build %v", build))
		}

		go registerLog(service, repo.Org, repo.Name, step, buildUUID, b.Name, build, kubectl)
	}
	for _, b := range services {
		s := &model.Service{Name: b.Name}
		build.Services = append(build.Services, s)
		err := service.SaveBuild(build)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("unable to save build %v", build))
		}
		go registerLogForService(service, repo.Org, repo.Name, buildUUID, b.Name, build, kubectl)
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
				var state string
				if exitCode == 0 {
					state = "success"
				} else {
					state = "error"
				}
				err := github.ReportBack(github.GithubStatus{State: state, Context: step.Name}, build.Org, build.Name, build.Commit, token)
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
	build.Duration = formatDuration(t)

	err = service.SaveBuild(build)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to save build %v", build))
	}

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

func getConfigfile(build *model.Build, token string) (*yaml.Config, error) {
	yamldata, err := github.GetConfigFile(build.Org, build.Name, build.Commit, token)
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

func doneCmd(count int) string {
	doneStr := fmt.Sprintf("build%v", count)
	doneCmd := "touch " + shareddir + "/" + doneStr + ".done"
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
	fmt.Println("constructing build")
	count := 0

	var containers []v1.Container
	for _, cont := range cfg.Pipeline.Containers {

		var cmds []string
		projectName := build.Org + "/" + build.Name
		// first command should be the wait for containers+
		cmds = append(cmds, waitForContainerCmd("git"))
		cmds = append(cmds, "ssh-keyscan -t rsa github.com > ~/.ssh/known_hosts")

		if count > 0 {
			cmds = append(cmds, waitForContainerCmd(fmt.Sprintf("build%v", count-1))) // wait for the previous build step
		}
		provider := "github.com"
		workspace := shareddir + "/go/src/" + provider + "/" + projectName

		cmds = append(cmds, fmt.Sprintf("cd %v", workspace))
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
				{
					Name:      "sshvolume",
					MountPath: "/root/.ssh",
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

func createGitContainer(build *model.Build) v1.Container {

	projectName := build.Org + "/" + build.Name
	url := "git@github.com:" + projectName
	provider := "github.com"
	workspace := shareddir + "/go/src/" + provider + "/" + projectName

	sshTrustCmd := "ssh-keyscan -t rsa github.com > ~/.ssh/known_hosts"
	cloneCmd := fmt.Sprintf("git clone %v %v", url, workspace)
	curWDCmd := fmt.Sprintf("cd %v", workspace)
	// TODO take the last element of the refs/head/init this is most likely not a good idea
	checkoutCmd := fmt.Sprintf("git checkout %v", strings.Split(build.Ref, "/")[2])
	fmt.Println(checkoutCmd)

	doneCmd := "touch " + shareddir + "/git.done"
	cmds := []string{waitForContainerCmd("ssh-key-add"), "mkdir ~/.ssh", sshTrustCmd, cloneCmd, curWDCmd, checkoutCmd, doneCmd}
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
		},

		Command: []string{"/bin/sh", "-c", "echo $CI_SCRIPT | base64 -d |/bin/sh -e"},

		Env: []v1.EnvVar{
			{Name: "SSH_AUTH_SOCK", Value: "/.ssh-agent/socket"},
			{Name: "CI_SCRIPT", Value: generateScript(cmds)},
		},
		Lifecycle: &v1.Lifecycle{
			PreStop: &v1.Handler{Exec: &v1.ExecAction{Command: []string{"/usr/bin/touch", shareddir + "/git.done"}}},
		},
	}
}

func createSSHAgentContainer() v1.Container {
	cmds := []string{"ssh-agent -a /.ssh-agent/socket -D"}
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

func createSSHKeyAdd() v1.Container {
	doneCmd := "touch " + shareddir + "/ssh-key-add.done"

	cmds := []string{"ssh-add /root/.ssh/id_rsa", doneCmd}
	return v1.Container{
		Name:            "ssh-key-add",
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

func registerLog(service storage.Service, org string, reponame string, step *model.Step, buildUUID string, name string, build *model.Build, kubectl *kubernetes.Clientset) error {
	//TODO THIS SEEMS WRONG
	bw := &webstream.BucketWriter{Service: service, Build: build, RepoID: org + reponame, Step: name}
	// start watching the logs in a separate go routine
	err := saveLog(kubectl, buildUUID, name, bw)
	if err != nil {
		log.Println("Error while getting log ", err)
	}
	step.Status = "Done"
	err = service.SaveBuild(build)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to save build %v", build))
	}
	return nil
}

func registerLogForService(service storage.Service, org string, reponame string, buildUUID string, name string, build *model.Build, kubectl *kubernetes.Clientset) error {
	//TODO THIS SEEMS WRONG
	bw := &webstream.BucketWriter{Service: service, Build: build, RepoID: org + reponame, Step: name}
	// get the log without waiting, since its a service and it should be running for ever...
	err := saveLog(kubectl, buildUUID, name, bw)
	if err != nil {
		log.Println("Error while getting log ", err)
	}
	err = service.SaveBuild(build)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unable to save build %v", build))
	}
	return nil
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
		log.Println("Waitting for " + buildname + " " + pod.Status.Reason)
		time.Sleep(2 * time.Second)
	}
}

// saveLog get the build of container in a running pod
func saveLog(kubectl *kubernetes.Clientset, pod string, container string, bw *webstream.BucketWriter) error {
	log.Printf("Trying to get log for %v %v\n", pod, container)
	req := kubectl.CoreV1().Pods("default").GetLogs(pod, &v1.PodLogOptions{
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
