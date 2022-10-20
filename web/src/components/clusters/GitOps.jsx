import * as React from "react";
import PropTypes from "prop-types";
import { KotsPageTitle } from "@components/Head";
import { withRouter } from "react-router-dom";
import GitOpsDeploymentManager from "../../features/Gitops/GitOpsDeploymentManager";
import { GitOpsProvider } from "../../features/Gitops/context";

import "../../scss/components/watches/WatchedApps.scss";
import "../../scss/components/watches/WatchCard.scss";

export class GitOps extends React.Component {
  render() {
    return (
      <GitOpsProvider>
        <div className="ClusterDashboard--wrapper container flex-column flex1 u-overflow--auto">
          <KotsPageTitle pageName="GitOps Deployments" />
          <div className="flex-column flex1">
            <div className="flex-column flex-1-auto u-paddingBottom--20 u-paddingTop--30 u-marginTop--10 u-overflow--auto">
              <GitOpsDeploymentManager appName={this.props.appName} />
            </div>
          </div>
        </div>
      </GitOpsProvider>
    );
  }
}

export default GitOps;
