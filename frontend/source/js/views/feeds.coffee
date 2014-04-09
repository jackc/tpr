class App.Views.FeedsPage extends App.Views.Base
  template: JST["templates/feeds_page"]
  className: 'feeds'

  constructor: ->
    super()

    @header = @createChild App.Views.LoggedInHeader
    @header.render()

    @subscribeForm = @createChild App.Views.SubscribeForm
    @subscribeForm.render()
    @subscribeForm.subscribed.add ()=> @fetch()

    @importForm = @createChild App.Views.ImportForm
    @importForm.render()
    @importForm.imported.add ()=> @fetch()

    @feedsListView = @createChild App.Views.FeedsList, collection: []

    @fetch()

  fetch: ->
    conn.getFeeds().then (data)=>
      @feedsListView.collection = data
      @feedsListView.render()

  render: ->
    @el.innerHTML = ""
    @el.appendChild(@header.el)
    @el.appendChild(@subscribeForm.el)
    @el.appendChild(@importForm.el)
    @el.appendChild(@feedsListView.render())
    @el

class App.Views.SubscribeForm extends App.Views.Base
  tagName: "form"
  className: "subscribe"
  template: JST["templates/feeds/subscribe"]

  constructor: ->
    super()
    @subscribed = new signals.Signal()

  render: ->
    @el.innerHTML = @template()
    @listen()
    @el

  listen: ->
    @el.addEventListener("submit", (e)=> @subscribe(e))

  subscribe: (e)->
    e.preventDefault()

    conn.subscribe(@el.elements.url.value).then =>
      @el.elements.url.value = ""
      @subscribed.dispatch()

class App.Views.ImportForm extends App.Views.Base
  tagName: "form"
  className: "import"
  template: JST["templates/feeds/import"]

  constructor: ->
    super()
    @imported = new signals.Signal()

  render: ->
    @el.innerHTML = @template()
    @listen()
    @el

  listen: ->
    @el.addEventListener("submit", (e)=> @import(e))

  import: (e)->
    e.preventDefault()
    fd = new FormData(e.target)

    $.ajax({
      url: "/api/feeds/import",
      type: "POST",
      data: fd,
      processData: false,
      contentType: false
    }).success =>
      @imported.dispatch()
      alert 'import success'

class App.Views.FeedsList extends App.Views.Base
  tagName: 'ul'

  constructor: (options)->
    super()
    @collection = options.collection

  render: ->
    @el.innerHTML = ""

    @feedViews = for model in @collection
      @createChild App.Views.Feed, model: model
    for feedView in @feedViews
      feedView.render()
      @el.appendChild feedView.el
    @el

class App.Views.Feed extends App.Views.Base
  template: JST["templates/feeds/feed"]
  tagName: 'li'

  constructor: (options)->
    super()
    @model = options.model

  listen: ->
    unsubscribeLink = @el.querySelector("a.unsubscribe")
    unsubscribeLink.addEventListener("click", (e)=> @unsubscribe(e))

  render: ->
    @el.innerHTML = @template(@model)
    @listen()
    @el

  unsubscribe: (e)->
    e.preventDefault()
    if confirm "Are you sure you want to unsubscribe from #{@model.name}?"
      conn.deleteSubscription(@model.id)
      @remove()
