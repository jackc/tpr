class App.Views.HomePage extends Backbone.View
  template: _.template($("#home_page_template").html())

  events:
    'click a.logout' : 'logout'

  initialize: ->
    @unreadItems = new App.Collections.UnreadItems()
    @unreadItemsView = new App.Views.UnreadItemsList collection: @unreadItems
    @unreadItems.fetch()

  render: ->
    @$el.html @template()
    @$el.append @unreadItemsView.render().$el
    @

  logout: (e)->
    e.preventDefault()
    authenticationService = new App.Services.Authentication
    authenticationService.logout().success ->
      Backbone.history.navigate('login', true)


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
