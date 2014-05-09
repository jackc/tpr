(function() {
  "use strict";

  App.Router = function(options) {
    if(options) {
      this.name = options.name;
      this.id = options.sessionID;
    }
  };

  App.Router.prototype = {
    routes: {
      login: "login",
      home: "home",
      register: "register",
      feeds: "feeds",
    },

    login: function() {
      this.changePage(App.Views.LoginPage);
    },

    home: function() {
      if(!State.Session.isAuthenticated()) {
        this.navigate("login");
        return;
      }

      this.changePage(App.Views.HomePage);
    },

    register: function() {
      this.changePage(App.Views.RegisterPage);
    },

    feeds: function() {
      if(!State.Session.isAuthenticated()) {
        this.navigate("login");
        return;
      }

      this.changePage(App.Views.FeedsPage);
    },

    changePage: function(pageClass, options) {
      if(this.currentPage) {
        this.currentPage.remove();
      }

      this.currentPage = new pageClass(options);
      var view = document.getElementById("view");
      view.innerHTML = "";
      view.appendChild(this.currentPage.render());
    },

    start: function() {
      var self = this;
      window.addEventListener("hashchange",
        function() { self.change() },
        false);
      this.change();
    },

    change: function() {
      var handler = this.routes[window.location.hash.slice(1)];
      if(!handler) {
        this.navigate("login");
        return;
      }

      return this[handler]();
    },

    navigate: function(route) {
      window.location.hash = "#" + route;
    }
  }
})();
