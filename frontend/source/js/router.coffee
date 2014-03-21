class App.Router extends Backbone.Router
  routes:
    "login" : "login"
    "home"  : "home"
    "register"  : "register"
    "feeds"  : "feeds"
    "*path" : "login"

  login: ->
    @changePage App.Views.LoginPage

  home: ->
    unless State.Session.isAuthenticated()
      @navigate 'login'
      return

    @changePage App.Views.HomePage

  register: ->
    registrationService = new App.Services.Registration
    @changePage App.Views.RegisterPage, registrationService: registrationService

  feeds: ->
    unless State.Session.isAuthenticated()
      @navigate 'login'
      return

    @changePage App.Views.FeedsPage

  changePage: (pageClass, options)->
    @currentPage.remove() if @currentPage
    @currentPage = new pageClass(options)
    $("#view").empty().append(@currentPage.render().$el)

