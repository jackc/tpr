class App.Views.LoginPage extends App.Views.Base
  template: _.template($("#login_page_template").html())
  className: 'login'

  initialize: (options)->
    super()
    @authenticationService = options.authenticationService

  events:
    "submit form" : "login"
    "click a.register" : "register"

  login: (e)->
    e.preventDefault()
    $form = $(e.currentTarget)
    credentials =
      name: $form.find("input[name='name']").val()
      password: $form.find("input[name='password']").val()
    @authenticationService.login(credentials)
      .success(@onLoginSuccess)
      .fail(@onLoginFailure)

  register: (e)->
    e.preventDefault()
    Backbone.history.navigate('register', true)

  onLoginSuccess: ->
    Backbone.history.navigate('home', true)

  onLoginFailure: (response)->
    alert response.responseText

  render: ->
    @$el.html @template()
    @
