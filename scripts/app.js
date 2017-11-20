
import Vue from 'vue'
import AnsiUp from 'ansi_up'

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
