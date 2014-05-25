(function() {
  "use strict"

  App.Views.RegisterPage = function() {
    view.View.call(this, "div")
    this.el.className = "register"

    this.register = this.register.bind(this)
    this.onRegistrationSuccess = this.onRegistrationSuccess.bind(this)
    this.onRegistrationFailure = this.onRegistrationFailure.bind(this)
  }

  App.Views.RegisterPage.prototype = Object.create(view.View.prototype)

  var p = App.Views.RegisterPage.prototype
  p.template = JST["templates/register_page"]

  p.listen = function() {
    var form = this.el.querySelector("form")
    form.addEventListener("submit", this.register)
  }

  p.register = function(e) {
    e.preventDefault()
    var form = e.target

    if(form.elements.password.value != form.elements.passwordConfirmation.value) {
      alert("Password and confirmation must match.")
      return
    }

    var registration = {
      name: form.elements.name.value,
      email: form.elements.email.value,
      password: form.elements.password.value
    }
    conn.register(registration, {
      succeeded: this.onRegistrationSuccess,
      failed: function(_, response) { this.onRegistrationFailure(response.responseText) }.bind(this)
    })
  }

  p.onRegistrationSuccess = function(data) {
    State.Session = new App.Models.Session(data)
    State.Session.save()
    window.router.navigate("home")
  }

  p.onRegistrationFailure = function(response) {
    alert(response)
  }

  p.render = function() {
    this.el.innerHTML = this.template()
    this.listen()
    return this.el
  }
})()
