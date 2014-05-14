(function() {
  "use strict"

  App.Views.FeedsPage = function() {
    view.View.call(this, "div")
    this.el.className = "feeds"

    this.fetch = this.fetch.bind(this)

    this.header = this.createChild(App.Views.LoggedInHeader)
    this.header.render()

    this.subscribeForm = this.createChild(App.Views.SubscribeForm)
    this.subscribeForm.render()
    this.subscribeForm.subscribed.add(this.fetch)

    this.importForm = this.createChild(App.Views.ImportForm)
    this.importForm.render()
    this.importForm.imported.add(this.fetch)

    this.feedsListView = this.createChild(App.Views.FeedsList, {collection: []})

    this.fetch()
  }

  App.Views.FeedsPage.prototype = Object.create(view.View.prototype)

  var p = App.Views.FeedsPage.prototype
  p.template = JST["templates/feeds_page"]

  p.fetch = function() {
    conn.getFeeds({
      succeeded: function(data) {
        this.feedsListView.collection = data
        this.feedsListView.render()
      }.bind(this)
    })
  }

  p.render = function() {
    this.el.innerHTML = ""
    this.el.appendChild(this.header.el)
    this.el.appendChild(this.subscribeForm.el)
    this.el.appendChild(this.importForm.el)
    this.el.appendChild(this.feedsListView.render())
    return this.el
  }

  App.Views.SubscribeForm = function() {
    view.View.call(this, "form")
    this.el.className = "subscribe"

    this.subscribed = new signals.Signal()

    this.subscribe = this.subscribe.bind(this)
  }

  App.Views.SubscribeForm.prototype = Object.create(view.View.prototype)

  var p = App.Views.SubscribeForm.prototype
  p.template = JST["templates/feeds/subscribe"]

  p.render = function() {
    this.el.innerHTML = this.template()
    this.listen()
    return this.el
  }

  p.listen = function() {
    this.el.addEventListener("submit", this.subscribe)
  }

  p.subscribe = function(e) {
    e.preventDefault()

    conn.subscribe(this.el.elements.url.value, {
      succeeded: function() {
        this.el.elements.url.value = ""
        this.subscribed.dispatch()
      }.bind(this)
    })
  }

  App.Views.ImportForm = function() {
    view.View.call(this, "form")
    this.el.className = "import"

    this.imported = new signals.Signal()

    this.import = this.import.bind(this)
  }

  App.Views.ImportForm.prototype = Object.create(view.View.prototype)

  var p = App.Views.ImportForm.prototype
  p.template = JST["templates/feeds/import"]

  p.render = function() {
    this.el.innerHTML = this.template()
    this.listen()
    return this.el
  }

  p.listen = function() {
    this.el.addEventListener("submit", this.import)
  }

  p.import = function(e) {
    e.preventDefault()
    var fd = new FormData(e.target)

    conn.importOPML(fd, {
      succeeded: function() {
        this.imported.dispatch()
        alert("import success")
      }.bind(this)
    })
  }

  App.Views.FeedsList = function(options) {
    view.View.call(this, "ul")

    this.collection = options.collection
  }

  App.Views.FeedsList.prototype = Object.create(view.View.prototype)

  var p = App.Views.FeedsList.prototype
  p.template = JST["templates/feeds/import"]

  p.render = function() {
    this.el.innerHTML = ""

    this.feedViews = this.collection.map(function(model) {
      return this.createChild(App.Views.Feed, {model: model})
    }.bind(this))

    this.feedViews.forEach(function(feedView) {
      feedView.render()
      this.el.appendChild(feedView.el)
    }.bind(this))

    return this.el
  }

  App.Views.Feed = function(options) {
    view.View.call(this, "li")

    this.model = options.model

    this.unsubscribe = this.unsubscribe.bind(this)
  }

  App.Views.Feed.prototype = Object.create(view.View.prototype)

  var p = App.Views.Feed.prototype
  p.template = JST["templates/feeds/feed"]

  p.listen = function() {
    var unsubscribeLink = this.el.querySelector("a.unsubscribe")
    unsubscribeLink.addEventListener("click", this.unsubscribe)
  }

  p.render = function() {
    this.el.innerHTML = this.template(this.model)
    this.listen()
    return this.el
  }

  p.unsubscribe = function(e) {
    e.preventDefault()
    if(confirm("Are you sure you want to unsubscribe from " + this.model.name + "?")) {
      conn.deleteSubscription(this.model.id)
      this.remove()
    }
  }
})()
