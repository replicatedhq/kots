import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import { compose, withApollo } from "react-apollo";
import "../../scss/components/watches/WatchDetailPage.scss";

class AppGitopsSettings extends Component {

  render() {
    return (
      <div>
      </div>
    )
  }
}

export default compose(
  withRouter,
  withApollo,
)(AppGitopsSettings);
