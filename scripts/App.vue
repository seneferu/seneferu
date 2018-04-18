<template>
    <div class="container-fluid" id="app">
        <div class="row">
            <div class="col-xs-3 sidebar"><div class="row">
                <div id="repolist" class="col-xs-10 sidebarlist">
                    <h2 class="listheader">REPOSITORIES</h2>
                    <ul class="nav">
                        <repo-item v-for="repo in repos" v-bind:repo="repo" v-bind:key="repo.id" v-on:select_repo="selectRepo"></repo-item>
                    </ul>
                </div>
                <div id="buildlist" class="col-xs-10 sidebarlist collapsed">
                    <h3 class="listheader">BUILDS</h3>
                    <ul class="nav">
                        <build-item v-for="build in builds" v-bind:build="build" v-bind:key="build.id" v-on:select_build="selectBuild"></build-item>
                    </ul>
                </div>
            </div></div>
            <div class="col-xs-offset-3 col-xs-9 main">
                <div class="row" id="build-info">
                    <div class="col-xs-12">
                        <build-info v-if="selectedBuild" v-bind:build="selectedBuild" v-on:select_step="showStep"></build-info>
                    </div>
                </div>
                <div class="row">
                    <div class="col-xs-12 output-container">
                        <console-output v-if="selectedBuild" v-bind:build-output="selectedBuild"></console-output>
                        <div v-if="!selectedBuild">
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
    name: 'App',
    data () {
        var self = this;
        this.$api.repos(wrapWrap((val) => self.repos = val ));

        return {
            repoSearch: '',
            repos: [{org: "Foo", name: "Baar"}],
            buildInfo: {},
            builds: [],
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
                this.setupLogStream(this.selectedStep);
            }
        },
        getBuildList: function(repo) {
            var self = this;
            this.$api.builds(wrapWrap(function(builds){
                builds.forEach((b) => b.selected = false);
                builds.sort((a,b) => b.number-a.number);
                self.builds = builds;
            }), repo.org, repo.name);
        },
        getBuild : function(build){
            var self = this;
            this.$api.steps(wrapWrap(function(steps){
                steps.forEach((s) => s.selected = false);
                self.selectedBuild.steps = steps;
                self.selectedStep = 0;
            }), self.selectedRepo.org, self.selectedRepo.name, build.number);
        },
        setupLogStream: function(step){
            console.log("Live logging - not implemented");
        },
        error: function(err){
            console.log(err);
            // Add some actual showing of error
        }
    }
}
</script>
