class App.Views.HomePage extends App.Views.Base
  template: _.template($("#home_page_template").html())

  events:
    'click a.markAllRead' : 'markAllRead'

  initialize: ->
    super()
    @header = @createChild App.Views.LoggedInHeader
    @unreadItems = @createChild App.Collections.UnreadItems
    @unreadItemsView = @createChild App.Views.UnreadItemsList, collection: @unreadItems
    @unreadItems.fetch()

  render: ->
    @$el.html @header.render().$el
    @$el.append @template()
    @$el.append @unreadItemsView.render().$el
    @

  markAllRead: (e)->
    e.preventDefault()
    $.ajax(url: "/api/items/unread?sessionID=#{State.Session.id}", method: "DELETE")
      .success => @unreadItems.fetch()

class App.Views.UnreadItemsList extends App.Views.Base
  tagName: 'ul'
  className: 'unreadItems'

  initialize: ->
    super()
    @listenTo @collection, 'sync', @render
    $(document).on 'keydown', (e)=> @keyDown(e)

  render: ->
    @$el.empty()

    @itemViews = for model in @collection.models
      @createChild App.Views.UnreadItem, model: model
    if @itemViews.length > 0
      @selected = @itemViews[0]
      @selected.select()

    for itemView in @itemViews
      @$el.append itemView.render().$el
    @

  keyDown: (e)->
    switch e.which
      # j
      when 74 then @selectNext()
      # k
      when 75 then @selectPrevious()
      # v
      when 86 then @viewSelected()

  selectNext: ->
    return if @itemViews.length == 0

    @selected.deselect()
    @selected.render()

    idx = @itemViews.indexOf(@selected) + 1
    if idx >= @itemViews.length
      idx = 0
    @selected = @itemViews[idx]

    @selected.select()
    @selected.render()

  selectPrevious: ->
    return if @itemViews.length == 0

    @selected.deselect()
    @selected.render()

    idx = @itemViews.indexOf(@selected) - 1
    if idx < 0
      idx = @itemViews.length - 1
    @selected = @itemViews[idx]

    @selected.select()
    @selected.render()

  viewSelected: ->
    return unless @selected
    @selected.view()

  remove: ->
    $(document).off 'keydown'
    super()

class App.Views.UnreadItem extends App.Views.Base
  tagName: 'li'
  template: _.template($("#item_template").html())

  events:
    'click a' : 'view'

  render: ->
    @$el.html @template(@model.toJSON())
    if @isSelected
      @$el.addClass 'selected'
    else
      @$el.removeClass 'selected'
    @

  view: (e)->
    e.preventDefault() if e
    @model.markRead()
    window.open(@model.get('url'))

  select: ->
    @isSelected = true

  deselect: ->
    @model.markRead()
    @isSelected = false
