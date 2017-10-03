package github

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"github.com/bmatsuo/go-jsontree"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/http"
)

const configfilename = ".ci.yaml"

// GetConfigFile tries to fetch the .ci.yaml file from the github repository
func GetConfigFile(owner, repo, commit string) ([]byte, error) {
	j, err := fetchConfigFromGithub(owner, repo, commit)
	if err != nil {
		return nil, errors.Wrap(err, "unable to binary data")
	}
	url, err := parseGithubTree(j)
	if err != nil {
		return nil, errors.Wrap(err, "unable fetch url")
	}
	client := getHTTPSClient()
	resp, err := client.Get(url)
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

func fetchConfigFromGithub(owner, repo, commit string) ([]byte, error) {
	client := getHTTPSClient()
	url := fmt.Sprintf("https://api.github.com/repos/%v/%v/git/trees/%v", owner, repo, commit)
	fmt.Println("About to fetch: ", url)
	resp, err := client.Get(url)
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
