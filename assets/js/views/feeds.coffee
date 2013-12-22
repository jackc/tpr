class App.Views.FeedsPage extends App.Views.Base
  template: _.template($("#feeds_page_template").html())

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

class App.Views.FeedsList extends App.Views.Base
  className: 'feeds'

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
  tagName: 'li'

  render: ->
    @$el.html(@model.get("name"))
    @