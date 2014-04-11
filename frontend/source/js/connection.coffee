class window.Connection
  ajax: (url, method, options={})->
    req = new XMLHttpRequest()
    req.open(method, url, true)

    if State.Session.id
      req.setRequestHeader("X-Authentication", State.Session.id)

    if options.contentType
      req.setRequestHeader("Content-Type", options.contentType)

    if options.headers
      for k, v of options.headers
        req.setRequestHeader(k, v)

    promise = new Promise (resolve, reject)->
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

    promise

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
    @post "/api/register", data: JSON.stringify(registration)

  getFeeds: ()->
    @get "/api/feeds"

  subscribe: (url)->
    @post "/api/subscriptions", data: JSON.stringify({url: url})

  importOPML: (formData)->
    @post "/api/feeds/import", data: formData

  deleteSubscription: (feedID)->
    @delete "api/subscriptions/#{feedID}"

  getUnreadItems: ()->
    @get "/api/items/unread"

  markItemRead: (itemID)->
    @delete "/api/items/unread/#{itemID}"

  markAllRead: (itemIDs)->
    @post "/api/items/unread/mark_multiple_read",
      data: JSON.stringify({itemIDs: itemIDs})
