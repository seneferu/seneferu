<template>
    <div class="container-fluid" id="app">
        <div class="row">
            <div class="col-xs-3 sidebar"><div class="row">
                <div id="repolist" class="col-xs-10 sidebarlist">
                    <h2 class="listheader">REPOSITORIES</h2>
                    <ul class="nav">
                        <repo-item v-for="repo in repos" v-bind:repo="repo" v-bind:key="repo.id" v-on:reposelected="selectRepo"></repo-item>
                    </ul>
                </div>
                <div id="buildlist" class="col-xs-10 sidebarlist collapsed">
                    <h3 class="listheader">BUILDS</h3>
                    <ul class="nav">
                        <build-item v-for="build in builds" v-bind:build="build" v-bind:key="build.id" v-on:buildselected="selectBuild"></build-item>
                    </ul>
                </div>
            </div></div>
            <div class="col-xs-offset-3 col-xs-9 main">
                <div class="row" id="build-info">
                    <div class="col-xs-12">
                        <build-info v-if="selectedBuild" v-bind:build="selectedBuild" v-on:step="showStep"></build-info>
                    </div>
                </div>
                <div class="row">
                    <div class="col-xs-12 output-container">
                        <console-output v-if="selectedStep.build" v-bind:build-output="selectedStep.build"></console-output>
                        <div v-if="!selectedStep.build">
                            SELECT A BUILD STEP TO SEE WHAT HAPPENS IN THERE :-O
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>
</template>

<script>
import RepoItem from './repo-item.vue'
import BuildItem from './build-item.vue'
import api from './api.js'

function cbWrap(errFn, fn){
    return function(err, val){
        if(err){ return errFn(err); }
        return fn(val);
    }
}
var wrapWrap = cbWrap.bind(undefined, function(err){ app.error(err); });

export default {
    name: 'app',
    data () {
        return {
            repoSearch: '',
            repos: [],
            buildInfo: {},
            builds: {},
            selectedRepo: [],
            selectedBuild: undefined,
            selectedStep: {build: undefined}
        }
    },
    components: {
        RepoItem,
        BuildItem
    },
    methods: {
        selectRepo: function(repo){
            this.selectedRepo = repo;
            toggleSidebars();
            this.getBuildList(this.selectedRepo);
        },
        selectBuild: function(build){
            this.selectedBuild = build;
            this.selectedStep = {};
            this.getBuild(this.selectedBuild)
        },
        showStep: function(step){
            this.selectedStep = step;
            if(step.status === "Running"){
                this.setupWebSocket(this.selectedStep);
            }
        },
        getBuildList: function(repo) {
            api.builds(wrapWrap(function(builds){
                builds.forEach((b) => b.selected = false);
                app.builds = builds
            }), repo.id);
        },
        getBuild : function(build){
            api.build(wrapWrap(function(build){
                build.steps.forEach((s) => s.selected = false);
                app.selectedBuild = build;

            }), this.selectedRepo.id, build.id)
            //buildStorage.fetch(this, this.selectedRepo, this.selectedBuild.number);
            // Do something setup
        },
        setupWebSocket: function(step){
            var socket =  new WebSocket("ws://"+window.location.host+"/ws");
            socket.onopen = function () {
                console.log('Connected')
            };

            socket.onmessage = function (evt) {
                var data = JSON.parse(evt.data);
                if(data["Step"] === step.name) {
                    step.build = step.build + data["Line"]
                }
                console.log(evt);
            }
        },
        error: function(err){
            console.log(err);
            // Add some actual showing of error
        }
    }
}
</script>
