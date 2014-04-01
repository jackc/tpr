class App.Views.LoginPage extends App.Views.Base
  template: JST["templates/login_page"]
  className: 'login'

  listen: ->
    form = @el.querySelector("form")
    form.addEventListener("submit", (e)=> @login(e))

  login: (e)->
    e.preventDefault()
    form = e.currentTarget
    credentials =
      name: form.elements.name.value
      password: form.elements.password.value
    conn.login(credentials,
      (data)=> @onLoginSuccess(data),
      (response)=> @onLoginFailure(response))

  onLoginSuccess: (data)->
    State.Session = new App.Models.Session data
    State.Session.save()
    window.router.navigate('home')

  onLoginFailure: (response)->
    alert response

  render: ->
    @el.innerHTML = @template()
    @listen()
    @el
