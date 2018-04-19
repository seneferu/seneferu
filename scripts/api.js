module.exports = api = {
    _success: function(cb){return (val) => cb(undefined, val) },
    _fail: function(cb){return (err) => cb(err, undefined) },
    baseUrl: "", // Mainly good for testing

    /* API is a progression into the builds
        First you get a list of repos - then you can get a single repo, but it doesn't contain more info - repos()
        Then you can fill out the repos, 'builds'-attribute with a list of builds - builds()
        And then you can fill out a single builds, 'steps'-attribute - steps()

        Special logic considered:
         - When a build is in the running, created or started -state, it should continually be updated, aka this
            API-client should have logic to cache objects and if the right parameters are present, should update them as
            necessary.
    */

    // List of repos
    repos: function(cb){
        return $.ajax({
            url: api.baseUrl + "/repos",
            type: "GET",
            success: api._success(cb),
            error: api._fail(cb)
        });
    },
    // Single repo object
    repo: function(cb, org, name){
        return $.ajax({
            url: api.baseUrl + "/repo/"+org+"/"+name,
            type: "GET",
            success: api._success(cb),
            error: api._fail(cb)
        });
    },
    // List of SimpleBuild
    builds: function(cb, org, name){
        return $.ajax({
            url: api.baseUrl + "/repo/"+org+"/"+name+"/builds",
            type: "GET",
            success: api._success(cb),
            error: api._fail(cb)
        });
    },
    // List of BuildStep
    steps: function(cb, org, repo, buildId) {
        return $.ajax({
            url: api.baseUrl + "/repo/" + org + "/" + repo + "/build/" + buildId,
            type: "GET",
            success: api._success(cb),
            error: api._fail(cb)
        });
    },

    /* Use as Vue plugin */
    install: function(Vue, options) {
        Vue.prototype.$api = api;
    }
};
