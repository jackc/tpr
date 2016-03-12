class Session {
  load() {
    const serializedSession = localStorage.getItem("session")
    if(serializedSession) {
      return JSON.parse(serializedSession)
    }
    return {}
  }

  save(session) {
    localStorage.setItem("session", JSON.stringify(session))
  }

  clear() {
    localStorage.clear()
  }

  isAuthenticated() {
    return !!this.id
  }

  get id() {
    return this.load().id
  }

  set id(val){
    const session = this.load()
    session.id = val
    this.save(session)
  }

  get name() {
    return this.load().name
  }

  set name(val){
    const session = this.load()
    session.name = val
    this.save(session)
  }
}

const session = new Session
export default session
