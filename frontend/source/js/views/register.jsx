(function() {
  "use strict"

  App.Views.RegisterPage = React.createClass({
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
        <div className="register">
          <header>
            <h1>The Pithy Reader</h1>
          </header>
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
            <a href="#login" className="login">Login</a>
          </form>
        </div>
      );
    },

    register: function(e) {
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
    },

    onRegistrationSuccess: function(data) {
      State.Session = new App.Models.Session(data)
      State.Session.save()
      window.router.navigate("home")
    },

    onRegistrationFailure: function(response) {
      alert(response)
    }
  })
})()
