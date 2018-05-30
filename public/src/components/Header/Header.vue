<template>
<header class="app-header navbar">
  <b-link class="navbar-brand" to="#"></b-link>
    <div class="text-muted col-sm">
  Seneferu cloud native build system
 </div>

  </header>
</template>
<script>

export default {
  name: 'c-header',
  components: {
  },
  methods: {
    sidebarToggle (e) {
      e.preventDefault()
      document.body.classList.toggle('sidebar-hidden')
    },
    sidebarMinimize (e) {
      e.preventDefault()
      document.body.classList.toggle('sidebar-minimized')
    },
    mobileSidebarToggle (e) {
      e.preventDefault()
      document.body.classList.toggle('sidebar-mobile-show')
    },
    asideToggle (e) {
      e.preventDefault()
      document.body.classList.toggle('aside-menu-hidden')
    }
  }
}
</script>
