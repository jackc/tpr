(function() {
  "use strict";

  App.Collections.UnreadItems = function() {
    this.changed = new signals.Signal();
    this.items = [];
  };

  App.Collections.UnreadItems.prototype = {
    fetch: function() {
      self = this;
      conn.getUnreadItems().then(function(data) {
        self.items = data.map(function(record) {
          var model = new App.Models.Item;
          for (var k in record) {
            model[k] = record[k];
          }
          return model;
        });
        self.changed.dispatch();
      }).catch(promiseFailed);
    },

    markAllRead: function() {
      self = this;
      var itemIDs = this.items.map(function(i) { return i.id; });
      conn.markAllRead(itemIDs).then(function() {
        self.changed.dispatch();
      }).then(function() {
        self.fetch();
      }).catch(promiseFailed);
    }
  };
})();
