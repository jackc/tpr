import React from 'react'
import { Link } from 'react-router'
import Session from '../session.js'
import {conn} from '../connection.js'

export default class Header extends React.Component {
  constructor(props, context) {
    super(props, context)

    this.logout = this.logout.bind(this)
  }

  render() {
    return (
      <header>
        <h1>The Pithy Reader</h1>
        <nav>
          <Link to="/home">Home</Link>
          {' '}
          <Link to="/feeds">Feeds</Link>
          {' '}
          <Link to="/account">Account</Link>
          {' '}
          <a href="#" onClick={this.logout}>Logout</a>
        </nav>
      </header>
    )
  }

  logout(e) {
    e.preventDefault()
    conn.logout()
    Session.clear()
    this.context.router.push('login')
  }
}

Header.contextTypes = {
  router: React.PropTypes.object
}
