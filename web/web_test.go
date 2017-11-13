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

	repoid := uuid.New()
	orgAndName := "testorg/" + repoid
	repo := &model.Repo{Id: orgAndName, Name: repoid}
	b := &model.Build{Repo: orgAndName, Number: 1, Status: "Running"}

	db.SaveRepo(repo)
	db.SaveBuild(b)

	if err != nil {
		t.Error(err)
	}
	hf := handleFetchBuilds(db)

	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/repo/"+orgAndName+"/builds", nil)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("org", "id")
	c.SetParamValues("testorg", repoid)

	err = hf(c)
	if err != nil {
		t.Fatal(err)
	}

	var res []model.Build
	bytes := rec.Body.Bytes()
	json.Unmarshal(bytes, &res)
	if res[0].Repo != orgAndName {
		t.Error("the id should be the same as the requested one")
	}
}

func TestHandleFetchRepoDataWithNoResultShouldWork(t *testing.T) {
	db, err := sql.New()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	repoid := uuid.New()
	orgAndName := "testorg/" + repoid

	repo := &model.Repo{Id: orgAndName, Name: repoid}

	err = db.SaveRepo(repo)
	if err != nil {
		t.Fatal(err)
	}

	hf := handleFetchRepoData(db)

	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/repo/"+orgAndName, nil)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	c.SetParamNames("org", "id")
	c.SetParamValues("testorg", repoid)
	err = hf(c)
	if err != nil {
		t.Fatal(err)
	}
	var res map[string]string
	json.Unmarshal(rec.Body.Bytes(), &res)
	assert.Equal(t, orgAndName, res["id"])
	assert.Equal(t, repoid, res["name"])
}

func TestGetRepos(t *testing.T) {
	db, err := sql.New()
	if err != nil {
		t.Error(err)
	}
	defer db.Close()

	repoid := uuid.New()
	repo := &model.Repo{Id: repoid, Name: "test" + repoid}

	err = db.SaveRepo(repo)
	if err != nil {
		t.Fatal(err)
	}

	repoid1 := uuid.New()
	repo1 := &model.Repo{Id: repoid1, Name: "test_1" + repoid1}

	err = db.SaveRepo(repo1)
	if err != nil {
		t.Fatal(err)
	}

	hf := handleFetchRepos(db)

	e := echo.New()
	req := httptest.NewRequest(echo.GET, "/repos", nil)

	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)
	err = hf(c)
	if err != nil {
		t.Fatal(err)
	}

	var res []model.Repo
	json.Unmarshal(rec.Body.Bytes(), &res)
	foundFirst := false
	foundSecond := false
	for _, v := range res {
		if v.Id == repoid {
			foundFirst = true
		}
		if v.Id == repoid1 {
			foundSecond = true
		}
	}
	if !foundFirst || !foundSecond {
		t.Error("Repos should have been in the repo list")
	}
}
