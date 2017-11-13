package sql

import (
	"testing"

	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"gitlab.com/sorenmat/seneferu/model"
)

func TestSaveAndLoadRepo(t *testing.T) {
	service, err := New()
	defer service.Close()
	if err != nil {
		t.Error(err)
	}
	s := "test-" + uuid.New()
	repo := &model.Repo{Id: s, Name: "Apple Jack", Url: "www.applejack.io"}
	service.SaveRepo(repo)

	loadedRepo, err := service.LoadById(s)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, repo.Id, loadedRepo.Id)
	assert.Equal(t, "Apple Jack", loadedRepo.Name)

}

func TestSaveAndLoadAllBuilds(t *testing.T) {
	service, err := New()
	defer service.Close()
	if err != nil {
		t.Error(err)
	}
	repo := &model.Repo{Id: "test"}
	b := &model.Build{Repo: "test", Number: 1, Status: "Running"}

	service.SaveRepo(repo)
	service.SaveBuild(b)

	loadedRepo, err := service.LoadBuilds("test")
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
	repo := &model.Repo{Id: "test"}
	service.SaveRepo(repo)

	loadedRepo, err := service.LoadById("test")
	if err != nil {
		t.Error(err)
	}
	if loadedRepo.Id != "test" {
		t.Error("should have loaded the correct repo")
	}
	build := &model.Build{Repo: "test", Number: 2, Owner: "me"}
	err = service.SaveBuild(build)
	if err != nil {
		t.Error(err)
	}

	loadedBuild, err := service.LoadBuild("test", 2)
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
	repoName := "test" + uuid.New()
	repo := &model.Repo{Id: repoName}
	service.SaveRepo(repo)

	build := &model.Build{Repo: repoName, Number: 2, Owner: "me"}
	err = service.SaveBuild(build)
	assert.NoError(t, err)
	err = service.SaveBuild(build)
	assert.NoError(t, err)

	loadedBuild, err := service.LoadBuilds(repoName)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(loadedBuild))
}

func TestSaveAndLoadStep(t *testing.T) {
	service, err := New()
	defer service.Close()
	assert.NoError(t, err)

	repoID := "test" + uuid.New()
	repo := &model.Repo{Id: repoID}
	service.SaveRepo(repo)

	loadedRepo, err := service.LoadById(repoID)
	assert.NoError(t, err)

	assert.Equal(t, repoID, loadedRepo.Id)
	build := &model.Build{Repo: repoID, Number: 2, Owner: "me"}
	err = service.SaveBuild(build)
	assert.NoError(t, err)

	loadedBuild, err := service.LoadBuild(repoID, 2)
	assert.NoError(t, err)
	assert.Equal(t, 2, loadedBuild.Number)

	step := &model.Step{Repo: repoID, BuildNumber: 2, Name: "git"}
	err = service.SaveStep(step)
	assert.NoError(t, err)

	loadedStep, err := service.LoadStep(repoID, 2, "git")
	assert.NoError(t, err)
	assert.Equal(t, "git", loadedStep.Name)
}
