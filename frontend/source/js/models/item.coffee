class App.Models.Item extends Backbone.Model
  markRead: ->
    return if @isRead
    $.ajax(url: "/api/items/unread/#{@get("id")}", method: "DELETE")
    @isRead = true
