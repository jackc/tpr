(function() {
  "use strict"

  App.Views.ResetPasswordPage = React.createClass({
    getInitialState: function() {
      return {
        token: window.location.hash.split("=")[1],
        password: null
      }
    },

    handleChange: function(name, event) {
      var h = {}
      h[name] = event.target.value
      this.setState(h)
    },

    render: function() {
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
      );
    },

    resetPassword: function(e) {
      e.preventDefault()
      var reset = {
        "token": this.state.token,
        "password": this.state.password
      }
      conn.resetPassword(reset, {
        succeeded: this.onResetPasswordSuccess,
        failed: function(_, response) { this.onResetPasswordFailure(response.responseText) }.bind(this)
      })
    },

    onResetPasswordSuccess: function(data) {
      alert("Successfully reset password")
      State.Session = new App.Models.Session(data)
      State.Session.save()
      window.router.navigate('home')
    },

    onResetPasswordFailure: function(response) {
      alert("Failure resetting password")
    }
  })
})()
