class App.Views.HomePage extends App.Views.Base
  className: 'home'

  constructor: ->
    super()

    @collection = new App.Collections.UnreadItems

    @header = @createChild App.Views.LoggedInHeader
    @header.render()

    @actions = @createChild App.Views.Actions, collection: @collection
    @actions.render()

    @unreadItemsView = @createChild App.Views.UnreadItemsList, collection: @collection
    @collection.fetch()

  render: ->
    @el.innerHTML = ""
    @el.appendChild @header.el
    @el.appendChild @actions.el
    @el.appendChild @unreadItemsView.el
    @el

class App.Views.Actions extends App.Views.Base
  template: JST["templates/home/actions"]
  className: 'pageActions'

  constructor: (options)->
    super()
    @collection = options.collection

  render: ->
    @el.innerHTML = @template()
    @listen()

  listen: ->
    markAllReadLink = @el.querySelector("a.markAllRead")
    markAllReadLink.addEventListener("click", (e)=> @markAllRead(e))

  markAllRead: (e)->
    e.preventDefault()
    @collection.markAllRead()

class App.Views.UnreadItemsList extends App.Views.Base
  tagName: 'ul'
  className: 'unreadItems'

  constructor: (options)->
    super()

    @collection = options.collection
    @collection.changed.add => @render()

    @boundKeyDown = (e)=> @keyDown(e)
    document.addEventListener 'keydown', @boundKeyDown

  render: ->
    @el.innerHTML = ""

    @itemViews = for model in @collection.items
      @createChild App.Views.UnreadItem, model: model
    if @itemViews.length > 0
      @selected = @itemViews[0]
      @selected.select()

    for itemView in @itemViews
      itemView.render()
      @el.appendChild itemView.el
    @el

  keyDown: (e)->
    switch e.which
      # j
      when 74 then @selectNext()
      # k
      when 75 then @selectPrevious()
      # v
      when 86 then @viewSelected()
      # m and shift
      when 77 then if e.shiftKey then @collection.markAllRead()

  selectNext: ->
    return if @itemViews.length == 0

    idx = @itemViews.indexOf(@selected) + 1
    return if idx >= @itemViews.length

    @selected.deselect()
    @selected.render()

    @selected = @itemViews[idx]

    @selected.select()
    @selected.render()
    @selected.ensureVisible()

  selectPrevious: ->
    return if @itemViews.length == 0

    idx = @itemViews.indexOf(@selected) - 1
    return if idx < 0

    @selected.deselect()
    @selected.render()

    @selected = @itemViews[idx]

    @selected.select()
    @selected.render()
    @selected.ensureVisible()

  viewSelected: ->
    return unless @selected
    @selected.view()

  remove: ->
    document.removeEventListener 'keydown', @boundKeyDown
    super()

class App.Views.UnreadItem extends App.Views.Base
  tagName: 'li'
  template: JST["templates/item"]

  constructor: (options)->
    super()
    @model = options.model

  listen: ->
    viewLink = @el.querySelector("a")
    viewLink.addEventListener("click", (e) => @view(e))

  render: ->
    @el.innerHTML = @template(@model)
    if @isSelected
      @el.className = 'selected'
    else
      @el.className = ''
    @el

  view: (e)->
    e.preventDefault() if e
    @model.markRead()
    window.open(@model.url)

  select: ->
    @isSelected = true

  deselect: ->
    @model.markRead()
    @isSelected = false

  ensureVisible: ->
    @el.scrollIntoView() unless @isEntirelyVisible()

  isEntirelyVisible: ->
    rect = @el.getBoundingClientRect()
    rect.top >= 0 and rect.left >= 0 and rect.bottom <= window.innerHeight and rect.right <= window.innerWidth
