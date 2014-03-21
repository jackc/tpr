class App.Views.RegisterPage extends App.Views.Base
  template: _.template($("#register_page_template").html())
  className: 'register'

  events:
    "submit form" : "register"
    "click a.login" : "login"

  register: (e)->
    e.preventDefault()
    $form = $(e.currentTarget)
    registration =
      name: $form.find("input[name='name']").val()
      password: $form.find("input[name='password']").val()
      passwordConfirmation: $form.find("input[name='passwordConfirmation']").val()
    conn.register(registration, @onRegistrationSuccess, @onRegistrationFailure)

  onRegistrationSuccess: (data)->
    State.Session = new App.Models.Session data
    State.Session.save()
    $.ajaxSetup headers: {"X-Authentication": State.Session.id}
    Backbone.history.navigate('home', true)

  onRegistrationFailure: (response)->
    alert response

  login: (e)->
    e.preventDefault()
    Backbone.history.navigate('login', true)

  render: ->
    @$el.html @template()
    @
