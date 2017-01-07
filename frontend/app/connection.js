import signals from 'signals'
import Session from './session.js'
import { router } from 'react-router'

class Connection {
  constructor() {
    this.pendingCount = 0
    this.firstAjaxStarted = new signals.Signal()
    this.lastAjaxFinished = new signals.Signal()
  }

  incAjax() {
    this.pendingCount++;
    if (this.pendingCount === 1) {
      this.firstAjaxStarted.dispatch()
    }
  }

  decAjax() {
    this.pendingCount--;
    if (this.pendingCount === 0) {
      this.lastAjaxFinished.dispatch()
    }
  }

  ajax(url, method, options) {
    var self = this

    if (options == null) {
      options = {}
    }

    var req = new XMLHttpRequest()
    req.open(method, url, true)

    if (Session.id) {
      req.setRequestHeader("X-Authentication", Session.id)
    }

    if (options.contentType) {
      req.setRequestHeader("Content-Type", options.contentType)
    }

    if (options.headers) {
      var headers = options.headers;
      for (k in headers) {
        v = headers[k]
        req.setRequestHeader(k, v)
      }
    }

    req.onload = function() {
      self.decAjax()
      var data = req.getResponseHeader("Content-Type") === "application/json" ? JSON.parse(req.responseText) : req.responseText

      if (200 <= req.status && req.status <= 299 && options.succeeded) {
        options.succeeded(data, req)
        return
      }

      if (req.status === 403 && req.responseText === "Bad or missing X-Authentication header") {
        Session.clear()
        window.location.reload()
      }

      if (options.failed) {
        options.failed(data, req)
      }
    }

    req.onerror = function() {
      self.decAjax()
      options.failed(undefined, req)
    }

    this.incAjax()

    req.send(options.data)
  }

  get(url, options) {
    this.ajax(url, "GET", options)
  }

  post(url, options) {
    this.ajax(url, "POST", options)
  }

  patch(url, options) {
    this.ajax(url, "PATCH", options)
  }

  delete(url, options) {
    this.ajax(url, "DELETE", options)
  }

  mergeCallbacks(options, callbacks) {
    if (callbacks) {
      options.succeeded = callbacks.succeeded
      options.failed = callbacks.failed
    }

    return options
  }

  login(credentials, callbacks) {
    var options = {
      contentType: "application/json",
      data: JSON.stringify(credentials)
    }

    options = this.mergeCallbacks(options, callbacks)

    this.post("/api/sessions", options)
  }

  logout() {
    return this.delete("/api/sessions/" + Session.id)
  }

  register(registration, callbacks) {
    var options = {
      data: JSON.stringify(registration)
    }

    options = this.mergeCallbacks(options, callbacks)

    return this.post("/api/register", options)
  }

  requestPasswordReset(email, callbacks) {
    var options = {
      contentType: "application/json",
      data: JSON.stringify({"email": email})
    }

    options = this.mergeCallbacks(options, callbacks)

    return this.post("/api/request_password_reset", options)
  }

  resetPassword(reset, callbacks) {
    var options = {
      contentType: "application/json",
      data: JSON.stringify(reset)
    }

    options = this.mergeCallbacks(options, callbacks)

    this.post("/api/reset_password", options)
  }

  getAccount(callbacks) {
    var options = this.mergeCallbacks({}, callbacks)

    return this.get("/api/account", options)
  }

  updateAccount(update, callbacks) {
    var options = {
      data: JSON.stringify(update)
    }

    options = this.mergeCallbacks(options, callbacks)

    return this.patch("/api/account", options)
  }

  getFeeds(callbacks) {
    var options = this.mergeCallbacks({}, callbacks)

    if (options.succeeded) {
      var succeeded = options.succeeded;
      options.succeeded = function(data, req) {
        data.forEach(function(feed) {
          ["last_fetch_time", "last_failure_time", "last_publication_time"].forEach(function(name) {
            if (feed[name]) {
              feed[name] = new Date(feed[name]*1000)
            }
          })
        })

        succeeded(data, req);
      };
    }

    this.get("/api/feeds", options)
  }

  subscribe(url, callbacks) {
    var options = {
      data: JSON.stringify({url: url })
    }

    options = this.mergeCallbacks(options, callbacks)

    this.post("/api/subscriptions", options)
  }

  importOPML(formData, callbacks) {
    var options = {
      data: formData
    }

    options = this.mergeCallbacks(options, callbacks)

    this.post("/api/feeds/import", options)
  }

  deleteSubscription(feedID) {
    this.delete("api/subscriptions/" + feedID)
  }

  getUnreadItems(callbacks) {
    var options = this.mergeCallbacks({}, callbacks)

    if (options.succeeded) {
      var succeeded = options.succeeded;
      options.succeeded = function(data, req) {
        data.forEach(function(item) {
          item.publication_time = new Date(item.publication_time*1000)
        })

        succeeded(data, req)
      }
    }

    this.get("/api/items/unread", options)
  }

  markItemRead(itemID, callbacks) {
    var options = this.mergeCallbacks({}, callbacks)

    this.delete("/api/items/unread/" + itemID, options)
  }

  markAllRead(itemIDs, callbacks) {
    var options = {
      data: JSON.stringify({itemIDs: itemIDs})
    }

    options = this.mergeCallbacks(options, callbacks)

    this.post("/api/items/unread/mark_multiple_read", options)
  }

  getArchivedItems(callbacks) {
    var options = this.mergeCallbacks({}, callbacks)

    if (options.succeeded) {
      var succeeded = options.succeeded;
      options.succeeded = function(data, req) {
        data.forEach(function(item) {
          item.publication_time = new Date(item.publication_time*1000)
        })

        succeeded(data, req)
      }
    }

    this.get("/api/items/archived", options)
  }
}

const conn = new Connection
export { conn }
