package sql

import (
	"testing"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"gitlab.com/sorenmat/seneferu/model"
)

func TestSaveAndLoadRepo(t *testing.T) {
	service, err := New()
	if err != nil {
		t.Error(err)
	}
	defer service.Close()
	org := "Seneferu"
	name := "coderepo"
	repo := &model.Repo{Org: org, Name: name, URL: "www.applejack.io"}
	err = service.SaveRepo(repo)
	assert.NoError(t, err)

	loadedRepo, err := service.LoadByOrgAndName(org, name)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, repo.Org, loadedRepo.Org)
	assert.Equal(t, name, loadedRepo.Name)

}

func TestSaveAndLoadAllBuilds(t *testing.T) {
	service, err := New()
	defer service.Close()
	if err != nil {
		t.Error(err)
	}
	org := "Seneferu"
	name := "coderepo-" + uuid.New()
	repo := &model.Repo{Org: org, Name: name, URL: "www.applejack.io"}

	b := &model.Build{Org: org, Name: name, Number: 1, Status: "Running"}

	err = service.SaveRepo(repo)
	assert.NoError(t, err)

	err = service.SaveBuild(b)
	assert.NoError(t, err)

	loadedRepo, err := service.LoadBuilds(org, name)
	if err != nil {
		t.Error(err)
	}
	if len(loadedRepo) <= 0 {
		t.Fatal("Expected to find one build, found ", len(loadedRepo))
	}
	if loadedRepo[0].Number != 1 {
		t.Error("should have loaded the correct repo")
	}
	if loadedRepo[0].Status != "Running" {
		t.Error("Expected status to be running")
	}
}

func TestSaveAndLoadBuild(t *testing.T) {
	service, err := New()
	defer service.Close()
	if err != nil {
		t.Error(err)
	}
	org := "Seneferu"
	name := "repo1"
	repo := &model.Repo{Org: org, Name: name}
	service.SaveRepo(repo)

	loadedRepo, err := service.LoadByOrgAndName(org, name)
	if err != nil {
		t.Error(err)
	}
	if loadedRepo.Org != org {
		t.Error("should have loaded the correct repo")
	}
	build := &model.Build{Org: org, Name: name, Number: 2}
	err = service.SaveBuild(build)
	if err != nil {
		t.Error(err)
	}

	loadedBuild, err := service.LoadBuild(org, name, 2)
	if err != nil {
		t.Error(err)
	}
	if loadedBuild.Number != 2 {
		t.Error("Build number should have been 2")
	}
}

func TestSaveBuildMultipleTimes(t *testing.T) {
	service, err := New()
	defer service.Close()
	if err != nil {
		t.Error(err)
	}
	org := "Seneferu"
	name := "repo-" + uuid.New()
	repo := &model.Repo{Org: org, Name: name}
	service.SaveRepo(repo)

	build := &model.Build{Org: org, Name: name, Number: 2}
	err = service.SaveBuild(build)
	assert.NoError(t, err)
	err = service.SaveBuild(build)
	assert.NoError(t, err)

	loadedBuild, err := service.LoadBuilds(org, name)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(loadedBuild))
}

func TestSaveAndLoadStep(t *testing.T) {
	service, err := New()
	defer service.Close()
	assert.NoError(t, err)

	org := "Seneferu"
	name := "repo-" + uuid.New()
	repo := &model.Repo{Org: org, Name: name}

	err = service.SaveRepo(repo)
	assert.NoError(t, err)

	loadedRepo, err := service.LoadByOrgAndName(org, name)
	assert.NoError(t, err)

	assert.Equal(t, org, loadedRepo.Org)
	assert.Equal(t, name, loadedRepo.Name)

	build := &model.Build{Org: org, Name: name, Number: 1}
	err = service.SaveBuild(build)
	assert.NoError(t, err)

	loadedBuild, err := service.LoadBuild(org, name, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, loadedBuild.Number)

	step := &model.Step{StepInfo: model.StepInfo{Org: org, Reponame: name, BuildNumber: 1, Name: "git"}}
	err = service.SaveStep(step)
	assert.NoError(t, err)

	loadedStep, err := service.LoadStep(org, name, 1, "git")
	assert.NoError(t, err)
	assert.Equal(t, 1, loadedStep.BuildNumber)
	assert.Equal(t, org, loadedStep.Org)
	assert.Equal(t, name, loadedStep.Reponame)
	assert.Equal(t, "git", loadedStep.Name)
}

func TestSaveAndLoadStepInfo(t *testing.T) {
	service, err := New()
	defer service.Close()
	assert.NoError(t, err)

	org := "Seneferu"
	name := "repo-" + uuid.New()
	repo := &model.Repo{Org: org, Name: name}
	service.SaveRepo(repo)

	loadedRepo, err := service.LoadByOrgAndName(org, name)
	assert.NoError(t, err)

	assert.Equal(t, org, loadedRepo.Org)
	build := &model.Build{Org: org, Name: name, Number: 1}
	err = service.SaveBuild(build)
	assert.NoError(t, err)

	loadedBuild, err := service.LoadBuild(org, name, 1)
	assert.NoError(t, err)
	assert.Equal(t, 1, loadedBuild.Number)

	step := &model.Step{StepInfo: model.StepInfo{Org: org, Reponame: name, BuildNumber: 2, Name: "git"}}
	err = service.SaveStep(step)
	assert.NoError(t, err)

	loadedStep, err := service.LoadStepInfo(org, name, "git", 2)
	assert.NoError(t, err)
	assert.Equal(t, "git", loadedStep.Name)
}
