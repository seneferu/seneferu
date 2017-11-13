// +build integration

package github

import (
	"bytes"
	"crypto/hmac"
	sha12 "crypto/sha1"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httputil"
	"testing"
)

func TestPayload(t *testing.T) {
	req, err := http.NewRequest("POST", "http://localhost:8080/webhook?access_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0ZXh0IjoiVHJhZGVzaGlmdC90cnVlYm4tZ2FiYnkiLCJ0eXBlIjoiaG9vayJ9.ev_sUvp7lE4NtXFIMTkR7qf7lw0zK9IH3ZdBNtSvJTI", bytes.NewBufferString(payload))
	if err != nil {
		t.Error(err)
	}
	req.Header.Set("User-Agent", "GitHub-Hookshot/aff25de")

	req.Header.Set("X-GitHub-Delivery", "b136ff40-9167-11e7-9bbf-502dfa80d048")
	req.Header.Set("X-GitHub-Event", "push")

	hashing := hmac.New(sha12.New, []byte("supersecretpassword..nice"))
	hashing.Write([]byte(payload))
	req.Header.Set("X-Hub-Signature", "sha1="+hex.EncodeToString(hashing.Sum(nil)))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	b, err := httputil.DumpResponse(resp, true)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(string(b))

}

const payload = `{
  "ref": "refs/heads/master",
  "before": "e04fcb940854d123d03974767bd7b266883c9c0c",
  "after": "23ec65cd1aeb93b872c7ceee6e1c02c9a277040f",
  "created": false,
  "deleted": false,
  "forced": false,
  "base_ref": null,
  "compare": "https://github.com/callebjorkell/cuddle-cake/compare/e04fcb940854...23ec65cd1aeb",
  "commits": [
    {
      "id": "4f601d23ab67dc871270ce907aee989723c71f19",
      "tree_id": "a537a55462b779befd0dd32a67b8fd115e3a252f",
      "distinct": true,
      "message": "Shit that didn't get added?",
      "timestamp": "2017-09-28T18:00:51+02:00",
      "url": "https://github.com/callebjorkell/cuddle-cake/commit/4f601d23ab67dc871270ce907aee989723c71f19",
      "author": {
        "name": "Carl-Magnus Björkell",
        "email": "cmb@tradeshift.com",
        "username": "callebjorkell"
      },
      "committer": {
        "name": "Carl-Magnus Björkell",
        "email": "cmb@tradeshift.com",
        "username": "callebjorkell"
      },
      "added": [
        "Dockerfile",
        "charts/cuddle-cake/Chart.yaml",
        "charts/cuddle-cake/templates/_helpers.tpl",
        "charts/cuddle-cake/templates/configmap.yaml",
        "charts/cuddle-cake/templates/deployment.yaml",
        "charts/cuddle-cake/values.yaml"
      ],
      "removed": [

      ],
      "modified": [

      ]
    },
    {
      "id": "23ec65cd1aeb93b872c7ceee6e1c02c9a277040f",
      "tree_id": "bf71cd6d28d893922e189628b84d4bc032dce7b0",
      "distinct": true,
      "message": "Merge branch 'master' of github.com:callebjorkell/cuddle-cake",
      "timestamp": "2017-09-28T18:01:03+02:00",
      "url": "https://github.com/callebjorkell/cuddle-cake/commit/23ec65cd1aeb93b872c7ceee6e1c02c9a277040f",
      "author": {
        "name": "Carl-Magnus Björkell",
        "email": "cmb@tradeshift.com",
        "username": "callebjorkell"
      },
      "committer": {
        "name": "Carl-Magnus Björkell",
        "email": "cmb@tradeshift.com",
        "username": "callebjorkell"
      },
      "added": [

      ],
      "removed": [

      ],
      "modified": [
        ".ci.yaml"
      ]
    }
  ],
  "head_commit": {
    "id": "23ec65cd1aeb93b872c7ceee6e1c02c9a277040f",
    "tree_id": "bf71cd6d28d893922e189628b84d4bc032dce7b0",
    "distinct": true,
    "message": "Merge branch 'master' of github.com:callebjorkell/cuddle-cake",
    "timestamp": "2017-09-28T18:01:03+02:00",
    "url": "https://github.com/callebjorkell/cuddle-cake/commit/23ec65cd1aeb93b872c7ceee6e1c02c9a277040f",
    "author": {
      "name": "Carl-Magnus Björkell",
      "email": "cmb@tradeshift.com",
      "username": "callebjorkell"
    },
    "committer": {
      "name": "Carl-Magnus Björkell",
      "email": "cmb@tradeshift.com",
      "username": "callebjorkell"
    },
    "added": [

    ],
    "removed": [

    ],
    "modified": [
      ".ci.yaml"
    ]
  },
  "repository": {
    "id": 105125490,
    "name": "cuddle-cake",
    "full_name": "callebjorkell/cuddle-cake",
    "owner": {
      "name": "callebjorkell",
      "email": "cmb@tradeshift.com",
      "login": "callebjorkell",
      "id": 4091674,
      "avatar_url": "https://avatars2.githubusercontent.com/u/4091674?v=4",
      "gravatar_id": "",
      "url": "https://api.github.com/users/callebjorkell",
      "html_url": "https://github.com/callebjorkell",
      "followers_url": "https://api.github.com/users/callebjorkell/followers",
      "following_url": "https://api.github.com/users/callebjorkell/following{/other_user}",
      "gists_url": "https://api.github.com/users/callebjorkell/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/callebjorkell/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/callebjorkell/subscriptions",
      "organizations_url": "https://api.github.com/users/callebjorkell/orgs",
      "repos_url": "https://api.github.com/users/callebjorkell/repos",
      "events_url": "https://api.github.com/users/callebjorkell/events{/privacy}",
      "received_events_url": "https://api.github.com/users/callebjorkell/received_events",
      "type": "User",
      "site_admin": false
    },
    "private": false,
    "html_url": "https://github.com/callebjorkell/cuddle-cake",
    "description": "You are a unicorn born in Canterlot. You are the prettiest pony this side of Horseshoe Bay! Your best friend is a lynx named Sunflower.",
    "fork": false,
    "url": "https://github.com/callebjorkell/cuddle-cake",
    "forks_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/forks",
    "keys_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/keys{/key_id}",
    "collaborators_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/collaborators{/collaborator}",
    "teams_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/teams",
    "hooks_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/hooks",
    "issue_events_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/issues/events{/number}",
    "events_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/events",
    "assignees_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/assignees{/user}",
    "branches_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/branches{/branch}",
    "tags_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/tags",
    "blobs_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/blobs{/sha}",
    "git_tags_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/tags{/sha}",
    "git_refs_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/refs{/sha}",
    "trees_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/trees{/sha}",
    "statuses_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/statuses/{sha}",
    "languages_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/languages",
    "stargazers_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/stargazers",
    "contributors_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/contributors",
    "subscribers_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/subscribers",
    "subscription_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/subscription",
    "commits_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/commits{/sha}",
    "git_commits_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/commits{/sha}",
    "comments_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/comments{/number}",
    "issue_comment_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/issues/comments{/number}",
    "contents_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/contents/{+path}",
    "compare_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/compare/{base}...{head}",
    "merges_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/merges",
    "archive_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/{archive_format}{/ref}",
    "downloads_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/downloads",
    "issues_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/issues{/number}",
    "pulls_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/pulls{/number}",
    "milestones_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/milestones{/number}",
    "notifications_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/notifications{?since,all,participating}",
    "labels_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/labels{/name}",
    "releases_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/releases{/id}",
    "deployments_url": "https://api.github.com/repos/callebjorkell/cuddle-cake/deployments",
    "created_at": 1506588880,
    "updated_at": "2017-09-28T14:06:18Z",
    "pushed_at": 1506614479,
    "git_url": "git://github.com/callebjorkell/cuddle-cake.git",
    "ssh_url": "git@github.com:callebjorkell/cuddle-cake.git",
    "clone_url": "https://github.com/callebjorkell/cuddle-cake.git",
    "svn_url": "https://github.com/callebjorkell/cuddle-cake",
    "homepage": null,
    "size": 30,
    "stargazers_count": 0,
    "watchers_count": 0,
    "language": "JavaScript",
    "has_issues": true,
    "has_projects": true,
    "has_downloads": true,
    "has_wiki": true,
    "has_pages": false,
    "forks_count": 1,
    "mirror_url": null,
    "open_issues_count": 0,
    "forks": 1,
    "open_issues": 0,
    "watchers": 0,
    "default_branch": "master",
    "stargazers": 0,
    "master_branch": "master"
  },
  "pusher": {
    "name": "callebjorkell",
    "email": "cmb@tradeshift.com"
  },
  "sender": {
    "login": "callebjorkell",
    "id": 4091674,
    "avatar_url": "https://avatars2.githubusercontent.com/u/4091674?v=4",
    "gravatar_id": "",
    "url": "https://api.github.com/users/callebjorkell",
    "html_url": "https://github.com/callebjorkell",
    "followers_url": "https://api.github.com/users/callebjorkell/followers",
    "following_url": "https://api.github.com/users/callebjorkell/following{/other_user}",
    "gists_url": "https://api.github.com/users/callebjorkell/gists{/gist_id}",
    "starred_url": "https://api.github.com/users/callebjorkell/starred{/owner}{/repo}",
    "subscriptions_url": "https://api.github.com/users/callebjorkell/subscriptions",
    "organizations_url": "https://api.github.com/users/callebjorkell/orgs",
    "repos_url": "https://api.github.com/users/callebjorkell/repos",
    "events_url": "https://api.github.com/users/callebjorkell/events{/privacy}",
    "received_events_url": "https://api.github.com/users/callebjorkell/received_events",
    "type": "User",
    "site_admin": false
  }
}`

const payload2 = `{
  "ref": "refs/heads/master",
  "before": "13d4e4dcf18e5915ff77ef2a59d2322f0aa6c9ff",
  "after": "0fd9ce62026bf40f64235f9cd766eae01736f713",
  "created": false,
  "deleted": false,
  "forced": false,
  "base_ref": null,
  "compare": "https://github.com/sorenmat/buildtest/compare/13d4e4dcf18e...0fd9ce62026b",
  "commits": [
    {
      "id": "0fd9ce62026bf40f64235f9cd766eae01736f713",
      "tree_id": "954c729f83dc8845a002347714f71aed46e32c9e",
      "distinct": true,
      "message": "made test more reliable",
      "timestamp": "2017-10-05T09:38:06+02:00",
      "url": "https://github.com/sorenmat/buildtest/commit/0fd9ce62026bf40f64235f9cd766eae01736f713",
      "author": {
        "name": "Soren Mathiasen",
        "email": "smo@tradeshift.com",
        "username": "sorenmat"
      },
      "committer": {
        "name": "Soren Mathiasen",
        "email": "smo@tradeshift.com",
        "username": "sorenmat"
      },
      "added": [

      ],
      "removed": [

      ],
      "modified": [
        "main_test.go"
      ]
    }
  ],
  "head_commit": {
    "id": "0fd9ce62026bf40f64235f9cd766eae01736f713",
    "tree_id": "954c729f83dc8845a002347714f71aed46e32c9e",
    "distinct": true,
    "message": "made test more reliable",
    "timestamp": "2017-10-05T09:38:06+02:00",
    "url": "https://github.com/sorenmat/buildtest/commit/0fd9ce62026bf40f64235f9cd766eae01736f713",
    "author": {
      "name": "Soren Mathiasen",
      "email": "smo@tradeshift.com",
      "username": "sorenmat"
    },
    "committer": {
      "name": "Soren Mathiasen",
      "email": "smo@tradeshift.com",
      "username": "sorenmat"
    },
    "added": [

    ],
    "removed": [

    ],
    "modified": [
      "main_test.go"
    ]
  },
  "repository": {
    "id": 105780377,
    "name": "buildtest",
    "full_name": "sorenmat/buildtest",
    "owner": {
      "name": "sorenmat",
      "email": "smo@tradeshift.com",
      "login": "sorenmat",
      "id": 335103,
      "avatar_url": "https://avatars1.githubusercontent.com/u/335103?v=4",
      "gravatar_id": "",
      "url": "https://api.github.com/users/sorenmat",
      "html_url": "https://github.com/sorenmat",
      "followers_url": "https://api.github.com/users/sorenmat/followers",
      "following_url": "https://api.github.com/users/sorenmat/following{/other_user}",
      "gists_url": "https://api.github.com/users/sorenmat/gists{/gist_id}",
      "starred_url": "https://api.github.com/users/sorenmat/starred{/owner}{/repo}",
      "subscriptions_url": "https://api.github.com/users/sorenmat/subscriptions",
      "organizations_url": "https://api.github.com/users/sorenmat/orgs",
      "repos_url": "https://api.github.com/users/sorenmat/repos",
      "events_url": "https://api.github.com/users/sorenmat/events{/privacy}",
      "received_events_url": "https://api.github.com/users/sorenmat/received_events",
      "type": "User",
      "site_admin": false
    },
    "private": false,
    "html_url": "https://github.com/sorenmat/buildtest",
    "description": null,
    "fork": false,
    "url": "https://github.com/sorenmat/buildtest",
    "forks_url": "https://api.github.com/repos/sorenmat/buildtest/forks",
    "keys_url": "https://api.github.com/repos/sorenmat/buildtest/keys{/key_id}",
    "collaborators_url": "https://api.github.com/repos/sorenmat/buildtest/collaborators{/collaborator}",
    "teams_url": "https://api.github.com/repos/sorenmat/buildtest/teams",
    "hooks_url": "https://api.github.com/repos/sorenmat/buildtest/hooks",
    "issue_events_url": "https://api.github.com/repos/sorenmat/buildtest/issues/events{/number}",
    "events_url": "https://api.github.com/repos/sorenmat/buildtest/events",
    "assignees_url": "https://api.github.com/repos/sorenmat/buildtest/assignees{/user}",
    "branches_url": "https://api.github.com/repos/sorenmat/buildtest/branches{/branch}",
    "tags_url": "https://api.github.com/repos/sorenmat/buildtest/tags",
    "blobs_url": "https://api.github.com/repos/sorenmat/buildtest/git/blobs{/sha}",
    "git_tags_url": "https://api.github.com/repos/sorenmat/buildtest/git/tags{/sha}",
    "git_refs_url": "https://api.github.com/repos/sorenmat/buildtest/git/refs{/sha}",
    "trees_url": "https://api.github.com/repos/sorenmat/buildtest/git/trees{/sha}",
    "statuses_url": "https://api.github.com/repos/sorenmat/buildtest/statuses/{sha}",
    "languages_url": "https://api.github.com/repos/sorenmat/buildtest/languages",
    "stargazers_url": "https://api.github.com/repos/sorenmat/buildtest/stargazers",
    "contributors_url": "https://api.github.com/repos/sorenmat/buildtest/contributors",
    "subscribers_url": "https://api.github.com/repos/sorenmat/buildtest/subscribers",
    "subscription_url": "https://api.github.com/repos/sorenmat/buildtest/subscription",
    "commits_url": "https://api.github.com/repos/sorenmat/buildtest/commits{/sha}",
    "git_commits_url": "https://api.github.com/repos/sorenmat/buildtest/git/commits{/sha}",
    "comments_url": "https://api.github.com/repos/sorenmat/buildtest/comments{/number}",
    "issue_comment_url": "https://api.github.com/repos/sorenmat/buildtest/issues/comments{/number}",
    "contents_url": "https://api.github.com/repos/sorenmat/buildtest/contents/{+path}",
    "compare_url": "https://api.github.com/repos/sorenmat/buildtest/compare/{base}...{head}",
    "merges_url": "https://api.github.com/repos/sorenmat/buildtest/merges",
    "archive_url": "https://api.github.com/repos/sorenmat/buildtest/{archive_format}{/ref}",
    "downloads_url": "https://api.github.com/repos/sorenmat/buildtest/downloads",
    "issues_url": "https://api.github.com/repos/sorenmat/buildtest/issues{/number}",
    "pulls_url": "https://api.github.com/repos/sorenmat/buildtest/pulls{/number}",
    "milestones_url": "https://api.github.com/repos/sorenmat/buildtest/milestones{/number}",
    "notifications_url": "https://api.github.com/repos/sorenmat/buildtest/notifications{?since,all,participating}",
    "labels_url": "https://api.github.com/repos/sorenmat/buildtest/labels{/name}",
    "releases_url": "https://api.github.com/repos/sorenmat/buildtest/releases{/id}",
    "deployments_url": "https://api.github.com/repos/sorenmat/buildtest/deployments",
    "created_at": 1507128378,
    "updated_at": "2017-10-04T14:54:36Z",
    "pushed_at": 1507189093,
    "git_url": "git://github.com/sorenmat/buildtest.git",
    "ssh_url": "git@github.com:sorenmat/buildtest.git",
    "clone_url": "https://github.com/sorenmat/buildtest.git",
    "svn_url": "https://github.com/sorenmat/buildtest",
    "homepage": null,
    "size": 1,
    "stargazers_count": 0,
    "watchers_count": 0,
    "language": "Go",
    "has_issues": true,
    "has_projects": true,
    "has_downloads": true,
    "has_wiki": true,
    "has_pages": false,
    "forks_count": 0,
    "mirror_url": null,
    "open_issues_count": 0,
    "forks": 0,
    "open_issues": 0,
    "watchers": 0,
    "default_branch": "master",
    "stargazers": 0,
    "master_branch": "master"
  },
  "pusher": {
    "name": "sorenmat",
    "email": "smo@tradeshift.com"
  },
  "sender": {
    "login": "sorenmat",
    "id": 335103,
    "avatar_url": "https://avatars1.githubusercontent.com/u/335103?v=4",
    "gravatar_id": "",
    "url": "https://api.github.com/users/sorenmat",
    "html_url": "https://github.com/sorenmat",
    "followers_url": "https://api.github.com/users/sorenmat/followers",
    "following_url": "https://api.github.com/users/sorenmat/following{/other_user}",
    "gists_url": "https://api.github.com/users/sorenmat/gists{/gist_id}",
    "starred_url": "https://api.github.com/users/sorenmat/starred{/owner}{/repo}",
    "subscriptions_url": "https://api.github.com/users/sorenmat/subscriptions",
    "organizations_url": "https://api.github.com/users/sorenmat/orgs",
    "repos_url": "https://api.github.com/users/sorenmat/repos",
    "events_url": "https://api.github.com/users/sorenmat/events{/privacy}",
    "received_events_url": "https://api.github.com/users/sorenmat/received_events",
    "type": "User",
    "site_admin": false
  }
}`
