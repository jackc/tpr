import React, { useState, useEffect } from 'react'
import {conn} from '../connection.js'

export default function AccountPage() {
  const [formData, setFormData] = useState({
    email: '',
    existingPassword: '',
    newPassword: '',
    passwordConfirmation: ''
  })

  useEffect(() => {
    conn.getAccount({
      succeeded: (data) => {
        setFormData(prev => ({...prev, email: data.email}))
      }
    })
  }, [])

  const handleChange = (name, e) => {
    setFormData(prev => ({...prev, [name]: e.target.value}))
  }

  const update = (e) => {
    e.preventDefault()

    if(formData.newPassword !== formData.passwordConfirmation) {
      alert("New password and confirmation must match.")
      return
    }

    const updateData = {
      email: formData.email,
      existingPassword: formData.existingPassword,
      newPassword: formData.newPassword
    }

    conn.updateAccount(updateData, {
      succeeded: () => {
        setFormData(prev => ({
          ...prev,
          existingPassword: "",
          newPassword: "",
          passwordConfirmation: ""
        }))
        alert("Update succeeded")
      },
      failed: (data) => {
        alert(data)
      }
    })
  }

  return (
    <div className="account">
      <form onSubmit={update}>
        <dl>
          <dt>
            <label htmlFor="email">Email</label>
          </dt>
          <dd>
            <input
              type="email"
              name="email"
              id="email"
              value={formData.email}
              onChange={(e) => handleChange("email", e)}
            />
          </dd>
          <dt>
            <label htmlFor="existingPassword">Existing Password</label>
          </dt>
          <dd>
            <input
              type="password"
              name="existingPassword"
              id="existingPassword"
              value={formData.existingPassword}
              onChange={(e) => handleChange("existingPassword", e)}
            />
          </dd>
          <dt>
            <label htmlFor="newPassword">New Password</label>
          </dt>
          <dd>
            <input
              type="password"
              name="newPassword"
              id="newPassword"
              value={formData.newPassword}
              onChange={(e) => handleChange("newPassword", e)}
            />
          </dd>
          <dt>
            <label htmlFor="passwordConfirmation">Password Confirmation</label>
          </dt>
          <dd>
            <input
              type="password"
              name="passwordConfirmation"
              id="passwordConfirmation"
              value={formData.passwordConfirmation}
              onChange={(e) => handleChange("passwordConfirmation", e)}
            />
          </dd>
        </dl>

        <input type="submit" value="Update" />
      </form>
    </div>
  )
}
