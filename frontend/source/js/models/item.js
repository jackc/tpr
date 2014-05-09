(function() {
  "use strict";

  App.Models.Item = function() {};
  App.Models.Item.prototype.markRead = function() {
    if(this.isRead) {
      return;
    }
    conn.markItemRead(this.id);
    this.isRead = true;
  };
})();
