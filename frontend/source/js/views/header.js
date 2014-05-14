(function() {
  "use strict"

  App.Views.LoggedInHeader = function() {
    view.View.call(this, "header")
  }

  App.Views.LoggedInHeader.prototype = Object.create(view.View.prototype)

  var p = App.Views.LoggedInHeader.prototype
  p.template = JST["templates/logged_in_header"]

  p.listen = function() {
    var logoutLink = this.el.querySelector("a.logout")
    logoutLink.addEventListener("click", this.logout.bind(this))
  }

  p.render = function() {
    this.el.innerHTML = this.template()
    this.listen()
    return this.el
  }

  p.logout = function(e) {
    e.preventDefault()
    conn.logout()
    State.Session.clear()
    window.router.navigate("login")
  }
})()
