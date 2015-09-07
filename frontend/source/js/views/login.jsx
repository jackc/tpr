(function() {
  "use strict"

  App.Views.LoginPage = React.createClass({
    getInitialState: function() {
      return {
        name: null,
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
        <div className="login">
          <header>
            <h1>The Pithy Reader</h1>
          </header>
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
            <a href="#register" className="register">Create an account</a>
            {' '}
            <a href="#lostPassword" className="lostPassword">Lost password</a>
          </form>
        </div>
      );
    },

    login: function(e) {
      e.preventDefault()
      var form = e.currentTarget
      var credentials = {
        name: form.elements.name.value,
        password: form.elements.password.value
      }
      conn.login(credentials, {
        succeeded: this.onLoginSuccess,
        failed: function(_, response) { this.onLoginFailure(response.responseText) }.bind(this)
      })
    },

    onLoginSuccess: function(data) {
      State.Session = new App.Models.Session(data)
      State.Session.save()
      window.router.navigate('home')
    },

    onLoginFailure: function(response) {
      alert(response)
    }
  })
})()
