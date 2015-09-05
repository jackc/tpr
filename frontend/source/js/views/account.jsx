(function() {
  "use strict"

  App.Views.AccountPage = React.createClass({
    getInitialState: function() {
      return {
        email: '',
        existingPassword: '',
        newPassword: '',
        passwordConfirmation: ''
      };
    },

    handleChange: function(name, event) {
      var h = {}
      h[name] = event.target.value
      this.setState(h)
    },

    componentDidMount: function() {
      this.fetch()
    },

    fetch: function() {
      conn.getAccount({
        succeeded: function(data) {
          this.setState(data)
        }.bind(this)
      })
    },

    render: function() {
      return (
        <div class="account">
          <App.Views.LoggedInHeaderR />
          <form onSubmit={this.update}>
            <dl>
              <dt>
                <label for="email">Email</label>
              </dt>
              <dd>
                <input type="email" name="email" id="email" value={this.state.email} onChange={this.handleChange.bind(null, "email")} />
              </dd>
              <dt>
                <label for="existingPassword">Existing Password</label>
              </dt>
              <dd>
                <input type="password" name="existingPassword" id="existingPassword" value={this.state.existingPassword} onChange={this.handleChange.bind(null, "existingPassword")} />
              </dd>
              <dt>
                <label for="newPassword">New Password</label>
              </dt>
              <dd>
                <input type="password" name="newPassword" id="newPassword" value={this.state.newPassword} onChange={this.handleChange.bind(null, "newPassword")} />
              </dd>
              <dt>
                <label for="passwordConfirmation">Password Confirmation</label>
              </dt>
              <dd>
                <input type="password" name="passwordConfirmation" id="passwordConfirmation" value={this.state.passwordConfirmation} onChange={this.handleChange.bind(null, "passwordConfirmation")} />
              </dd>
            </dl>

            <input type="submit" value="Update" />
          </form>
        </div>
      );
    },

    update: function(e) {
      e.preventDefault()

      if(this.state.newPassword != this.state.passwordConfirmation) {
        alert("New password and confirmation must match.")
        return
      }

      var update = {}
      update.email = this.state.email
      update.existingPassword = this.state.existingPassword
      update.newPassword = this.state.newPassword

      conn.updateAccount(update, {
        succeeded: function() {
          this.setState({
            existingPassword: "",
            newPassword: "",
            passwordConfirmation: ""
          })
          alert("Update succeeded")
        }.bind(this),
        failed: function(data) {
          alert(data)
        }.bind(this)
      })
    }
  })
})()
