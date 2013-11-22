class App.Views.ImportPage extends Backbone.View
  template: _.template($("#import_page_template").html())

  events:
    "submit form" : "upload"

  initialize: ->
    @header = new App.Views.LoggedInHeader

  render: ->
    @$el.html @header.render().$el
    @$el.append @template()
    @

  upload: (e)->
    e.preventDefault()
    fd = new FormData(e.target)

    $.ajax({
      url: "/api/feeds/import?sessionID=#{State.Session.id}",
      type: "POST",
      data: fd,
      processData: false,
      contentType: false
    }).success ->
      alert 'import success'
