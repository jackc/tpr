(function() {
  "use strict"

  App.Views.LostPasswordPage = React.createClass({
    getInitialState: function() {
      return {
        email: null
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
      );
    },

    requestPasswordReset: function(e) {
      e.preventDefault()
      var form = e.currentTarget
      conn.requestPasswordReset(this.state.email, {
        succeeded: this.onRequestPasswordResetSuccess,
        failed: function(_, response) { this.onRequestPasswordResetFailure(response.responseText) }.bind(this)
      })
    },

    onRequestPasswordResetSuccess: function(data) {
      alert("Please check your email for reset information")
    },

    onRequestPasswordResetFailure: function(response) {
      alert("Failure resetting password")
    }
  })
})()
