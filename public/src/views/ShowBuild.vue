<template>
<div class="animated fadeIn">
  <b-row>
    <b-card style="min-width: 100%;" :header=header()>
      <div slot="header">
        {{header(this.params.item.success)}}
        <b-badge :variant=variant(this.params.item.success) class="float-right">{{variantText(this.params.item.success)}}</b-badge>
      </div>
      <b-row>
        <b-col>
          <div v-for="x in item">
            <b-card no-body :class=state(x.exitcode) v-b-toggle="accodianID(x.name)">
              <div slot="header">
                {{x.name}}
                <b-badge :variant=variant(x.exitcode) class="float-right">{{variantText(x.exitcode)}}</b-badge>
                </div>
              <b-collapse :id=accodianID(x.name) visible accordion="accodian(x.name)" role="tabpanel">
                <b-card-body>
                  <p class="card-text" v-html="logify(x.log)" style="font-family: 'Lucida Console', Monaco, monospace;">
                  </p>
                </b-card-body>
              </b-collapse>
            </b-card>
          </div>
        </b-col>
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
  name: 'showbuild',
  components: {
    Callout
  },
  created () {
    var p = this.$route.params
    this.params = p
    var url = process.env.API_BASE_URL + '/repo/' + p.org + '/' + p.project + '/build/' + p.buildnumber
    let promise = axios.get(url)
    // Must return a promise that resolves to an array of items
    return promise.then((data) => {
      this.item = data.data
      console.log(JSON.stringify(this.item))
      return (data.data || [])
    })
  },
  data: function () {
    return {
      params: {},
      item: {}
    }
  },
  methods: {
    state (x) {
      if (x === 0) {
        return 'mb-1 card-accent-success'
      } else {
        return 'mb-1 card-accent-danger'
      }
    },
    logify (x) {
      const aup = new AnsiUp()
      let logText = aup.ansi_to_html(x)
      return logText.replace(/(?:\r\n|\r|\n)/g, '<br/>')
    },
    accodianID (x) {
      return '.accordion-' + x
    },
    header () {
      return this.params.project + ' build ' + this.params.buildnumber
    },
    variant (x) {
      if (x === true || x === 0) {
        return 'success'
      } else {
        return 'danger'
      }
    },
    variantText (x) {
      if (x === true || x === 0) {
        return 'Success'
      } else {
        return 'Failure'
      }
    }
  }
}
</script>
