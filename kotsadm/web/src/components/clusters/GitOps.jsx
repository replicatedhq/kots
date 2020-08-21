import * as React from "react";
import PropTypes from "prop-types";
import Helmet from "react-helmet";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import GitOpsDeploymentManager from "../gitops/GitOpsDeploymentManager";

import "../../scss/components/watches/WatchedApps.scss";
import "../../scss/components/watches/WatchCard.scss";

export class GitOps extends React.Component {
  static propTypes = {
    history: PropTypes.object.isRequired,
  };

  render() {
    return (
      <div className="ClusterDashboard--wrapper container flex-column flex1 u-overflow--auto">
        <Helmet>
          <title>GitOps deployments</title>
        </Helmet>
        <div className="flex-column flex1">
          <div className="flex-column flex-1-auto u-paddingBottom--20 u-paddingTop--30 u-marginTop--10 u-overflow--auto">
            <GitOpsDeploymentManager appName={this.props.appName} />
          </div>
        </div>
      </div>
    );
  }
}

export default compose(
  withRouter,
  withApollo,
)(GitOps);
