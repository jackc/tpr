class App.Views.LoginPage
  template: _.template($("#login_page_template").html())
  className: 'login'

  constructor: ->
    @$el = $("<div></div>")
    @$el.addClass @className
    @$el.on "submit", "form", (e)=> @login(e)
    @$el.on "click", "a.register", (e) =>@register(e)

  login: (e)->
    e.preventDefault()
    $form = $(e.currentTarget)
    credentials =
      name: $form.find("input[name='name']").val()
      password: $form.find("input[name='password']").val()
    conn.login(credentials,
      (data)=> @onLoginSuccess(data),
      (response)=> @onLoginFailure(response))

  register: (e)->
    e.preventDefault()
    window.router.navigate('register')

  onLoginSuccess: (data)->
    State.Session = new App.Models.Session data
    State.Session.save()
    $.ajaxSetup headers: {"X-Authentication": State.Session.id}
    window.router.navigate('home')

  onLoginFailure: (response)->
    alert response

  render: ->
    @$el.html @template()
    @

  remove: ->
    @
