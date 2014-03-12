class App.Views.RegisterPage extends App.Views.Base
  template: _.template($("#register_page_template").html())
  className: 'register'

  initialize: (options)->
    super()
    @registrationService = options.registrationService

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
    @registrationService.register(registration)
      .success(@onRegistrationSuccess)
      .fail(@onRegistrationFailure)

  onRegistrationSuccess: ->
    Backbone.history.navigate('home', true)

  onRegistrationFailure: (response)->
    alert response.responseText

  login: (e)->
    e.preventDefault()
    Backbone.history.navigate('login', true)

  render: ->
    @$el.html @template()
    @
