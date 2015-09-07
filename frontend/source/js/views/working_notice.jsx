(function() {
  "use strict";

  App.Views.WorkingNotice = React.createClass({
    getInitialState: function() {
      return {
        display: "none"
      }
    },

    componentDidMount: function() {
      conn.firstAjaxStarted.add(function() {
        this.setState({display: ""})
      }.bind(this))

      conn.lastAjaxFinished.add(function() {
        this.setState({display: "none"})
      }.bind(this))
    },

    render: function() {
      return (
        <div id="working_notice" style={{display: this.state.display}}>
          <div>Working...</div>
        </div>
      )
    },
  })
})()
