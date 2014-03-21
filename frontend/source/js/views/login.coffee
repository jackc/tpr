class App.Views.LoginPage extends App.Views.Base
  template: _.template($("#login_page_template").html())
  className: 'login'

  events:
    "submit form" : "login"
    "click a.register" : "register"

  login: (e)->
    e.preventDefault()
    $form = $(e.currentTarget)
    credentials =
      name: $form.find("input[name='name']").val()
      password: $form.find("input[name='password']").val()
    conn.login(credentials, @onLoginSuccess, @onLoginFailure)

  register: (e)->
    e.preventDefault()
    Backbone.history.navigate('register', true)

  onLoginSuccess: (data)->
    State.Session = new App.Models.Session data
    State.Session.save()
    $.ajaxSetup headers: {"X-Authentication": State.Session.id}
    Backbone.history.navigate('home', true)

  onLoginFailure: (response)->
    alert response

  render: ->
    @$el.html @template()
    @
