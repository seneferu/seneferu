package builder

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/sorenmat/pipeline/pipeline/frontend/yaml"

	"fmt"
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
	build := &model.Build{Number: 1, Owner: "sorenmat", Repo: "ci-server"}
	container := &yaml.Container{Environment: map[string]string{"Name": "sorenmat"}}
	cfg := &yaml.Config{}
	cfg.Pipeline.Containers = append(cfg.Pipeline.Containers, container)

	steps, err := createBuildSteps(build, cfg)
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

	container := &yaml.Container{Environment: map[string]string{"Name": "sorenmat"}, Coverage: str}
	cfg := &yaml.Config{}
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
