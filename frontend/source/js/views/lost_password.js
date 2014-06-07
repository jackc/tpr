(function() {
  "use strict"

  App.Views.LostPasswordPage = function() {
    view.View.call(this, "div")
    this.el.className = "lostPassword"
  }

  App.Views.LostPasswordPage.prototype = Object.create(view.View.prototype)

  var p = App.Views.LostPasswordPage.prototype
  p.template = JST["templates/lost_password_page"]

  p.listen = function() {
    var form = this.el.querySelector("form")
    form.addEventListener("submit", this.requestPasswordReset.bind(this))
  }

  p.render = function() {
    this.el.innerHTML = this.template()
    this.listen()
    return this.el
  }

  p.requestPasswordReset = function(e) {
    e.preventDefault()
    var form = e.currentTarget
    conn.requestPasswordReset(form.elements.email.value, {
      succeeded: this.onRequestPasswordResetSuccess,
      failed: function(_, response) { this.onRequestPasswordResetFailure(response.responseText) }.bind(this)
    })
  }

  p.onRequestPasswordResetSuccess = function(data) {
    alert("Please check your email for reset information")
  }

  p.onRequestPasswordResetFailure = function(response) {
    alert("Failure resetting password")
  }
})()
