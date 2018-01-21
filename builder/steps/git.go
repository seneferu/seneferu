package steps

import (
	"fmt"

	"strings"

	"gitlab.com/sorenmat/seneferu/model"
	"k8s.io/api/core/v1"
)

// CreateGitContainer this create the git container used for cloning the sourcecode.
// this is a mandatory build step in Seneferu
func CreateGitContainer(shareddir string, build *model.Build) v1.Container {

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
	cmds := []string{waitForContainerCmd("ssh-key-add", shareddir), "mkdir ~/.ssh", sshTrustCmd, cloneCmd, curWDCmd, checkoutCmd, doneCmd}
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
