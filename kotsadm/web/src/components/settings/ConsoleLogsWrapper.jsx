import React, { Component } from "react";
import { withRouter } from "react-router-dom";

class ConsoleLogsWrapper extends Component {
  render() {
    return (
      <div>
        <p>Console logs wrapper</p>
      </div>
    )
  }
}

export default withRouter(ConsoleLogsWrapper);