class window.Connection
  login: (credentials, onSuccess, onFailure)->
    reqwest
      url: "/api/sessions"
      method: "post"
      contentType: "application/json"
      data: JSON.stringify(credentials)
      success: onSuccess
      error: (r)-> onFailure(r.responseText)

  logout: ->
    reqwest
      url: "/api/sessions/#{State.Session.id}"
      method: "delete"

  # registration -- name, password, passwordConfirmation
  register: (registration, onSuccess, onFailure)->
    reqwest
      url: "/api/register"
      method: "post"
      data: JSON.stringify(registration)
      success: onSuccess
      error: (r)-> onFailure(r.responseText)

  getFeeds: (onSuccess)->
    reqwest
      url: "/api/feeds"
      method: "get"
      headers: {"X-Authentication" : State.Session.id}
      success: onSuccess

  subscribe: (url, onSuccess)->
    data =
      url: url

    reqwest
      url: "/api/subscriptions"
      method: "post"
      headers: {"X-Authentication" : State.Session.id}
      data: JSON.stringify(data)
      success: onSuccess

  deleteSubscription: (feedID)->
    reqwest
      url: "api/subscriptions/#{feedID}"
      method: "delete"
      headers: {"X-Authentication" : State.Session.id}

  getUnreadItems: (onSuccess)->
    reqwest
      url: "/api/items/unread"
      method: "get"
      headers: {"X-Authentication" : State.Session.id}
      success: (data)->
        models = for record in data
          model = new App.Models.Item
          for k, v of record
            model[k] = v
          model
        onSuccess(models)

  markItemRead: (itemID)->
    reqwest
      url: "/api/items/unread/#{itemID}"
      method: "DELETE"
      headers: {"X-Authentication" : State.Session.id}

  markAllRead: (itemIDs, onSuccess)->
    reqwest
      url: "/api/items/unread/mark_multiple_read"
      method: "post"
      headers: {"X-Authentication" : State.Session.id}
      data: JSON.stringify({itemIDs: itemIDs})
      success: onSuccess
