package web

import (
	"github.com/labstack/echo"
	"github.com/pborman/uuid"
	"github.com/stretchr/testify/assert"
	"gitlab.com/sorenmat/seneferu/model"
	"gitlab.com/sorenmat/seneferu/storage/sql"
	"k8s.io/apimachinery/pkg/util/json"
	"net/http/httptest"
	"testing"
)

func TestHandleFetchBuildsWithEmptyIdShouldFail(t *testing.T) {
	db, err := sql.New()
	defer db.Close()
	if err != nil {
		t.Error(err)
	}
	hf := handleFetchBuilds(db)

	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/", nil)
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err = hf(c)
	if err == nil {
		t.Error("should have failed")
	}

}

func TestHandleFetchBuildsWithNoResultShouldReturnAnError(t *testing.T) {
	db, err := sql.New()
	defer db.Close()

	if err != nil {
		t.Error(err)
	}
	hf := handleFetchBuilds(db)

	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/repo/testorg/1123/builds", nil)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("org", "id")
	c.SetParamValues("testorg", "1123")
	err = hf(c)
	if err != nil && err.Error() != "unable to load bucket 1123: not found" {
		t.Error(err)
	}

}
func TestHandleFetchBuildsWithNoResultShouldWork(t *testing.T) {
	db, err := sql.New()

	defer db.Close()

	org := "testorg"
	name := "repoid"
	repo := &model.Repo{Org: org, Name: name}
	b := &model.Build{Org: org, Name: name, Number: 1, Status: "Running"}

	err = db.SaveRepo(repo)
	assert.NoError(t, err)

	err = db.SaveBuild(b)
	assert.NoError(t, err)

	if err != nil {
		t.Error(err)
	}
	hf := handleFetchBuilds(db)

	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/repo/"+org+"/"+name+"/builds", nil)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("org", "id")
	c.SetParamValues(org, name)

	err = hf(c)
	if err != nil {
		t.Fatal(err)
	}

	var res []model.Build
	bytes := rec.Body.Bytes()
	err = json.Unmarshal(bytes, &res)
	assert.NoError(t, err)
	assert.Equal(t, org, res[0].Org)
	assert.Equal(t, name, res[0].Name)
}

func TestHandleFetchRepoDataWithNoResultShouldWork(t *testing.T) {
	db, err := sql.New()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	name := uuid.New()
	org := "testorg"

	repo := &model.Repo{Org: org, Name: name}

	err = db.SaveRepo(repo)
	assert.NoError(t, err)

	hf := handleFetchRepoData(db)

	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/repo/"+org+"/"+name, nil)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("org", "id")
	c.SetParamValues(org, name)
	err = hf(c)
	assert.NoError(t, err)

	var res map[string]string
	err = json.Unmarshal(rec.Body.Bytes(), &res)
	assert.NoError(t, err)

	assert.Equal(t, org, res["org"])
	assert.Equal(t, name, res["name"])
}

func TestGetRepos(t *testing.T) {
	db, err := sql.New()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	org := "seneferu-" + uuid.New()
	firstName := uuid.New()
	repo := &model.Repo{Org: org, Name: firstName}

	err = db.SaveRepo(repo)
	assert.NoError(t, err)

	repoid1 := uuid.New()
	secoundName := "test_1" + repoid1
	repo1 := &model.Repo{Org: org, Name: secoundName}

	err = db.SaveRepo(repo1)
	assert.NoError(t, err)

	hf := handleFetchRepos(db)

	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/repos", nil)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err = hf(c)
	assert.NoError(t, err)

	var res []model.Repo
	err = json.Unmarshal(rec.Body.Bytes(), &res)
	assert.NoError(t, err)

	foundFirst := false
	foundSecond := false
	for _, v := range res {
		if v.Name == firstName && v.Org == org {
			foundFirst = true
		}
		if v.Name == secoundName && v.Org == org {
			foundSecond = true
		}
	}
	if !foundFirst || !foundSecond {
		t.Error("Repos should have been in the repo list", foundFirst, foundSecond)
	}
}
