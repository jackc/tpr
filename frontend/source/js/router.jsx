(function() {
  "use strict";

  App.Views.Layout = React.createClass({
    render: function() {
      return (
        <div>
          <App.Views.LoggedInHeader />
          <this.props.page />
        </div>
      );
    }
  })

  App.Router = function(options) {
    if(options) {
      this.name = options.name;
      this.id = options.sessionID;
    }
  };

  App.Router.prototype = {
    routes: {
      login: "login",
      register: "register",
      lostPassword: "lostPassword",
      resetPassword: "resetPassword",
      home: "home",
      feeds: "feeds",
      account: "account"
    },

    login: function() {
      this.changePage(App.Views.LoginPage)
    },

    lostPassword: function() {
      this.changePage(App.Views.LostPasswordPage)
    },

    resetPassword: function() {
      this.changePage(App.Views.ResetPasswordPage)
    },

    home: function() {
      if(!State.Session.isAuthenticated()) {
        this.navigate("login");
        return;
      }

      this.changePage(App.Views.HomePage)
    },

    register: function() {
      this.changePage(App.Views.RegisterPage)
    },

    feeds: function() {
      if(!State.Session.isAuthenticated()) {
        this.navigate("login");
        return;
      }

      this.changePage(App.Views.FeedsPage)
    },

    account: function() {
      if(!State.Session.isAuthenticated()) {
        this.navigate("login");
        return;
      }

      this.changePage(App.Views.AccountPage)
    },

    changePage: function(pageClass, options) {
      if(!this.reactApp) {
        var el = document.getElementById("view");

        this.reactApp = React.render(
          <App.Views.Layout page={pageClass}/>,
          el
        );
      }

      this.reactApp.setProps({page: pageClass})
    },

    start: function() {
      var self = this;
      window.addEventListener("hashchange",
        function() { self.change() },
        false);
      this.change();
    },

    change: function() {
      var hash = window.location.hash.slice(1)
      var route = hash.split("?")[0]
      var handler = this.routes[route]
      if(!handler) {
        this.navigate("login")
        return
      }

      return this[handler]()
    },

    navigate: function(route) {
      window.location.hash = "#" + route
    }
  }
})();
