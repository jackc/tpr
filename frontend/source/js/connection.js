(function() {
  "use strict";

  window.Connection = function() {
    this.firstAjaxStarted = new signals.Signal();
    this.lastAjaxFinished = new signals.Signal();
  };

  Connection.prototype = {
    pendingCount: 0,

    ajax: function(url, method, options) {
      if (options == null) {
        options = {};
      }

      var req = new XMLHttpRequest();
      req.open(method, url, true);

      if (State.Session.id) {
        req.setRequestHeader("X-Authentication", State.Session.id);
      }

      if (options.contentType) {
        req.setRequestHeader("Content-Type", options.contentType);
      }

      if (options.headers) {
        var headers = options.headers;
        for (k in headers) {
          v = headers[k];
          req.setRequestHeader(k, v);
        }
      }

      var promise = new Promise(function(resolve, reject) {
        req.onload = function() {
          if (200 <= req.status && req.status <= 299) {
            var data = req.getResponseHeader("Content-Type") === "application/json" ? JSON.parse(req.responseText) : req.responseText;
            resolve(data);
          } else {
            if (req.status === 403 && req.responseText === "Bad or missing X-Authentication header") {
              State.Session.clear();
              window.router.navigate('login');
            }
            reject(req, Error(req.statusText));
          }
        };

        req.onerror = function() {
          reject(req, Error("Network Error"));
        };
      });

      this.pendingCount++;
      if (this.pendingCount === 1) {
        this.firstAjaxStarted.dispatch();
      }

      var self = this;
      var finish = function() {
        self.pendingCount--;
        if (self.pendingCount === 0) {
          self.lastAjaxFinished.dispatch();
        }
      };

      promise.then(finish, finish);

      req.send(options.data);
      return promise;
    },

    get: function(url, options) {
      return this.ajax(url, "GET", options);
    },

    post: function(url, options) {
      return this.ajax(url, "POST", options);
    },

    delete: function(url, options) {
      return this.ajax(url, "DELETE", options);
    },

    login: function(credentials) {
      return this.post("/api/sessions", {
        contentType: "application/json",
        data: JSON.stringify(credentials)
      });
    },

    logout: function() {
      return this.delete("/api/sessions/" + State.Session.id);
    },

    register: function(registration) {
      return this.post("/api/register", {
        data: JSON.stringify(registration)
      });
    },

    getFeeds: function() {
      return this.get("/api/feeds").then(function(data) {
        data.forEach(function(feed) {
          ["last_fetch_time", "last_failure_time", "last_publication_time"].forEach(function(name) {
            if (feed[name]) {
              feed[name] = new Date(feed[name]*1000)
            }
          })
        });
        return data;
      });
    },

    subscribe: function(url) {
      return this.post("/api/subscriptions", {
        data: JSON.stringify({url: url })
      });
    },

    importOPML: function(formData) {
      return this.post("/api/feeds/import", {
        data: formData
      });
    },

    deleteSubscription: function(feedID) {
      return this.delete("api/subscriptions/" + feedID);
    },

    getUnreadItems: function() {
      return this.get("/api/items/unread").then(function(data) {
        data.forEach(function(item) {
          item.publication_time = new Date(item.publication_time*1000)
        });
        return data;
      });
    },

    markItemRead: function(itemID) {
      return this.delete("/api/items/unread/" + itemID);
    },

    markAllRead: function(itemIDs) {
      return this.post("/api/items/unread/mark_multiple_read", {
        data: JSON.stringify({itemIDs: itemIDs})
      });
    }
  };
})();
