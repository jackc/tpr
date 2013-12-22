class App.Router extends Backbone.Router
  routes:
    "login" : "login"
    "home"  : "home"
    "register"  : "register"
    "subscribe"  : "subscribe"
    "feeds"  : "feeds"
    "import"  : "import"
    "*path" : "login"

  login: ->
    authenticationService = new App.Services.Authentication
    @changePage App.Views.LoginPage, authenticationService: authenticationService

  home: ->
    unless State.Session.isAuthenticated()
      Backbone.history.navigate('login', true)
      return

    @changePage App.Views.HomePage

  subscribe: ->
    unless State.Session.isAuthenticated()
      Backbone.history.navigate('login', true)
      return

    @changePage App.Views.SubscribePage

  import: ->
    unless State.Session.isAuthenticated()
      Backbone.history.navigate('login', true)
      return

    @changePage App.Views.ImportPage

  register: ->
    registrationService = new App.Services.Registration
    @changePage App.Views.RegisterPage, registrationService: registrationService

  feeds: ->
    unless State.Session.isAuthenticated()
      Backbone.history.navigate('login', true)
      return

    @changePage App.Views.FeedsPage

  changePage: (pageClass, options)->
    @currentPage.remove() if @currentPage
    @currentPage = new pageClass(options)
    $("#view").empty().append(@currentPage.render().$el)

