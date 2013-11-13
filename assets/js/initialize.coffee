$ ->
  window.State = {}
  State.Session = new App.Models.Session
  State.Session.load()

  window.router = new App.Router
  Backbone.history.start()
