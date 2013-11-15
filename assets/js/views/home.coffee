class App.Views.HomePage extends Backbone.View
  template: _.template($("#home_page_template").html())

  initialize: ->
    @header = new App.Views.LoggedInHeader
    @unreadItems = new App.Collections.UnreadItems()
    @unreadItemsView = new App.Views.UnreadItemsList collection: @unreadItems
    @unreadItems.fetch()

  render: ->
    @$el.html @header.render().$el
    @$el.append @template()
    @$el.append @unreadItemsView.render().$el
    @

class App.Views.UnreadItemsList extends Backbone.View
  tagName: 'ul'
  className: 'unreadItems'

  initialize: ->
    @listenTo @collection, 'sync', @render

  render: ->
    @$el.empty()

    @itemViews = for model in @collection.models
      new App.Views.UnreadItem model: model
    for itemView in @itemViews
      @$el.append itemView.render().$el
    @

class App.Views.UnreadItem extends Backbone.View
  tagName: 'li'

  render: ->
    @$el.html(@model.get("title"))
    @
