
import Vue from 'vue'
import App from './App.vue'

import Api from './api.js'
import app2 from './app.js'

Vue.use(Api);

var app = new Vue({
    el: "#app",
    render: (h) => h(App) // Why
});

