<template>
<div class="animated fadeIn">
  <b-row>
    <b-card style="min-width: 400px;" :header=item.name>
      <b-row>
        <b-col>
          <b-card>
            <div v-html=item.log style="font-family: 'Lucida Console', Monaco, monospace;">
            </div>
          </b-card>
        </b-col>
      </b-row>
      <b-row md="2" sm="2">
        <div>&nbsp;</div>
      </b-row>
      <b-row md="2" sm="2">
        <b-col>
        </b-col>
        <div>&nbsp;</div>
      </b-row>
    </b-card>
  </b-row>
</div>
</template>

<script>
import { Callout } from '../components/'
import Vue from 'vue'
import Resource from 'vue-resource'
import axios from 'axios'
import AnsiUp from 'ansi_up'
Vue.use(Resource)

export default {
  name: 'buildlog',
  components: {
    Callout
  },
  created () {
    var p = this.$route.params
    var url = process.env.API_BASE_URL + '/repo/' + p.org + '/' + p.project + '/build/' + p.buildnumber + '/step/' + p.step
    let promise = axios.get(url)
    // Must return a promise that resolves to an array of items
    return promise.then((data) => {
      this.item = data.data
      const aup = new AnsiUp()
      let logText = aup.ansi_to_html(this.item.log)
      this.item.log = logText.replace(/(?:\r\n|\r|\n)/g, '<br/>')
      return (data.data || [])
    })
  },
  data: function () {
    return {
      item: {}
    }
  },
  methods: {
  }
}
</script>
