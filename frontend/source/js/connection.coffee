class window.Connection
  login: (credentials, onSuccess, onFailure)->
    promise = $.ajax(url: "/api/sessions", method: "POST", data: JSON.stringify(credentials))
    if onSuccess
      promise = promise.success (data)-> onSuccess(data)
    if onFailure
      promise = promise.fail (response)-> onFailure(response.responseText)

  logout: ->
    $.ajax(url: "/api/sessions/#{State.Session.id}", method: "DELETE")
