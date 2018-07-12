import axios from 'axios'

const items = {
  items: [
    {
      name: 'Dashboard',
      url: '/',
      icon: 'icon-speedometer',
      badge: {
        variant: 'primary'
      }
    },
    {
      name: 'Repositories',
      icon: 'icon-layers',
      children: []
    }
  ]
}

var url = process.env.API_BASE_URL + '/repos'
let promise = axios.get(url)
promise.then((data) => {
  var i = 0
  var repo = {}
  for (i = 0; i < items.items.length; i++) {
    if (items.items[i].name === 'Repositories') {
      repo = items.items[i]
      for (i = 0; i < data.data.length; i++) {
        var r = data.data[i]
        repo.children.push(
          {
            name: r.org + '/' + r.name,
            url: '/repo/' + r.org + '/' + r.name + '/builds',
            icon: 'icon-puzzle'
          }
        )
      }
    }
  }
})

export default items
