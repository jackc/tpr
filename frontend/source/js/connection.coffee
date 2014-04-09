class window.Connection
  login: (credentials)->
    reqwest
      url: "/api/sessions"
      method: "post"
      contentType: "application/json"
      data: JSON.stringify(credentials)

  logout: ->
    reqwest
      url: "/api/sessions/#{State.Session.id}"
      method: "delete"

  # registration -- name, password, passwordConfirmation
  register: (registration)->
    reqwest
      url: "/api/register"
      method: "post"
      data: JSON.stringify(registration)

  getFeeds: ()->
    reqwest
      url: "/api/feeds"
      method: "get"
      headers: {"X-Authentication" : State.Session.id}

  subscribe: (url)->
    data =
      url: url

    reqwest
      url: "/api/subscriptions"
      method: "post"
      headers: {"X-Authentication" : State.Session.id}
      data: JSON.stringify(data)

  deleteSubscription: (feedID)->
    reqwest
      url: "api/subscriptions/#{feedID}"
      method: "delete"
      headers: {"X-Authentication" : State.Session.id}

  getUnreadItems: ()->
    reqwest
      url: "/api/items/unread"
      method: "get"
      headers: {"X-Authentication" : State.Session.id}

  markItemRead: (itemID)->
    reqwest
      url: "/api/items/unread/#{itemID}"
      method: "DELETE"
      headers: {"X-Authentication" : State.Session.id}

  markAllRead: (itemIDs)->
    reqwest
      url: "/api/items/unread/mark_multiple_read"
      method: "post"
      headers: {"X-Authentication" : State.Session.id}
      data: JSON.stringify({itemIDs: itemIDs})
