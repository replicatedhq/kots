import React, { Component } from "react";
import { withRouter } from "react-router-dom";

class ConsoleAuthentication extends Component {
  render() {
    return (
      <div>
        <p>Console authentication</p>
      </div>
    )
  }
}

export default withRouter(ConsoleAuthentication);