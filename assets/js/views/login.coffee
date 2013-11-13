class App.Views.LoginPage extends Backbone.View
  template: _.template($("#login_page_template").html())

  initialize: (options)->
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
