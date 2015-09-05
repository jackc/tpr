(function() {
  "use strict"

  App.Views.LoggedInHeaderR = React.createClass({
    render: function() {
      return (
        <header>
          <h1>The Pithy Reader</h1>
          <nav>
            <a href="#home">Home</a>
            {' '}
            <a href="#feeds">Feeds</a>
            {' '}
            <a href="#account">Account</a>
            {' '}
            <a href="#" onClick={this.logout}>Logout</a>
          </nav>
        </header>
      )
    },

    logout: function(e) {
      e.preventDefault()
      conn.logout()
      State.Session.clear()
      window.router.navigate("login")
    }
  })
})()
