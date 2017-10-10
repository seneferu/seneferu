
Vue.component('repo-list',{
    props: ['repos'],
    template: '<ul class="nav">' +
            '<repo-item v-for="repo in repos" v-bind:repo="repo" v-bind:key="repo.id" v-on:reposelected="selectRepo"></repo-item>' +
            '</ul>',
    methods: {
        selectRepo: function(repo){
            this.$emit('reposelected', repo)
        }
    }
});

Vue.component('repo-item', {
    props: ['repo'],
    template: "<li><div><a v-on:click='selectRepo'>{{ repo.name }}</a></div></li>",
    methods: {
        selectRepo: function(){
            this.$emit('reposelected', this.repo)
        }
    }

});

Vue.component('build-list', {
    props: ['builds'],
    template: '<ul class="nav">' +
        '<build-item v-for="build in builds" v-bind:build="build" v-bind:key="build.id" v-on:buildselected="selectBuild"></build-item>' +
        '</ul>',
    methods: {
        selectBuild: function(build){
            this.builds.forEach((b) => b.selected = false);
            build.selected = true;
            this.$emit('buildselected', build);
        }
    }
});

Vue.component('build-item', {
    props: ['build'],
    template: "<li class='builditem'>" +
    '<a v-on:click="selectBuild" :class="[status_class(build), { active: isSelected }]">' +
        '<span v-bind:class="[\'glyphicon\', status_icon(build) ]"></span>' +
        "<span>{{ time(build.timestamp) }}</span>" +
    "</a></li>",
    computed: {
        isSelected: function(){ return this.build.selected; }
    },
    methods: {
        selectBuild: function(){
            this.$emit('buildselected', this.build);
        },
        time: function(timestamp){
            return timestamp.substring(0, 16).replace('T', ' ');
        },
        status_icon: function(build){
            if(build.status.toLowerCase() === "done"){
                return 'glyphicon-' + (build.success ? 'ok' : 'remove');
            }
            var statuses = {
                "started": "hourglass",
                "running": "hourglass",
                "created": "asterisk"
            };

            return 'glyphicon-'+statuses[build.status.toLowerCase()];
        },
        status_class: function(build){
            if(build.status.toLowerCase() === "done"){
                return build.success ? 'text-success' : 'text-danger';
            }else{
                return 'text-muted';
            }
        }
    }
});

Vue.component('build-info', {
    props: ['build'],
    template: '<div>' +
    '<div class="row"><div class="col-xs-12"> ' +
    'Build Information: <br/>' +
    '<pre>Build start: {{build.timestamp}} \n' +
        'Test coverage: {{build.coverage}} \n' +
        'Build took: {{build.took}}\n' +
    '</pre>' +
    '</div></div>' +
    '<div class="row">' +
    '<div class="col-xs-12">' +
    '<pipeline v-if="build" v-bind:pipeline="build" v-on:step="selectStep"></pipeline>' +
    '</div> </div>' +
    '</div>',
    methods: {
        selectStep: function(step){
            return this.$emit('step', step);
        }
    }
});

Vue.component('pipeline-group', {
    props: ['step'],
    template: '<li class="build-group"><div class="build-header"></div>' +
                '<ul class="jobs-container">' +
                    '<li v-on:click="selectStep" :class="{active: step.selected}"><div class="job">' +
                        '<span v-bind:class="status_class(step)" aria-hidden="true"></span>' +
                    '<span>{{ step.name }}</span>' +
                '</div></li></ul></li>',
    methods: {
        status_class: function(step){
            var icon = 'glyphicon-remove';
            var text_color = 'text-danger';

            if(step.status == "Done") {
            	if (step.exitcode == 0) {
					text_color = 'text-success';
					icon = 'glyphicon-ok';
				}
            } else {
				text_color = 'text-muted';
				icon = 'glyphicon-hourglass';
			}

            return ['glyphicon', icon, text_color]
        },
        selectStep: function(){
            return this.$emit('step', this.step);
        }
    }
});

Vue.component('pipeline', {
    props: ['pipeline'],
    template: '<div class="pipeline-container"><ul class="pipeline">' +
        '<pipeline-group v-for="step in pipeline.steps" v-bind:step="step" v-bind:key="step.name" v-on:step="selectStep"></pipeline-group>' +
        '</ul></div>',
    methods: {
        selectStep: function(step){
            this.pipeline.steps.forEach((s)=>s.selected = false);
            step.selected = true;
            return this.$emit('step', step);
        }
    }
});

Vue.component('console-output', {
    props: ['buildOutput'],
    template: '<div>' +
    '<h4>Build step output:</h4>' +
    '<div class="output-log" v-html="consolified(buildOutput)"></div>' +
    '</div>',
    methods: {
        consolified: function(input){
            var ansi_up = new AnsiUp;
            var html = ansi_up.ansi_to_html(input).replace(/(?:\r\n|\r|\n)/g, '<br/>');
            return html;
        }
    }
});

const repoStorage = {
    fetchAll: function(app){
        return $.ajax({
            url: '/repos',
            type: 'GET',
            success: function(repos){ app.repos = repos; },
            error: function(error){ app.error = error; }
        });
    }
};

const buildStorage = {
    fetchAll: function(app, repoId){
        return $.ajax({
            url: '/repo/'+repoId,
            type: 'GET',
            success: function(repo){
                repo.builds.forEach((b) => b.selected = false);
                app.selectedRepo = repo;
            },
            error: function(error){ app.error = error; }
        });
    },
    fetch: function(app, repoId, buildId){
        return $.ajax({
            url: '/repo/'+repoId +"/build/"+buildId,
            type: 'GET',
            success: function(buildInfo){
                buildInfo.steps.forEach((s) => s.selected = false);
                app.selectedBuild = buildInfo
            },
            error: function(error){ app.error = error; }
        });
    }
};

document.addEventListener('DOMContentLoaded', function(){
    var app = new Vue({
        el: "#app",
        data: {
            repoSearch: '',
            repos: [],
            buildInfo: {},
            selectedRepo: {},
            selectedBuild: undefined,
            selectedStep: {build: undefined}
        },
        computed: {
            sortedRepoList: function(){
                return this.repoList.sort();
            },
            sortedBuilds: function(){
                if(!this.selectedRepo.builds){ return []; }
                return this.selectedRepo.builds.sort(function(a,b){return a.timestamp < b.timestamp;})
            }
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
                buildStorage.fetchAll(this, this.selectedRepo.id);
            },
            getBuild : function(build){
                buildStorage.fetch(this, this.selectedRepo.id, this.selectedBuild.number);
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
            }
        }
    });
    repoStorage.fetchAll(app);
});