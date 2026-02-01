import React from 'react'
import { Link, useNavigate } from 'react-router-dom'
import Session from '../session.js'
import {conn} from '../connection.js'

export default function Header() {
  const navigate = useNavigate()

  const logout = (e) => {
    e.preventDefault()
    conn.logout()
    Session.clear()
    navigate('/login')
  }

  const renderLoggedInNav = () => (
    <nav>
      <Link to="/home">Home</Link>
      {' '}
      <Link to="/archive">Archive</Link>
      {' '}
      <Link to="/feeds">Feeds</Link>
      {' '}
      <Link to="/account">Account</Link>
      {' '}
      <a href="#" onClick={logout}>Logout</a>
    </nav>
  )

  return (
    <header>
      <h1>The Pithy Reader</h1>
      {Session.isAuthenticated() ? renderLoggedInNav() : null}
    </header>
  )
}
