class App.Collections.UnreadItems extends Backbone.Collection
  model: App.Models.Item
  url: ->
    "api/items/unread"
