class App.Views.LoggedInHeader extends App.Views.Base
  template: JST["templates/logged_in_header"]
  tagName: 'header'

  listen: ->
    logoutLink = @el.querySelector("a.logout")
    logoutLink.addEventListener("click", (e)=> @logout(e))

  render: ->
    @el.innerHTML = @template()
    @listen()
    @el

  logout: (e)->
    e.preventDefault()
    conn.logout()
    State.Session.clear()
    window.router.navigate('login')
