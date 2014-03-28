class App.Views.LoggedInHeader
  template: _.template($("#logged_in_header_template").html())
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
    Backbone.history.navigate('login', true)

  remove: ->
    @
