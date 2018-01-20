package github

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"

	"bytes"
	"encoding/json"

	"github.com/bmatsuo/go-jsontree"
	"github.com/pkg/errors"
)

const configfilename = ".ci.yaml"

// GetConfigFile tries to fetch the .ci.yaml file from the github repository
func GetConfigFile(org, repo, commit, token string) ([]byte, error) {
	j, err := fetchConfigFromGithub(org, repo, commit, token)
	if err != nil {
		return nil, errors.Wrap(err, "unable to binary data")
	}
	url, err := parseGithubTree(j)
	if err != nil {
		return nil, errors.Wrap(err, "unable fetch url")
	}
	client := getHTTPSClient()

	req, err := githubRequest("GET", url, token)
	resp, err := client.Do(req)

	if err != nil {
		return nil, errors.Wrap(err, "unable to read body")
	}

	j, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read body")
	}

	request := jsontree.New()
	err = request.UnmarshalJSON(j)
	k, err := request.Get("content").String()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get content")
	}

	o, err := base64.StdEncoding.DecodeString(k)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode content")
	}

	return o, nil
}

func parseGithubTree(j []byte) (string, error) {
	request := jsontree.New()
	err := request.UnmarshalJSON(j)
	if err != nil {
		return "", errors.Wrap(err, "unable to parse JSON")
	}
	t := request.Get("tree")
	l, err := t.Len()
	if err != nil {
		return "", errors.Wrap(err, "unable to get length")
	}
	for i := 0; i < l; i++ {
		path, err := t.GetIndex(i).Get("path").String()
		if err != nil {
			return "", errors.Wrap(err, "unable to get path")
		}

		if path == configfilename {
			url, err := t.GetIndex(i).Get("url").String()
			if err != nil {
				return "", errors.Wrap(err, "unable to get url")
			}
			return url, nil
		}
	}
	return "", fmt.Errorf("unable to find URL to .ci.yaml file in repository")
}

func fetchConfigFromGithub(org, repo, commit, token string) ([]byte, error) {
	client := getHTTPSClient()
	url := fmt.Sprintf("https://api.github.com/repos/%v/%v/git/trees/%v", org, repo, commit)
	fmt.Println("About to fetch: ", url)

	req, err := githubRequest("GET", url, token)

	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch file from github")
	}

	j, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "unable to read body from response")
	}

	return j, nil
}

func githubRequest(method, url, token string) (*http.Request, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create request to github")
	}
	req.SetBasicAuth("", token)
	return req, nil
}

func getHTTPSClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	return client
}

// ReportBack sends the build status back to Github
func ReportBack(state GithubStatus, owner, repo, sha, token string) error {
	body, err := json.Marshal(&state)
	if err != nil {
		return errors.Wrap(err, "unable to marshal status struct")
	}

	client := getHTTPSClient()
	url := fmt.Sprintf("https://api.github.com/repos/%v/%v/statuses/%v", owner, repo, sha)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(body))
	if err != nil {
		return errors.Wrap(err, "unable to create request to github")
	}
	req.SetBasicAuth("", token)
	_, err = client.Do(req)
	if err != nil {
		return errors.Wrap(err, "unable to post status to github")
	}
	return nil
}

// GithubStatus is the payload we use to update the build information
// on Github
type GithubStatus struct {
	State       string `json:"state"`
	TargetURL   string `json:"target_url"`
	Description string `json:"description"`
	Context     string `json:"context"`
}
