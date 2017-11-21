
import Vue from 'vue'
import api from './api.js'
import app2 from './app.js'

function cbWrap(errFn, fn){
    return function(err, val){
        if(err){ return errFn(err); }
        return fn(val);
    }
}

import App from './App.vue'

var app = new Vue({
    el: "#app",
    render: (h) => h(App)
});

var wrapWrap = cbWrap.bind(undefined, (err) => app.error(err));
api.repos(wrapWrap((val) => app.repos = val ));
