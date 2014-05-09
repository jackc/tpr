(function() {
  "use strict";

  document.addEventListener('DOMContentLoaded', function() {
    window.conn = new Connection;
    window.State = {};
    State.Session = new App.Models.Session;
    State.Session.load();
    window.router = new App.Router;
    window.router.start();
    return new App.Views.WorkingNotice;
  });
})();
