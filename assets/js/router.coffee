class App.Router extends Backbone.Router
  routes:
    "login" : "login"
    "home"  : "home"
    "register"  : "register"
    "subscribe"  : "subscribe"
    "feeds"  : "feeds"
    "*path" : "login"

  login: ->
    authenticationService = new App.Services.Authentication
    @renderPage App.Views.LoginPage, authenticationService: authenticationService

  home: ->
    unless State.Session.isAuthenticated()
      Backbone.history.navigate('login', true)
      return

    @renderPage App.Views.HomePage

  subscribe: ->
    unless State.Session.isAuthenticated()
      Backbone.history.navigate('login', true)
      return

    @renderPage App.Views.SubscribePage

  register: ->
    registrationService = new App.Services.Registration
    @renderPage App.Views.RegisterPage, registrationService: registrationService

  feeds: ->
    unless State.Session.isAuthenticated()
      Backbone.history.navigate('login', true)
      return

    @renderPage App.Views.FeedsPage

  renderPage: (pageClass, options)->
    page = new pageClass(options)
    $("#view").empty().append(page.render().$el)

