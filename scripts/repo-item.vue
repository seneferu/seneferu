<template>
    <li>
        <div><a v-on:click='selectRepo'>{{ repo.org }}/{{ repo.name }}</a></div>
    </li>
</template>

<script>
export default {
    props: {
        repo : { type: Object }
    },
    methods: {
        selectRepo: function(){
            this.$emit('reposelected', this.repo)
        }
    }
};
</script>

<style>
</style>
