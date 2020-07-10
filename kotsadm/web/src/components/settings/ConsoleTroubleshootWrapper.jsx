import React, { Component } from "react";
import { withRouter } from "react-router-dom";

class ConsoleTroubleshootWrapper extends Component {
  render() {
    return (
      <div>
        <p>Console troubleshoot wrapper</p>
      </div>
    )
  }
}

export default withRouter(ConsoleTroubleshootWrapper);