import React from 'react';
import {conn} from '../connection.js'
import Session from '../session.js'

export default class ResetPasswordPage extends React.Component {
  constructor(props, context) {
    super(props, context)
    this.state = {
      token: window.location.hash.split("=")[1],
      password: null
    }

    this.handleChange = this.handleChange.bind(this)
    this.resetPassword = this.resetPassword.bind(this)
    this.onResetPasswordSuccess = this.onResetPasswordSuccess.bind(this)
    this.onResetPasswordFailure = this.onResetPasswordFailure.bind(this)
  }

  handleChange(name, event) {
    var h = {}
    h[name] = event.target.value
    this.setState(h)
  }

  render() {
    return (
      <div className="lostPassword">
        <header>
          <h1>The Pithy Reader</h1>
        </header>
        <form onSubmit={this.resetPassword}>
          <dl>
            <dt>
              <label htmlFor="password">Password</label>
            </dt>
            <dd><input type="password" id="password" autofocus value={this.state.password} onChange={this.handleChange.bind(null, "password")} /></dd>
          </dl>

          <input type="submit" value="Reset Password" />
        </form>
      </div>
    )
  }

  resetPassword(e) {
    e.preventDefault()
    var reset = {
      "token": this.state.token,
      "password": this.state.password
    }
    conn.resetPassword(reset, {
      succeeded: this.onResetPasswordSuccess,
      failed: function(_, response) { this.onResetPasswordFailure(response.responseText) }.bind(this)
    })
  }

  onResetPasswordSuccess(data) {
    alert("Successfully reset password")
    Session.id = data.sessionID
    Session.name = data.name
    this.context.router.push('home')
  }

  onResetPasswordFailure(response) {
    alert("Failure resetting password")
  }
}

