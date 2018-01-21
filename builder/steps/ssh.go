package steps

import "k8s.io/api/core/v1"

// CreateSSHAgentContainer creates a container with a ssh agent running
// this will be used for sharing git credentials in the build steps
func CreateSSHAgentContainer(shareddir string) v1.Container {
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

// CreateSSHKeyAdd will add the ssh key to the ssh agent
func CreateSSHKeyAdd(shareddir string) v1.Container {
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
