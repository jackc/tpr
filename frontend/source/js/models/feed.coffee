class App.Models.Feed extends Backbone.Model
  url: ->
    "api/subscriptions/#{@get('id')}?sessionID=#{State.Session.id}"
