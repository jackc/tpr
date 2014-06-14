(function() {
  "use strict";

  App.Collections.UnreadItems = function() {
    this.changed = new signals.Signal();
    this.items = [];
  };

  App.Collections.UnreadItems.prototype = {
    fetch: function() {
      var self = this;
      conn.getUnreadItems({ succeeded: function(data) {
        self.items = data.map(function(record) {
          var model = new App.Models.Item;
          for (var k in record) {
            model[k] = record[k];
          }
          return model;
        });
        self.changed.dispatch();
      }});
    },

    markAllRead: function() {
      var self = this;
      var itemIDs = this.items.map(function(i) { return i.id; });
      conn.markAllRead(itemIDs, { succeeded: function() {
        self.changed.dispatch();
        self.fetch();
      }});
    }
  };
})();
