import React from 'react'
import PropTypes from 'prop-types'
import { Link } from 'react-router'
import {conn} from '../connection.js'
import Session from '../session.js'

export default class LoginPage extends React.Component {
  constructor(props, context) {
    super(props, context)
    this.state = {
      name: null,
      password: null
    }

    this.handleChange = this.handleChange.bind(this)
    this.login = this.login.bind(this)
    this.onLoginSuccess = this.onLoginSuccess.bind(this)
    this.onLoginFailure = this.onLoginFailure.bind(this)
  }

  handleChange(name, event) {
    var h = {}
    h[name] = event.target.value
    this.setState(h)
  }

  render() {
    return (
      <div className="login">
        <form onSubmit={this.login}>
          <dl>
            <dt>
              <label htmlFor="name">User name</label>
            </dt>
            <dd><input type="text" id="name" autofocus value={this.state.name} onChange={this.handleChange.bind(null, "name")} /></dd>

            <dt>
              <label htmlFor="password">Password</label>
            </dt>
            <dd><input type="password" id="password" value={this.state.password} onChange={this.handleChange.bind(null, "password")} /></dd>
          </dl>

          <input type="submit" value="Login" />
          {' '}

          <Link to="/register" className="register">Create an account</Link>
          {' '}
          <Link to="/lostPassword" className="lostPassword">Lost password</Link>
        </form>
      </div>
    );
  }

  login(e) {
    e.preventDefault()

    var credentials = {
      name: this.state.name,
      password: this.state.password
    }
    conn.login(credentials, {
      succeeded: this.onLoginSuccess,
      failed: function(_, response) { this.onLoginFailure(response.responseText) }.bind(this)
    })
  }

  onLoginSuccess(data) {
    Session.id = data.sessionID
    Session.name = data.name
    this.context.router.push('home')
  }

  onLoginFailure(response) {
    alert(response)
  }
}

LoginPage.contextTypes = {
  router: PropTypes.object
}
