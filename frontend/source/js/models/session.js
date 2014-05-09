(function() {
  "use strict";

  App.Models.Session = function(options) {
    if(options) {
      this.name = options.name;
      this.id = options.sessionID;
    }
  };

  App.Models.Session.prototype = {
    load: function() {
      var serializedSession = localStorage.getItem("session");
      if(serializedSession) {
        var session = JSON.parse(serializedSession);
        this.name = session.name;
        this.id = session.id;
      }
    },

    save: function() {
      localStorage.setItem("session", JSON.stringify({name: this.name, id: this.id}));
    },

    clear: function() {
      localStorage.clear();
      State.Session = new App.Models.Session;
    },

    isAuthenticated: function() {
      return !!this.id;
    }
  }
})();


