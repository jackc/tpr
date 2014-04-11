document.addEventListener 'DOMContentLoaded', ->
  window.conn = new Connection

  window.State = {}
  State.Session = new App.Models.Session
  State.Session.load()

  window.router = new App.Router
  window.router.start()

  new App.Views.WorkingNotice
