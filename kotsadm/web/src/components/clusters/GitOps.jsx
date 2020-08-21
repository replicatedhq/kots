import * as React from "react";
import PropTypes from "prop-types";
import Helmet from "react-helmet";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import GitOpsDeploymentManager from "../gitops/GitOpsDeploymentManager";
import { Utilities } from "../../utilities/utilities";

import "../../scss/components/watches/WatchedApps.scss";
import "../../scss/components/watches/WatchCard.scss";

export class GitOps extends React.Component {
  static propTypes = {
    history: PropTypes.object.isRequired,
  };

  componentDidMount() {
    this.getClusters();
  }

  render() {
    const hasClusters = this.state && this.state.clusters && this.state.clusters?.length && this.state.clusters[0];

    return (
      <div className="ClusterDashboard--wrapper container flex-column flex1 u-overflow--auto">
        <Helmet>
          <title>GitOps deployments</title>
        </Helmet>
        <div className="flex-column flex1">
          {hasClusters && 
            <div className="flex-column flex-1-auto u-paddingBottom--20 u-paddingTop--30 u-marginTop--10 u-overflow--auto">
              <GitOpsDeploymentManager appName={this.props.appName} />
            </div>
          }
        </div>
      </div>
    );
  }

  getClusters = async () => {
    fetch(`${window.env.API_ENDPOINT}/clusters/list`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    }).then(async (response) => {
      const data = await response.json();
      this.setState({
          clusters: data,
      })
      return data;
    }).catch((error) => {
      console.log(error);
    });
  }
}

export default compose(
  withRouter,
  withApollo,
)(GitOps);
