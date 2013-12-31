class App.Views.FeedsPage extends App.Views.Base
  template: _.template($("#feeds_page_template").html())
  className: 'feeds'

  events:
    'submit form.subscribe' : 'subscribe'
    'submit form.import' : 'import'

  initialize: ->
    super()
    @header = @createChild App.Views.LoggedInHeader
    @feeds = @createChild App.Collections.Feeds
    @feedsListView = @createChild App.Views.FeedsList, collection: @feeds
    @feeds.fetch()

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
      url: "/api/subscriptions?sessionID=#{State.Session.id}",
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
      url: "/api/feeds/import?sessionID=#{State.Session.id}",
      type: "POST",
      data: fd,
      processData: false,
      contentType: false
    }).success =>
      @feeds.fetch()
      alert 'import success'

class App.Views.FeedsList extends App.Views.Base
  tagName: 'ul'

  initialize: ->
    super()
    @listenTo @collection, 'sync', @render

  render: ->
    @$el.empty()

    @feedViews = for model in @collection.models
      @createChild App.Views.Feed, model: model
    for feedView in @feedViews
      @$el.append feedView.render().$el
    @

class App.Views.Feed extends App.Views.Base
  template: _.template($("#feeds_page_feed").html())
  tagName: 'li'

  events:
    'click a.unsubscribe' : 'unsubscribe'

  render: ->
    @$el.html(@template(@model.toJSON()))
    @

  unsubscribe: (e)->
    e.preventDefault()
    if confirm "Are you sure you want to unsubscribe from #{@model.get('name')}?"
      @model.destroy()
      @remove()
