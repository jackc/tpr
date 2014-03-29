class App.Views.RegisterPage
  template: JST["templates/register_page"]
  className: 'register'

  constructor: ->
    @$el = $("<div></div>")
    @$el.addClass @className
    @$el.on "submit", "form", (e)=> @register(e)
    @$el.on "click", "a.login", (e) => @login(e)

  register: (e)->
    e.preventDefault()
    $form = $(e.currentTarget)
    registration =
      name: $form.find("input[name='name']").val()
      password: $form.find("input[name='password']").val()
      passwordConfirmation: $form.find("input[name='passwordConfirmation']").val()
    conn.register(registration,
      (data)=> @onRegistrationSuccess(data),
      (response)=> @onRegistrationFailure(response))

  onRegistrationSuccess: (data)->
    State.Session = new App.Models.Session data
    State.Session.save()
    $.ajaxSetup headers: {"X-Authentication": State.Session.id}
    window.router.navigate('home')

  onRegistrationFailure: (response)->
    alert response

  login: (e)->
    e.preventDefault()
    window.router.navigate('login')

  render: ->
    @$el.html @template()
    @

  remove: ->
    @
