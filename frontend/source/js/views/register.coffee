class App.Views.RegisterPage extends App.Views.Base
  template: JST["templates/register_page"]
  className: 'register'

  listen: ->
    form = @el.querySelector("form")
    form.addEventListener("submit", (e)=> @register(e))

  register: (e)->
    e.preventDefault()
    form = e.target
    registration =
      name: form.elements.name.value
      email: form.elements.email.value
      password: form.elements.password.value
      passwordConfirmation: form.elements.passwordConfirmation.value
    conn.register registration,
      succeeded: (data)=> @onRegistrationSuccess(data)
      failed: (_, response)=> @onRegistrationFailure(response.responseText)

  onRegistrationSuccess: (data)->
    State.Session = new App.Models.Session data
    State.Session.save()
    window.router.navigate('home')

  onRegistrationFailure: (response)->
    alert response

  render: ->
    @el.innerHTML = @template()
    @listen()
    @el
