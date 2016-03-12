import React from 'react'
import { Link } from 'react-router'
import {conn} from '../connection.js'
import Session from '../session.js'

export default class RegisterPage extends React.Component {
  constructor(props, context) {
    super(props, context)
    this.state = {
      name: null,
      password: null
    }

    this.handleChange = this.handleChange.bind(this)
    this.register = this.register.bind(this)
    this.onRegistrationSuccess = this.onRegistrationSuccess.bind(this)
    this.onRegistrationFailure = this.onRegistrationFailure.bind(this)
  }

  handleChange(name, event) {
    var h = {}
    h[name] = event.target.value
    this.setState(h)
  }

  render() {
    return (
      <div className="register">
        <form onSubmit={this.register}>
          <dl>
            <dt>
              <label htmlFor="name">User name</label>
            </dt>
            <dd><input type="text" id="name" value={this.state.name} onChange={this.handleChange.bind(null, "name")} /></dd>

            <dt>
              <label htmlFor="email">Email (optional)</label>
            </dt>
            <dd><input type="email" id="email" value={this.state.email} onChange={this.handleChange.bind(null, "email")} /></dd>

            <dt>
              <label htmlFor="password">Password</label>
            </dt>
            <dd><input type="password" id="password" value={this.state.password} onChange={this.handleChange.bind(null, "password")} /></dd>

            <dt>
              <label htmlFor="passwordConfirmation">Password Confirmation</label>
            </dt>
            <dd><input type="password" id="passwordConfirmation" value={this.state.passwordConfirmation} onChange={this.handleChange.bind(null, "passwordConfirmation")} /></dd>
          </dl>

          <input type="submit" value="Register" />
          <Link to="/login" className="login">Login</Link>
        </form>
      </div>
    )
  }

  register(e) {
    e.preventDefault()

    if(this.state.password != this.state.passwordConfirmation) {
      alert("Password and confirmation must match.")
      return
    }

    var registration = {
      name: this.state.name,
      email: this.state.email,
      password: this.state.password
    }
    conn.register(registration, {
      succeeded: this.onRegistrationSuccess,
      failed: function(_, response) { this.onRegistrationFailure(response.responseText) }.bind(this)
    })
  }

  onRegistrationSuccess(data) {
    Session.id = data.sessionID
    Session.name = data.name
    this.context.router.push('home')
  }

  onRegistrationFailure(response) {
    alert(response)
  }
}
