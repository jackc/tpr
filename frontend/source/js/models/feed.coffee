class App.Models.Feed extends Backbone.Model
  url: ->
    "api/subscriptions/#{@get('id')}"
