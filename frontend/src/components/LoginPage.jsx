import React, { useState } from 'react'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import {conn} from '../connection.js'
import Session from '../session.js'

export default function LoginPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const [formData, setFormData] = useState({
    name: '',
    password: ''
  })

  const handleChange = (name, e) => {
    setFormData(prev => ({...prev, [name]: e.target.value}))
  }

  const login = (e) => {
    e.preventDefault()

    const credentials = {
      name: formData.name,
      password: formData.password
    }

    conn.login(credentials, {
      succeeded: (data) => {
        Session.id = data.sessionID
        Session.name = data.name
        const from = location.state?.from?.pathname || '/home'
        navigate(from, { replace: true })
      },
      failed: (_, response) => {
        alert(response.responseText)
      }
    })
  }

  return (
    <div className="login">
      <form onSubmit={login}>
        <dl>
          <dt>
            <label htmlFor="name">User name</label>
          </dt>
          <dd>
            <input
              type="text"
              id="name"
              autoFocus
              value={formData.name}
              onChange={(e) => handleChange("name", e)}
            />
          </dd>

          <dt>
            <label htmlFor="password">Password</label>
          </dt>
          <dd>
            <input
              type="password"
              id="password"
              value={formData.password}
              onChange={(e) => handleChange("password", e)}
            />
          </dd>
        </dl>

        <input type="submit" value="Login" />
        {' '}
        <Link to="/register" className="register">Create an account</Link>
        {' '}
        <Link to="/lostPassword" className="lostPassword">Lost password</Link>
      </form>
    </div>
  )
}
