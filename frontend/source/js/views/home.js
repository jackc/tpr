(function() {
  "use strict"

  App.Views.HomePage = function() {
    view.View.call(this, "div")
    this.el.className = "home"

    this.collection = new App.Collections.UnreadItems

    this.header = this.createChild(App.Views.LoggedInHeader)
    this.header.render()

    this.actions = this.createChild(App.Views.Actions, {collection: this.collection})
    this.actions.render()

    this.unreadItemsView = this.createChild(App.Views.UnreadItemsList, {collection: this.collection})
    this.collection.fetch()
  }

  App.Views.HomePage.prototype = Object.create(view.View.prototype)

  var p = App.Views.HomePage.prototype

  p.render = function() {
    this.el.innerHTML = ""
    this.el.appendChild(this.header.el)
    this.el.appendChild(this.actions.el)
    this.el.appendChild(this.unreadItemsView.el)
    return this.el
  }


  App.Views.Actions = function(options) {
    view.View.call(this, "div")
    this.el.className = "pageActions"

    this.collection = options.collection
    this.collection.changed.add(this.render.bind(this))
  }

  App.Views.Actions.prototype = Object.create(view.View.prototype)

  var p = App.Views.Actions.prototype

  p.template = JST["templates/home/actions"]

  p.render = function() {
    this.el.innerHTML = this.template({collection: this.collection})
    this.listen()
    return this.el
  }

  p.listen = function() {
    var markAllReadLink = this.el.querySelector("a.markAllRead")
    if(markAllReadLink) {
      markAllReadLink.addEventListener("click", this.markAllRead.bind(this))
    }

    var refreshLink = this.el.querySelector("a.refresh")
    if(refreshLink) {
      refreshLink.addEventListener("click", this.refresh.bind(this))
    }
  }

  p.markAllRead = function(e) {
    e.preventDefault()
    this.collection.markAllRead()
  }

  p.refresh = function(e) {
    e.preventDefault()
    this.collection.fetch()
  }

  App.Views.UnreadItemsList = function(options) {
    view.View.call(this, "ul")
    this.el.className = "unreadItems"

    this.collection = options.collection
    this.collection.changed.add(this.render.bind(this))

    this.keyDown = this.keyDown.bind(this)
    document.addEventListener("keydown", this.keyDown)
  }

  App.Views.UnreadItemsList.prototype = Object.create(view.View.prototype)

  var p = App.Views.UnreadItemsList.prototype

  p.render = function() {
    this.el.innerHTML = ""

    this.itemViews = this.collection.items.map(function(model) {
      return this.createChild(App.Views.UnreadItem, {model: model})
    }.bind(this))
    if(this.itemViews.length > 0) {
      this.selected = this.itemViews[0]
      this.selected.select()
    }

    this.itemViews.forEach(function(itemView) {
      itemView.render()
      this.el.appendChild(itemView.el)
    }.bind(this))

    return this.el
  }

  p.keyDown = function(e) {
    switch(e.which) {
      // j
      case 74:
        this.selectNext()
        break
      // k
      case 75:
        this.selectPrevious()
        break
      // v
      case 86:
        this.viewSelected()
        break
      // a
      case 65:
        if(e.shiftKey) {
          this.collection.markAllRead()
        }
        break
    }
  }

  p.selectNext = function() {
    if(this.itemViews.length == 0) {
      return
    }

    var idx = this.itemViews.indexOf(this.selected) + 1
    if(idx >= this.itemViews.length) {
      return
    }

    this.selected.deselect()
    this.selected.render()

    this.selected = this.itemViews[idx]

    this.selected.select()
    this.selected.render()
    this.selected.ensureVisible()
  }

  p.selectPrevious = function() {
    if(this.itemViews.length == 0) {
      return
    }

    var idx = this.itemViews.indexOf(this.selected) - 1
    if(idx < 0) {
      return
    }

    this.selected.deselect()
    this.selected.render()

    this.selected = this.itemViews[idx]

    this.selected.select()
    this.selected.render()
    this.selected.ensureVisible()
  }

  p.viewSelected = function() {
    if(this.selected) {
      this.selected.view()
    }
  }

  p.remove = function() {
    document.removeEventListener('keydown', this.keyDown)
    view.View.prototype.remove.call(this)
  }

  App.Views.UnreadItem = function(options) {
    view.View.call(this, "li")

    this.model = options.model
  }

  App.Views.UnreadItem.prototype = Object.create(view.View.prototype)

  var p = App.Views.UnreadItem.prototype

  p.template = JST["templates/item"]

  p.listen = function() {
    var viewLink = this.el.querySelector("a")
    viewLink.addEventListener("click", this.view.bind(this))
  }

  p.render = function() {
    this.el.innerHTML = this.template(this.model)
    if(this.isSelected) {
      this.el.className = "selected"
    } else {
      this.el.className = ""
    }
    this.listen()
    return this.el
  }

  p.view = function(e) {
    if(e) {
      e.preventDefault()
    }
    this.model.markRead()
    window.open(this.model.url)
  }

  p.select = function() {
    this.isSelected = true
  }

  p.deselect = function() {
    this.model.markRead()
    this.isSelected = false
  }

  p.ensureVisible = function() {
    if(!this.isEntirelyVisible()) {
      this.el.scrollIntoView()
    }
  }

  p.isEntirelyVisible = function() {
    var rect = this.el.getBoundingClientRect()
    return rect.top >= 0 && rect.left >= 0 && rect.bottom <= window.innerHeight && rect.right <= window.innerWidth
  }
})()
