class App.Views.FeedsPage extends App.Views.Base
  template: _.template($("#feeds_page_template").html())
  className: 'feeds'

  events:
    'submit form.subscribe' : 'subscribe'
    'submit form.import' : 'import'

  constructor: ->
    super()

    @$el = $("<div></div>")
    @$el.addClass @className

    @$el.on "submit", "form.subscribe", (e)=> @subscribe(e)
    @$el.on "submit", "form.import", (e)=> @import(e)

    @header = @createChild App.Views.LoggedInHeader
    @feedsListView = @createChild App.Views.FeedsList, collection: []
    @fetch()

  fetch: ->
    conn.getFeeds (data)=>
      @feedsListView.collection = data
      @feedsListView.render()

  render: ->
    @$el.html @header.render().$el
    @$el.append @template()
    @$el.append @feedsListView.render().$el
    @

  subscribe: (e)->
    e.preventDefault()

    data =
      url: @$("input[name=url]").val()

    $.ajax(
      url: "/api/subscriptions",
      type: "POST",
      data: JSON.stringify(data)
      contentType: "application/json"
    ).success =>
      @$("input[name=url]").val("")
      @feeds.fetch()

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
      @feeds.fetch()
      alert 'import success'

class App.Views.FeedsList extends App.Views.Base
  tagName: 'ul'

  constructor: (options)->
    super()

    @$el = $("<#{@tagName}></#{@tagName}>")
    @collection = options.collection

  render: ->
    @$el.empty()

    @feedViews = for model in @collection
      @createChild App.Views.Feed, model: model
    for feedView in @feedViews
      @$el.append feedView.render().$el
    @

class App.Views.Feed extends App.Views.Base
  template: _.template($("#feeds_page_feed_template").html())
  tagName: 'li'

  constructor: (options)->
    super()

    @$el = $("<#{@tagName}></#{@tagName}>")
    @model = options.model

    @$el.on "click", "a.unsubscribe", (e)=> @unsubscribe(e)

  render: ->
    @$el.html(@template(@model))
    @

  unsubscribe: (e)->
    e.preventDefault()
    if confirm "Are you sure you want to unsubscribe from #{@model.name}?"
      conn.deleteSubscription(@model.id)
      @remove()
