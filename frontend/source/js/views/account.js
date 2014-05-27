(function() {
  "use strict"

  App.Views.AccountPage = function() {
    view.View.call(this, "div")
    this.el.className = "account"

    this.header = this.createChild(App.Views.LoggedInHeader)
    this.header.render()

    this.form = this.createChild(App.Views.AccountForm)
    this.form.render()

    this.fetch()
  }

  App.Views.AccountPage.prototype = Object.create(view.View.prototype)

  var p = App.Views.AccountPage.prototype

  p.render = function() {
    this.el.innerHTML = ""
    this.el.appendChild(this.header.el)
    this.el.appendChild(this.form.el)
    return this.el
  }

  p.fetch = function() {
    conn.getAccount({
      succeeded: function(data) {
        this.form.model = data
        this.form.render()
      }.bind(this)
    })
  }

  App.Views.AccountForm = function() {
    view.View.call(this, "form")
    this.model = {}
    this.el.addEventListener("submit", this.update.bind(this))
  }

  App.Views.AccountForm.prototype = Object.create(view.View.prototype)

  var p = App.Views.AccountForm.prototype
  p.template = JST["templates/account/form"]

  p.render = function() {
    this.el.innerHTML = this.template()
    var email
    if(this.model.email) {
      email = this.model.email
    } else {
      email = ""
    }
    this.el.querySelector("#email").value = email

    return this.el
  }

  p.update = function(e) {
    e.preventDefault()

    var form = e.target
    if(form.elements.newPassword.value != form.elements.passwordConfirmation.value) {
      alert("New password and confirmation must match.")
      return
    }

    var update = {}
    update.email = form.elements.email.value
    update.existingPassword = form.elements.existingPassword.value
    update.newPassword = form.elements.newPassword.value

    conn.updateAccount(update, {
      succeeded: function() {
        form.elements.existingPassword.value = ""
        form.elements.newPassword.value = ""
        form.elements.passwordConfirmation.value = ""
        alert("Update succeeded")
      }.bind(this),
      failed: function(data) {
        alert(data)
      }.bind(this)
    })
  }
})()
