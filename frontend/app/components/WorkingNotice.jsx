import React from 'react';
import {conn} from '../connection.js'
import Session from '../session.js'
import{toTPRString} from '../date.js'

export default class WorkingNotice extends React.Component {
  constructor(props, context) {
    super(props, context)
    this.state = {
        display: "none"
      }
  }

  componentDidMount() {
    conn.firstAjaxStarted.add(function() {
      this.setState({display: ""})
    }.bind(this))

    conn.lastAjaxFinished.add(function() {
      this.setState({display: "none"})
    }.bind(this))
  }

  render() {
    return (
      <div id="working_notice" style={{display: this.state.display}}>
        <div>Working...</div>
      </div>
    )
  }
}
