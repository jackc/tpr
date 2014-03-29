class window.Connection
  login: (credentials, onSuccess, onFailure)->
    promise = $.ajax(url: "/api/sessions", method: "POST", data: JSON.stringify(credentials))
    if onSuccess
      promise = promise.success (data)-> onSuccess(data)
    if onFailure
      promise = promise.fail (response)-> onFailure(response.responseText)

  logout: ->
    $.ajax(url: "/api/sessions/#{State.Session.id}", method: "DELETE")

  # registration -- name, password, passwordConfirmation
  register: (registration, onSuccess, onFailure)->
    promise = $.post("/api/register", JSON.stringify(registration))
    if onSuccess
      promise = promise.success (data)-> onSuccess(data)
    if onFailure
      promise = promise.fail (response)-> onFailure(response.responseText)

  getFeeds: (onSuccess)->
    promise = $.getJSON("/api/feeds")
    if onSuccess
      promise = promise.success (data)-> onSuccess(data)

  deleteSubscription: (feedID)->
    $.ajax(url: "api/subscriptions/#{feedID}", method: "DELETE")

  getUnreadItems: (onSuccess)->
    promise = $.getJSON("/api/items/unread")
    if onSuccess
      promise = promise.success (data)->
        models = for record in data
          model = new App.Models.Item
          for k, v of record
            model[k] = v
          model
        onSuccess(models)

  markItemRead: (itemID)->
    $.ajax(url: "/api/items/unread/#{itemID}", method: "DELETE")
