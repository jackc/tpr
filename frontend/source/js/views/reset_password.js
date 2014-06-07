(function() {
  "use strict"

  App.Views.ResetPasswordPage = function() {
    view.View.call(this, "div")
    this.el.className = "lostPassword"
    this.token = window.location.hash.split("=")[1]
  }

  App.Views.ResetPasswordPage.prototype = Object.create(view.View.prototype)

  var p = App.Views.ResetPasswordPage.prototype
  p.template = JST["templates/reset_password_page"]

  p.listen = function() {
    var form = this.el.querySelector("form")
    form.addEventListener("submit", this.resetPassword.bind(this))
  }

  p.render = function() {
    this.el.innerHTML = this.template()
    this.listen()
    return this.el
  }

  p.resetPassword = function(e) {
    e.preventDefault()
    var form = e.currentTarget
    var reset = {
      "token": this.token,
      "password": form.elements.password.value
    }
    conn.resetPassword(reset, {
      succeeded: this.onResetPasswordSuccess,
      failed: function(_, response) { this.onResetPasswordFailure(response.responseText) }.bind(this)
    })
  }

  p.onResetPasswordSuccess = function(data) {
    alert("Successfully reset password")
    State.Session = new App.Models.Session(data)
    State.Session.save()
    window.router.navigate('home')
  }

  p.onResetPasswordFailure = function(response) {
    alert("Failure resetting password")
  }
})()
