class App.Views.LoggedInHeader
  template: JST["templates/logged_in_header"]
  tagName: 'header'

  constructor: ->
    @$el = $("<#{@tagName}></#{@tagName}>")
    @$el.on "click", "a.logout", (e) => @logout(e)

  render: ->
    @$el.html @template()
    @

  logout: (e)->
    e.preventDefault()
    conn.logout()
    State.Session.clear()
    $.ajaxSetup headers: {}
    window.router.navigate('login')

  remove: ->
    @
