class App.Models.Session
  constructor: (options)->
    if options
      @name = options.name
      @id = options.sessionID

  load: ->
    serializedSession = localStorage.getItem('session')
    if serializedSession
      session = JSON.parse serializedSession
      @name = session.name
      @id = session.id

  save: ->
    localStorage.setItem('session', JSON.stringify(name: @name, id: @id))

  clear: ->
    localStorage.clear()
    State.Session = new App.Models.Session

  isAuthenticated: ->
    !!@id
