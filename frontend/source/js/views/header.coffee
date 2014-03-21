class App.Views.LoggedInHeader extends App.Views.Base
  template: _.template($("#logged_in_header_template").html())
  tagName: 'header'

  events:
    'click a.logout' : 'logout'

  render: ->
    @$el.html @template()
    @

  logout: (e)->
    e.preventDefault()
    conn.logout()
    State.Session.clear()
    $.ajaxSetup headers: {}
    Backbone.history.navigate('login', true)