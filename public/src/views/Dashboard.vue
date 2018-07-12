<template>
<div class="animated fadeIn">
  <b-table class="mb-0 table-outline" responsive="sm" hover :items="gettableItems" :fields="tableFields" head-variant="light">
    <div slot="status" class="avatar" slot-scope="item">
      <span class="avatar-status" v-bind:class="{ 'bg-success': item.item.success == true, 'bg-danger': item.item.success == false, 'bg-secondary': item.Name == '' }"></span>
    </div>
    <div slot="name" slot-scope="item">
      <router-link :to="{ name: 'ShowBuild', params: {org: item.item.organisation, project: item.item.name, buildnumber: item.item.number, item: item.item } }">
        <div>{{item.item.name}}</div>
      </router-link>
    </div>
    <div slot="version" slot-scope="item">
      <div :id="item.item.number">{{item.item.number}}</div>
    </div>
    <div slot="duration" slot-scope="item">
      <div class="clearfix">
        <div class="float-center">
          {{item.item.duration}}
        </div>
      </div>
    </div>
    <div slot="coverage" slot-scope="item">
      {{item.item.coverage}}
    </div>
    <div slot="activity" slot-scope="item">
      {{timeSince(new Date(item.item.timestamp))}} ago
    </div>
  </b-table>
</div>
</template>

<script>
import { Callout } from '../components/'
import Vue from 'vue'
import Resource from 'vue-resource'
import axios from 'axios'

Vue.use(Resource)

export default {
  name: 'dashboard',
  components: {
    Callout
  },
  data: function () {
    return {
      selected: 'Month',
      tableFields: {
        status: {
          label: 'Status'
        },
        name: {
          label: 'Name'
        },
        version: {
          label: 'Version'
        },
        duration: {
          label: 'Duration'
        },
        coverage: {
          label: 'Coverage'
        },
        activity: {
          label: 'Last executed'
        }
      }
    }
  },
  methods: {
    gettableItems () {
      console.log('hello')
      let promise = axios.get(process.env.API_BASE_URL + '/builds?max=50')

      // Must return a promise that resolves to an array of items
      return promise.then((data) => {
        console.log('promised call')
        console.log(data.data.length)
        console.log(data.data[0])
        return (data.data || [])
      })
    },
    timeSince (date) {
      var seconds = Math.floor((new Date() - date) / 1000)
      var interval = Math.floor(seconds / 31536000)
      if (interval > 1) {
        return interval + ' years'
      }
      interval = Math.floor(seconds / 2592000)
      if (interval > 1) {
        return interval + ' months'
      }
      interval = Math.floor(seconds / 86400)
      if (interval > 1) {
        return interval + ' days'
      }
      interval = Math.floor(seconds / 3600)
      if (interval > 1) {
        return interval + ' hours'
      }
      interval = Math.floor(seconds / 60)
      if (interval > 1) {
        return interval + ' minutes'
      }
      return Math.floor(seconds) + ' seconds'
    }
  }
}
</script>
