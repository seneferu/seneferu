<template>
    <li>
        <div><a v-on:click="$emit('select_repo', repo)">{{ repo.org }}/{{ repo.name }}</a></div>
    </li>
</template>

<script>
export default {
    props: {
        repo : { type: Object }
    },
    methods: { }
};
</script>

<style>
</style>
