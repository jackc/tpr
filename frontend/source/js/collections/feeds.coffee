class App.Collections.Feeds extends Backbone.Collection
  model: App.Models.Feed
  url: ->
    "api/feeds?sessionID=#{State.Session.id}"
