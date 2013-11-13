class App.Views.FeedsPage extends Backbone.View
  template: _.template($("#feeds_page_template").html())

  initialize: ->
    @feeds = new App.Collections.Feeds()
    @feedsListView = new App.Views.FeedsList collection: @feeds
    @feeds.fetch()

  render: ->
    @$el.html @template()
    @$el.append @feedsListView.render().$el
    @

class App.Views.FeedsList extends Backbone.View
  tagName: 'ul'
  className: 'feeds'

  initialize: ->
    @listenTo @collection, 'sync', @render

  render: ->
    @$el.empty()

    @feedViews = for model in @collection.models
      new App.Views.Feed model: model
    for feedView in @feedViews
      @$el.append feedView.render().$el
    @

class App.Views.Feed extends Backbone.View
  tagName: 'li'

  render: ->
    @$el.html(@model.get("name"))
    @