package builder

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/sorenmat/pipeline/pipeline/frontend/yaml"
	"github.com/stretchr/testify/assert"
	"gitlab.com/sorenmat/seneferu/model"
)

func TestSomePath(t *testing.T) {
	commands := []string{"ls -la"}
	b64 := generateScript(commands)
	cmd, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Error(err)
	}
	if !strings.Contains(string(cmd), "ls -la") {
		t.Error("Seems like the command wasn't generated correctly: ", err)
	}
}

func TestDoneCmd(t *testing.T) {
	if doneCmd(0) == "build.done" {
		t.Error()
	}
	if doneCmd(1) == "build0.done" {
		t.Error()
	}

}

func TestCreateBuild(t *testing.T) {
	build := &model.Build{Number: 1, Name: "sorenmat", Org: "ci-server"}
	container := &Container{Environment: map[string]string{"Name": "sorenmat"}}
	cfg := &Config{}
	cfg.Pipeline.Containers = append(cfg.Pipeline.Containers, container)

	steps, err := createBuildSteps(build, cfg, "")
	if err != nil {
		t.Error("createBuildStep should not have failed")
	}
	if len(steps) != 1 {
		t.Error("expected 1 build")
	}
	found := false
	for _, value := range steps[0].Env {
		if value.Name == "Name" && value.Value == "sorenmat" {
			found = true
		}
	}
	if !found {
		t.Error("should have found environment variable in container")
	}
}

func TestCoverage(t *testing.T) {
	str := `coverage: (\d+?.?\d+\%)`

	container := &Container{Environment: map[string]string{"Name": "sorenmat"}, Coverage: str}
	cfg := &Config{}
	cfg.Pipeline.Containers = append(cfg.Pipeline.Containers, container)

	build := &model.Build{
		Number: 1,
		Steps:  []*model.Step{{Log: "Build worked\ncoverage: 40%"}},
	}
	coverageResult := getCoverageFromLogs(build, 1, str)
	if coverageResult == "" {
		t.Error("not able to get coverageResult ")
	}
	fmt.Print(coverageResult)
}

func TestDBLogWriter(t *testing.T) {
	step := &model.Step{}
	d := DBLogWriter{step: step}
	d.Write([]byte("Hello world"))
	d.Write([]byte("Hello world, from me"))
	if step.Log != "Hello worldHello world, from me" {
		t.Error(step)
	}
}

func TestStepSerialize(t *testing.T) {
	step := &model.Step{}
	step.Name = "test"
	step.Log = "Say "
	d := DBLogWriter{step: step}
	d.Write([]byte("Hello world"))
	assert.Equal(t, "Say Hello world", step.Log)
	b, err := json.Marshal(step)
	assert.NoError(t, err)
	assert.NotNil(t, b)
}

func TestCreateDockerContainer(t *testing.T) {
	c := createDockerContainer("some.host.com")
	assert.Equal(t, "/var/run", c.VolumeMounts[0].MountPath)
	assert.Equal(t, "/etc/docker/certs.d/some.host.com", c.VolumeMounts[1].MountPath)
	assert.True(t, *c.SecurityContext.Privileged)
}

func TestBranchMatch(t *testing.T) {
	c := yaml.Config{Pipeline: yaml.Containers{Containers: []*yaml.Container{
		&yaml.Container{
			Name: "MyStep",
			Constraints: yaml.Constraints{
				Branch: yaml.Constraint{
					Include: []string{"master"},
				},
			},
		},
	}}}
	m := c.Pipeline.Containers[0].Constraints.Branch.Match("master")
	assert.True(t, m)
	assert.False(t, c.Pipeline.Containers[0].Constraints.Branch.Match("master1"))
}

func TestBranchMatchExclude(t *testing.T) {
	c := yaml.Config{Pipeline: yaml.Containers{Containers: []*yaml.Container{
		&yaml.Container{
			Name: "MyStep",
			Constraints: yaml.Constraints{
				Branch: yaml.Constraint{
					Exclude: []string{"master"},
				},
			},
		},
	}}}
	m := c.Pipeline.Containers[0].Constraints.Branch.Match("master")
	assert.False(t, m)
	assert.True(t, c.Pipeline.Containers[0].Constraints.Branch.Match("master1"))
}

func TestBuildPipeline(t *testing.T) {
	c := Config{Pipeline: Containers{Containers: []*Container{
		&Container{
			Name: "MyStep",
			Constraints: yaml.Constraints{
				Branch: yaml.Constraint{
					Include: []string{"master"},
				},
			},
		}, &Container{
			Name: "My Production Step",
			Constraints: yaml.Constraints{
				Branch: yaml.Constraint{
					Include: []string{"production"},
				},
			},
		}, &Container{
			Name: "My Excluded Production Step",
			Constraints: yaml.Constraints{
				Branch: yaml.Constraint{
					Exclude: []string{"myproduction"},
				},
			},
		},
	}}}
	build := model.Build{Ref: "refs/heads/master"}
	containers, err := createBuildSteps(&build, &c, "")
	assert.NoError(t, err)
	assert.NotNil(t, containers)
	assert.Equal(t, 2, len(containers))
	assert.Equal(t, "MyStep", containers[0].Name)
	assert.Equal(t, "My Excluded Production Step", containers[1].Name)
}

func TestEmptyBuildPipeline(t *testing.T) {
	c := Config{Pipeline: Containers{Containers: []*Container{
		&Container{
			Name: "MyStep",
		},
	}}}
	build := model.Build{Ref: "refs/heads/master"}
	containers, err := createBuildSteps(&build, &c, "")
	assert.NoError(t, err)
	assert.NotNil(t, containers)
	assert.Equal(t, 1, len(containers))
	assert.Equal(t, "MyStep", containers[0].Name)
}

func TestYAMLFile(t *testing.T) {
	data, err := ioutil.ReadFile("ci.yaml")
	assert.NoError(t, err)
	c, err := yamlToConfig(data)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(c.Pipeline.Containers))
}

func TestYAMLFileWithBranch(t *testing.T) {
	data, err := ioutil.ReadFile("ci.yaml")
	assert.NoError(t, err)
	c, err := yamlToConfig(data)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(c.Pipeline.Containers))
	assert.Equal(t, 2, len(c.Pipeline.Containers[2].Constraints.Branch.Include))
}

func TestBranchesFromYAMLFile(t *testing.T) {
	data, err := ioutil.ReadFile("ci.yaml")
	assert.NoError(t, err)
	c, err := yamlToConfig(data)

	build := model.Build{Ref: "refs/heads/master"}
	containers, err := createBuildSteps(&build, c, "")
	assert.NoError(t, err)
	assert.NotNil(t, containers)
	assert.Equal(t, 3, len(containers))
	// branch not matching
	build = model.Build{Ref: "refs/heads/mysuperbranch"}
	containers, err = createBuildSteps(&build, c, "")
	assert.NoError(t, err)
	assert.NotNil(t, containers)
	assert.Equal(t, 2, len(containers))
	// mathcing tags
	build = model.Build{Ref: "refs/tags/v1.0"}
	containers, err = createBuildSteps(&build, c, "")
	assert.NoError(t, err)
	assert.NotNil(t, containers)
	assert.Equal(t, 3, len(containers))

}
