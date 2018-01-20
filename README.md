Simple Kubernetes based build

[![Go Report Card](https://goreportcard.com/badge/gitlab.com/sorenmat/seneferu)](https://goreportcard.com/report/gitlab.com/sorenmat/seneferu)

Named after the pharo that build 3 pyramids, because he was surely awesome... like this tools

Was initial conceived in the Tradeshift Global Hackathon in 2017.

__DO NOT USE IN PRODUCTION__ or well that's up to you I guess.


Build system that schedules build in a pod in a Kubernetes cluster.
That way we can leverage Kubernetes for scaling and reliability.


# Screenshot

![Seneferu main screen](docs/seneferu.png "Main screen")


# Building

`go build`

# Requirements

A Postgres database up and running.

A Secret in Kubernetes called sshkey that contains a SSH key for accessing private repositories


# Running

Start Seneferu, and configure a Github webhook to point to http://your-server.com/webhook

When a push event is triggered on Github, Seneferu will then receive the payload and start a build.
The build will be executed in the same Kubernetes cluster as the build server is running in.


```shell
usage: seneferu --githubsecret=GITHUBSECRET --githubToken=GITHUBTOKEN --sshkey=SSHKEY [<flags>]

Flags:
  --help                       Show context-sensitive help (also try --help-long and --help-man).
  --kubeconfig=KUBECONFIG      Kubernetes Config File
  --githubsecret=GITHUBSECRET  Github secret token, needs to match the one on Github
  --helmhost=HELMHOST          Hostname and port of the Helm host / tiller
  --githubToken=GITHUBTOKEN    Github access token, to access the API
  --sshkey=SSHKEY              Github ssh key, used for cloning the repositories
```

Build repositories that contains a .ci.yaml file

.ci.yaml example

```yaml
pipeline:
  build:
    group: build
    image: golang:latest
    commands:
      - go build
      - go test
      - docker build -t sorenmat/test:${HASH} .
```

# Building and running tests

`go build` create a server binary that can be executed from the commandline



# FAQ

1. How do i add environment variables to my build step

```yaml
pipeline:
  build:
    group: build
    image: golang:latest
    environment:
        - NAME=testing
    commands:
      - echo ${NAME}

```

2. How do I get test coverage for my build

```yaml
pipeline:
  build:
    group: build
    image: golang:latest
    coverage: coverage: (\d+?.?\d+\%)
    environment:
        - NAME=testing
    commands:
      - echo ${NAME}

```

3. Known environment variables
   `GIT_REF` ref to the git hash being build, the head of the branch


# Contributers

Soren Mathiasen @sorenmat

Carl-Magnus Björkell @callebjorkell
 
Christian Nilsson @nchrisdk
 
Nicolai Willems @nwillems
 

