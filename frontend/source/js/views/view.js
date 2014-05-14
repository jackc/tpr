(function() {
  "use strict"

  window.view = {}
  view.View = function(tagName) {
    this.el = document.createElement(tagName)
    this.children = []
  }

  view.View.prototype = {
    createChild: function(klass, options) {
      var child = new klass(options)
      this.attachChild(child)
      return child
    },

    removeChild: function(child) {
      this.detatchChild(child)
      child.remove()
    },

    removeAllChildren: function() {
      // dup children because child.remove() will call parent.detatchChild
      // which will mutate the array while it is being interated over
      var children = this.children.slice(0)
      children.forEach(function(c) {
        c.remove()
      })
    },

    attachChild: function(child) {
      this.children.push(child)
      child.parent = this
    },

    detatchChild: function(child) {
      var idx = this.children.indexOf(child)
      this.children.splice(idx, 1)
      child.parent = null
    },

    remove: function() {
      if(this.parent) {
        this.parent.detatchChild(this)
      }
      this.removeAllChildren()

      var parentNode = this.el.parentNode
      if(parentNode) {
        parentNode.removeChild(this.el)
      }
    }
  }

  view.create = function(tagName) {
    var v = new view.View(tagName)
    return Object.create(v)
  }
})()
