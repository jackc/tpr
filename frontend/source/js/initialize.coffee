$ ->
  window.conn = new Connection

  window.State = {}
  State.Session = new App.Models.Session
  State.Session.load()

  $.ajaxSetup headers: {"X-Authentication": State.Session.id}

  window.router = new App.Router
  window.router.start()
