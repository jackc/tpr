import React from 'react'
import Session from '../session.js'

export default (ProtectedComponent) => {
  return class AuthenticatedComponent extends React.Component {
    static willTransitionTo(transition) {
        if (!session.isAuthenticated()) {
          transition.redirect('login')
        }
    }

    constructor(props, context) {
      super(props, context)
    }

    render() {
      alert("bar")
      return (
        <ProtectedComponent {...this.props} />
      )
    }
  }
}
