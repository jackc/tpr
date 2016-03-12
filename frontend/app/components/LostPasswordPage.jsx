import React from 'react';
import {conn} from '../connection.js'
import Session from '../session.js'

export default class LostPasswordPage extends React.Component {
  constructor(props, context) {
    super(props, context)
    this.state = {
      email: null
    }

    this.handleChange = this.handleChange.bind(this)
    this.requestPasswordReset = this.requestPasswordReset.bind(this)
    this.onRequestPasswordResetSuccess = this.onRequestPasswordResetSuccess.bind(this)
    this.onRequestPasswordResetFailure = this.onRequestPasswordResetFailure.bind(this)
  }

  handleChange(name, event) {
    var h = {}
    h[name] = event.target.value
    this.setState(h)
  }

  render() {
    return (
      <div className="lostPassword">
        <form onSubmit={this.requestPasswordReset}>
          <dl>
            <dt>
              <label htmlFor="email">Email</label>
            </dt>
            <dd><input type="email" id="email" autofocus value={this.state.email} onChange={this.handleChange.bind(null, "email")} /></dd>
          </dl>

          <input type="submit" value="Reset Password" />
        </form>
      </div>
    )
  }

  requestPasswordReset(e) {
    e.preventDefault()
    var form = e.currentTarget
    conn.requestPasswordReset(this.state.email, {
      succeeded: this.onRequestPasswordResetSuccess,
      failed: function(_, response) { this.onRequestPasswordResetFailure(response.responseText) }.bind(this)
    })
  }

  onRequestPasswordResetSuccess(data) {
    alert("Please check your email for reset information")
  }

  onRequestPasswordResetFailure(response) {
    alert("Failure resetting password")
  }
}
