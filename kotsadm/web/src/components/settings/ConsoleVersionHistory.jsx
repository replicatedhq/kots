import React, { Component } from "react";
import { withRouter } from "react-router-dom";

class ConsoleVersionHistory extends Component {
  render() {
    return (
      <div>
        <p>Console version history</p>
      </div>
    )
  }
}

export default withRouter(ConsoleVersionHistory);