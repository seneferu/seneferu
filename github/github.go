package github

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"

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

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create request to github")
	}
	req.SetBasicAuth("", token)
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

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create request to github")
	}
	req.SetBasicAuth("", token)
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

func getHTTPSClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	return client
}
