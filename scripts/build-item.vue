<template>
    <li class='builditem'>
        <a v-on:click="$emit('select_build', build)" :class="[status_class(build), { active: isSelected }]">
            <span v-bind:class="['glyphicon', status_icon(build) ]"></span>
            <span>{{ time(build.timestamp) }}</span>
        </a>
    </li>
</template>

<script>
export default {
    props: {
        build : { type: Object }
    },
    computed: {
        isSelected: function(){ 
            return this.build.selected;
        }
    },
    methods: {
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
};
</script>

<style>
</style>