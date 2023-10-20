import { Component } from "react";
import { KotsPageTitle } from "@components/Head";
import GitOpsDeploymentManager from "../../features/Gitops/GitOpsDeploymentManager";
import { GitOpsProvider } from "../../features/Gitops/context";

import "../../scss/components/watches/WatchedApps.scss";
import "../../scss/components/watches/WatchCard.scss";

interface Props {
  appName: string;
}
export class GitOps extends Component<Props> {
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
