module.exports = api = {
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
