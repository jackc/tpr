import React, { useState } from 'react'
import { Link, useNavigate, useLocation } from 'react-router-dom'
import {conn} from '../connection.js'
import Session from '../session.js'

export default function RegisterPage() {
  const navigate = useNavigate()
  const location = useLocation()
  const [formData, setFormData] = useState({
    name: '',
    email: '',
    password: '',
    passwordConfirmation: ''
  })

  const handleChange = (name, e) => {
    setFormData(prev => ({...prev, [name]: e.target.value}))
  }

  const register = (e) => {
    e.preventDefault()

    if(formData.password !== formData.passwordConfirmation) {
      alert("Password and confirmation must match.")
      return
    }

    const registration = {
      name: formData.name,
      email: formData.email,
      password: formData.password
    }

    conn.register(registration, {
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
    <div className="register">
      <form onSubmit={register}>
        <dl>
          <dt>
            <label htmlFor="name">User name</label>
          </dt>
          <dd>
            <input
              type="text"
              id="name"
              value={formData.name}
              onChange={(e) => handleChange("name", e)}
            />
          </dd>

          <dt>
            <label htmlFor="email">Email (optional)</label>
          </dt>
          <dd>
            <input
              type="email"
              id="email"
              value={formData.email}
              onChange={(e) => handleChange("email", e)}
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

          <dt>
            <label htmlFor="passwordConfirmation">Password Confirmation</label>
          </dt>
          <dd>
            <input
              type="password"
              id="passwordConfirmation"
              value={formData.passwordConfirmation}
              onChange={(e) => handleChange("passwordConfirmation", e)}
            />
          </dd>
        </dl>

        <input type="submit" value="Register" />
        <Link to="/login" className="login">Login</Link>
      </form>
    </div>
  )
}
