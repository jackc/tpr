class window.Connection
  ajax: (url, method, options)->
    new Promise (resolve, reject)->
      req = new XMLHttpRequest()
      req.open(method, url, true)

      if options.contentType
        req.setRequestHeader("Content-Type", options.contentType)

      if options.headers
        for k, v of options.headers
          req.setRequestHeader(k, v)

      req.onload = ->
        if 200 <= req.status and req.status <= 299
          data = if req.getResponseHeader("Content-Type") == "application/json"
            JSON.parse(req.responseText)
          else
            req.responseText
          resolve(data)
        else
          reject(req, Error(req.statusText))

      req.onerror = ->
        reject(req, Error("Network Error"))

      req.send(options.data)

  get: (url, options)->
    @ajax(url, "GET", options)

  post: (url, options)->
    @ajax(url, "POST", options)

  delete: (url, options)->
    @ajax(url, "DELETE", options)

  login: (credentials)->
    @post "/api/sessions",
      contentType: "application/json",
      data: JSON.stringify(credentials)

  logout: ->
    @delete "/api/sessions/#{State.Session.id}"

  # registration -- name, password, passwordConfirmation
  register: (registration)->
    @post "/api/register",
      data: JSON.stringify(registration)

  getFeeds: ()->
    @get "/api/feeds",
      headers: {"X-Authentication" : State.Session.id}

  subscribe: (url)->
    data =
      url: url

    @post "/api/subscriptions",
      headers: {"X-Authentication" : State.Session.id},
      data: JSON.stringify(data)

  importOPML: (formData)->
    @post "/api/feeds/import",
      headers: {"X-Authentication" : State.Session.id},
      data: formData

  deleteSubscription: (feedID)->
    @delete "api/subscriptions/#{feedID}",
      headers: {"X-Authentication" : State.Session.id}

  getUnreadItems: ()->
    @get "/api/items/unread",
      headers: {"X-Authentication" : State.Session.id}

  markItemRead: (itemID)->
    @delete "/api/items/unread/#{itemID}",
      headers: {"X-Authentication" : State.Session.id}

  markAllRead: (itemIDs)->
    @post "/api/items/unread/mark_multiple_read",
      headers: {"X-Authentication" : State.Session.id},
      data: JSON.stringify({itemIDs: itemIDs})
