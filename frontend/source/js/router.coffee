class App.Router
  routes:
    "login" : "login"
    "home"  : "home"
    "register"  : "register"
    "feeds"  : "feeds"

  login: ->
    @changePage App.Views.LoginPage

  home: ->
    unless State.Session.isAuthenticated()
      @navigate 'login'
      return

    @changePage App.Views.HomePage

  register: ->
    @changePage App.Views.RegisterPage

  feeds: ->
    unless State.Session.isAuthenticated()
      @navigate 'login'
      return

    @changePage App.Views.FeedsPage

  changePage: (pageClass, options)->
    @currentPage.remove() if @currentPage
    @currentPage = new pageClass(options)
    view = document.getElementById("view")
    view.innerHTML = ""
    view.appendChild(@currentPage.render())

  start: ->
    window.addEventListener("hashchange",
      => @change(),
      false)
    @change()

  change: ->
    handler = @routes[window.location.hash.slice(1)]
    if !handler
      @navigate("login")
      return
    this[handler]()

  navigate: (route)->
    window.location.hash = "#" + route
