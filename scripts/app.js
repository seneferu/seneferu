
Vue.component('repo-item', {
    props: ['repo'],
    template: "#repo-item",
    methods: {
        selectRepo: function(){
            this.$emit('reposelected', this.repo)
        }
    }
});

Vue.component('build-item', {
    props: ['build'],
    template: "#build-item",
    computed: {
        isSelected: function(){ return this.build.selected; }
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

            if(step.status == "Done" && step.exitcode == 0){
                text_color = 'text-success';
                icon = 'glyphicon-ok';
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

const api = {
    _success: function(cb){return (val) => cb(undefined, val) },
    _fail: function(cb){return (err) => cb(err, undefined) },

    repos: function(cb){
        return $.ajax({
            url: "/repos",
            type: "GET",
            success: api._success(cb),
            error: api._fail(cb)
        });
    },
    builds: function(cb, repoId){
        return $.ajax({
            url: "/repo/"+repoId+"/builds",
            type: "GET",
            success: api._success(cb),
            error: api._fail(cb)
        });
    },
    build: function(cb, repoId, buildId){
        return $.ajax({
            url: "/repo/"+repoId+"/build/"+buildId,
            type: "GET",
            success: api._success(cb),
            error: api._fail(cb)
        });
    }
};

function cbWrap(errFn, fn){
    return function(err, val){
        if(err){ return errFn(err); }
        return fn(val);
    }
}

document.addEventListener('DOMContentLoaded', function(){
    var wrapWrap = cbWrap.bind(undefined, [function(err){ app.error = err; }]);

    var app = new Vue({
        el: "#app",
        data: {
            repoSearch: '',
            repos: [],
            buildInfo: {},
            builds: {},
            selectedRepo: [],
            selectedBuild: undefined,
            selectedStep: {build: undefined}
        },
        computed: {
            sortedRepoList: function(){
                return this.repoList.sort();
            },
            sortedBuilds: function(){
                return this.builds
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
            }
        }
    });
    api.repos(wrapWrap((val) => app.repos = val));
});