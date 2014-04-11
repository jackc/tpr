(function() {
  "use strict";

  App.Views.WorkingNotice = function() {
    var self = this;
    this.el = document.getElementById("working_notice");
    this.el.style.display = "none";

    conn.firstAjaxStarted.add(function() {
      self.el.style.display = "";
    });
    conn.lastAjaxFinished.add(function() {
      self.el.style.display = "none";
    });
  };
})();
