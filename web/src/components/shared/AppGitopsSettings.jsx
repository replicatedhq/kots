import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import { compose, withApollo } from "react-apollo";
import "../../scss/components/watches/WatchDetailPage.scss";

class AppGitopsSettings extends Component {

  render() {
    const appTitle = "test";

    return (
      <div className="u-marginTop--15">
        <h2 className="u-fontSize--larger u-fontWeight--bold u-color--tuna">Analyze {appTitle} for support</h2>
        <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--medium u-marginTop--5">
          To diagnose any problems with the application, click the button below to get started. This will
          collect logs, resources and other data from the running application and analyze them against a set of known
          problems in {appTitle}. Logs, cluster info and other data will not leave your cluster.
        </p>
      </div>
    )
  }
}

export default compose(
  withRouter,
  withApollo,
)(AppGitopsSettings);
