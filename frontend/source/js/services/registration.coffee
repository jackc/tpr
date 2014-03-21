class App.Services.Registration
  register: (registration)->
    $.post("/api/register", JSON.stringify(registration))
      .success (data)->
        State.Session = new App.Models.Session data
        State.Session.save()
        $.ajaxSetup headers: {"X-Authentication": State.Session.id}
