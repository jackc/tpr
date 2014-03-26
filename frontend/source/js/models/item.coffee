class App.Models.Item
  markRead: ->
    return if @isRead
    conn.markItemRead(@id)
    @isRead = true
