import React from 'react'
import Header from './Header.jsx'
import WorkingNotice from './WorkingNotice.jsx'

export default class App extends React.Component {
  constructor(props, context) {
    super(props, context)
    window.router = context.router
  }

  render() {
    return (
      <div>
        <WorkingNotice />
        <Header />
        {this.props.children}
      </div>
    )
  }
}
