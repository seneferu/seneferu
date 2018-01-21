package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/labstack/echo"
	"github.com/matryer/silk/runner"
	"gitlab.com/sorenmat/seneferu/model"
	"gitlab.com/sorenmat/seneferu/storage/memory"
)

func TestStatusSpec(t *testing.T) {
	// start a server
	//ts := httptest.NewServer(http.HandlerFunc(handleStatus()))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e := echo.New()
		c := e.NewContext(r, w)
		handleStatus()(c)
	}))
	defer ts.Close()
	runner.New(t, ts.URL).RunGlob(filepath.Glob("status.silk.md"))
}
func TestRepoSpec(t *testing.T) {
	// start a server
	//ts := httptest.NewServer(http.HandlerFunc(handleStatus()))
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		e := echo.New()
		c := e.NewContext(r, w)
		storage := memory.New()
		storage.SaveRepo(&model.Repo{Name: "TestRepo", Org: "someorg", URL: "https://github.com/blabla/blabla"})
		handleFetchRepos(storage)(c)
	}))
	defer ts.Close()
	runner.New(t, ts.URL).RunGlob(filepath.Glob("repos.silk.md"))
}
