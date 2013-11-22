class App.Views.SubscribePage extends Backbone.View
  template: _.template($("#subscribe_page_template").html())

  events:
    'submit form' : 'subscribe'

  initialize: ->
    @header = new App.Views.LoggedInHeader

  render: ->
    @$el.html @header.render().$el
    @$el.append @template()
    @

  subscribe: (e)->
    e.preventDefault()

    data =
      url: @$("input[name=url]").val()

    $.ajax(
      url: "/api/subscriptions?sessionID=#{State.Session.id}",
      type: "POST",
      data: JSON.stringify(data)
      contentType: "application/json"
    ).success ->
      Backbone.history.navigate('home', true)