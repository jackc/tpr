class App.Models.Item extends Backbone.Model
  markRead: ->
    return if @isRead
    $.ajax(url: "/api/items/unread/#{@get("id")}?sessionID=#{State.Session.id}", method: "DELETE")
    @isRead = true
