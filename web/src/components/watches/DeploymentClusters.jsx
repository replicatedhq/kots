import React from "react";
import classNames from "classnames";
import { withRouter } from "react-router-dom";
import Helmet from "react-helmet";
import { Utilities } from "@src/utilities/utilities";
import "../../scss/components/clusters/ClusterCard.scss";
import "../../scss/components/watches/WatchCard.scss";
import DownstreamCard from "./DownstreamCard";
import DownstreamKotsCard from "./DownstreamKotsCard";

class DeploymentClusters extends React.Component {

  state = {
    pendingUri: "",
    isDownloadingAssets: false
  }

  installLatestVersion = (watchId, sequence) => {
    if (this.props.installLatestVersion && typeof this.props.installLatestVersion === "function") {
      this.props.installLatestVersion(watchId, sequence);
    }
  }

  downloadAssetsForCluster = async (watchId) => {
    this.setState({ isDownloadingAssets: true });
    await Utilities.handleDownload(watchId);
    this.setState({ isDownloadingAssets: false });
  }

  render() {
    const {
      appDetailPage,
      childWatches,
      handleAddNewCluster,
      parentWatch,
      toggleDeleteDeploymentModal,
      title,
      kotsApp,
      handleViewFiles,
      displayDownloadCommand
    } = this.props;
    const { isDownloadingAssets } = this.state;
    const pageTitle = title || parentWatch.watchName;

    
    return (
      <div className={classNames("installed-watch-github flex-column u-paddingTop--20 u-width--full", {
        padded: !appDetailPage
      })}>
        <Helmet>
          <title>{`${pageTitle} Downstreams`}</title>
        </Helmet>
        {childWatches?.length ?
          <div className="flex-column">
            <div className="flex justifyContent--spaceBetween alignItems--center u-marginBottom--20">
              <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna">Downstream deployments</p>
              <button type="button" className="btn secondary" onClick={handleAddNewCluster}>Deploy a new downstream</button>
            </div>
            <div className="integrations u-overflow--auto flex flex1">
              {childWatches && childWatches.map((childWatch) => {
                if (kotsApp) {
                  return (
                    <DownstreamKotsCard
                      key={childWatch.cluster.id}
                      downstream={childWatch}
                      appSlug={this.props.match.params.slug}
                      viewFiles={handleViewFiles}
                      downloadAssetsForCluster={this.downloadAssetsForCluster}
                      displayDownloadCommand={displayDownloadCommand}
                      toggleDeleteDeploymentModal={toggleDeleteDeploymentModal}
                    />
                  )
                } else {
                  return (
                   <DownstreamCard
                    key={childWatch.id}
                    childWatch={childWatch}
                    installLatestVersion={this.installLatestVersion}
                    downloadAssetsForCluster={this.downloadAssetsForCluster}
                    toggleDeleteDeploymentModal={toggleDeleteDeploymentModal}
                    isDownloadingAssets={isDownloadingAssets}
                    parentWatch={parentWatch}
                  />
                  )
                }
              })
              }
            </div>
          </div>
        :
        <div className="flex-column flex1">
          <div className="EmptyState--wrapper flex-column flex1">
            <div className="EmptyState flex-column flex1 alignItems--center justifyContent--center">
              <div className="flex alignItems--center justifyContent--center">
                <span className="icon ship-complete-icon-gh"></span>
                <span className="deployment-or-text">OR</span>
                <span className="icon ship-medium-size"></span>
              </div>
              <div className="u-textAlign--center u-marginTop--10">
                <p className="u-fontSize--largest u-color--tuna u-lineHeight--medium u-fontWeight--bold u-marginBottom--10">Deploy to a cluster</p>
                <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--medium u-fontWeight--medium">{pageTitle} has been configured but still needs to be deployed. Select a cluster you would like to deploy {pageTitle} to.</p>
              </div>
              <div className="u-marginTop--20">
                <button className="btn secondary" onClick={handleAddNewCluster}>Add a deployment cluster</button>
              </div>
            </div>
          </div>
        </div>
        }
      </div>
    );
  }
}

export default withRouter(DeploymentClusters);
