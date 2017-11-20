<template>
    <li>
        <div><a v-on:click='selectRepo'>{{ repo.id }}</a></div>
    </li>
</template>

<script>
module.exports = {
    props: ['repo'],
    template: "#repo-item",
    methods: {
        selectRepo: function(){
            this.$emit('reposelected', this.repo)
        }
    }
};
</script>

<style>
</style>
