import Vue from 'vue'
import Router from 'vue-router'

// Containers
import Full from '@/containers/Full'

// Views
import Dashboard from '@/views/Dashboard'
import BuildLog from '@/views/BuildLog'
import ProjectBuilds from '@/views/ProjectBuilds'
import ShowBuild from '@/views/ShowBuild'

Vue.use(Router)

const router = new Router({
  mode: 'hash', // Demo is living in GitHub.io, so required!
  linkActiveClass: 'open active',
  scrollBehavior: () => ({ y: 0 }),
  routes: [
    {
      path: '/',
      redirect: '/dashboard',
      name: 'Home',
      component: Full,
      children: [
        {
          path: '',
          name: 'Dashboard',
          component: Dashboard
        },
        {
          path: '/repo/:org/:project/builds',
          name: 'ProjectBuilds',
          component: ProjectBuilds,
          props: true
        },
        {
          path: '/repo/:org/:project/builds/:buildnumber',
          name: 'ShowBuild',
          component: ShowBuild,
          props: true
        },

        {
          path: '/repo/:org/:project/build/:buildnumber/step/:step',
          name: 'BuildLog',
          component: BuildLog,
          props: true
        }
      ]
    }
  ]
})
export default router
