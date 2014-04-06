class App.Views.Base
  tagName: "div"

  constructor: ->
    @el = document.createElement(@tagName)
    @el.className = @className if @className?
    @children = []

  createChild: (klass, options)->
    child = new klass options
    @attachChild child
    child

  removeChild: (child)->
    @detatchChild child
    child.remove()

  removeAllChildren: ->
    # dup children because child.remove() will call parent.detatchChild
    # which will mutate the array while it is being interated over
    children = @children.slice(0)
    child.remove() for child in children

  attachChild: (child)->
    @children.push child
    child.parent = this

  detatchChild: (child)->
    idx = @children.indexOf(child)
    @children.splice(idx, 1)
    child.parent = null

  remove: ->
    @parent.detatchChild(this) if @parent
    @removeAllChildren()

    parentNode = @el.parentNode
    parentNode.removeChild(@el) if parentNode
