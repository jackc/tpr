class App.Services.Authentication
  login: (credentials)->
    $.post("/api/sessions", JSON.stringify(credentials))
      .success (data)->
        State.Session = new App.Models.Session data
        State.Session.save()
        $.ajaxSetup headers: {"X-Authentication": State.Session.id}

  logout: ->
    $.ajax(url: "/api/sessions/#{State.Session.id}", method: "DELETE")
    State.Session.clear()
    $.ajaxSetup headers: {}

