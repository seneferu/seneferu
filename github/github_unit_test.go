package github

import (
	"testing"
)

func TestURLExtractor(t *testing.T) {
	url, err := parseGithubTree([]byte(gittreedata))
	if err != nil {
		t.Error(err)
	}
	if url == "" {
		t.Error("url should not be empty")
	}
}

const gittreedata = `{
  "sha": "0c92b8088b3dec03be37c7c523b8d32ea92cf681",
  "url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/trees/0c92b8088b3dec03be37c7c523b8d32ea92cf681",
  "tree": [
    {
      "path": ".ci.yaml",
      "mode": "100644",
      "type": "blob",
      "sha": "4e5f17d166c69a57010c63cff97e8575eb40aaab",
      "size": 340,
      "url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/blobs/4e5f17d166c69a57010c63cff97e8575eb40aaab"
    },
    {
      "path": ".gitignore",
      "mode": "100644",
      "type": "blob",
      "sha": "485dee64bcfb48793379b200a1afd14e85a8aaf4",
      "size": 6,
      "url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/blobs/485dee64bcfb48793379b200a1afd14e85a8aaf4"
    },
    {
      "path": "README.md",
      "mode": "100644",
      "type": "blob",
      "sha": "f0f9a39f1cdf7d4e745fe0aeb0191c33a17f76d2",
      "size": 150,
      "url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/blobs/f0f9a39f1cdf7d4e745fe0aeb0191c33a17f76d2"
    },
    {
      "path": "build.sh",
      "mode": "100755",
      "type": "blob",
      "sha": "cea192594f076226ea1548bc346d2559bb2ed109",
      "size": 52,
      "url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/blobs/cea192594f076226ea1548bc346d2559bb2ed109"
    },
    {
      "path": "charts",
      "mode": "040000",
      "type": "tree",
      "sha": "1b94fd887ee4602e1d8b10f8075ff43c9c1754fd",
      "url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/trees/1b94fd887ee4602e1d8b10f8075ff43c9c1754fd"
    },
    {
      "path": "index.js",
      "mode": "100644",
      "type": "blob",
      "sha": "c40f9b0755a4c3d44b691a1dd5b192b1a3abc6b5",
      "size": 551,
      "url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/blobs/c40f9b0755a4c3d44b691a1dd5b192b1a3abc6b5"
    },
    {
      "path": "package-lock.json",
      "mode": "100644",
      "type": "blob",
      "sha": "84f7cbb4342bdd977cfa1a7eb4f8225716f83e38",
      "size": 74,
      "url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/blobs/84f7cbb4342bdd977cfa1a7eb4f8225716f83e38"
    },
    {
      "path": "package.json",
      "mode": "100644",
      "type": "blob",
      "sha": "c3856880a054ccb4fc131be8d2b8da24daceb854",
      "size": 912,
      "url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/blobs/c3856880a054ccb4fc131be8d2b8da24daceb854"
    },
    {
      "path": "public",
      "mode": "040000",
      "type": "tree",
      "sha": "e30651f0f260954659fe75214fc12c6309959202",
      "url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/trees/e30651f0f260954659fe75214fc12c6309959202"
    },
    {
      "path": "src",
      "mode": "040000",
      "type": "tree",
      "sha": "cc9461f0716f7c46c5195ab471f15f349a69023c",
      "url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/trees/cc9461f0716f7c46c5195ab471f15f349a69023c"
    },
    {
      "path": "webpack.config.js",
      "mode": "100644",
      "type": "blob",
      "sha": "746cf1feacefa13a31ea8bba224dadadcff55414",
      "size": 399,
      "url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/blobs/746cf1feacefa13a31ea8bba224dadadcff55414"
    },
    {
      "path": "yarn.lock",
      "mode": "100644",
      "type": "blob",
      "sha": "bd493748712340df79513fba6e0044b8a7272055",
      "size": 95213,
      "url": "https://api.github.com/repos/callebjorkell/cuddle-cake/git/blobs/bd493748712340df79513fba6e0044b8a7272055"
    }
  ],
  "truncated": false
}`
