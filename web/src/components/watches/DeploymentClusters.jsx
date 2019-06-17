import React from "react";
import classNames from "classnames";
import truncateMiddle from "truncate-middle";
import { Link } from "react-router-dom";
import { Utilities } from "@src/utilities/utilities";
import "../../scss/components/clusters/ClusterCard.scss";
import "../../scss/components/watches/WatchCard.scss";

export default class DeploymentClusters extends React.Component {

  state = {
    pendingUri: "",
    isDownloadingAssets: false
  }

  installLatestVersion = (watchId, sequence) => {
    this.props.installLatestVersion(watchId, sequence);
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
      parentClusterName,
      toggleDeleteDeploymentModal
    } = this.props;

    return (
      <div className={classNames("installed-watch-github flex-column u-width--full", {
        padded: !appDetailPage
      })}>
        {childWatches?.length ?
          <div className="flex-column">
            {appDetailPage ?
              <div className="flex justifyContent--spaceBetween alignItems--center u-marginBottom--20">
                <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna">Downstream deployments</p>
                <button type="button" className="btn secondary" onClick={handleAddNewCluster}>Deploy a new downstream</button>
              </div>
            :
              <div className="flex">
                <p className="uppercase-title">CLUSTERS</p>
                <span className="u-marginLeft--10 replicated-link u-fontSize--small" onClick={handleAddNewCluster}>Deploy to a new cluster</span>
              </div>
            }
            <div className="integrations u-overflow--auto flex flex1">
              {childWatches && childWatches.map((childWatch) => {
                const { cluster } = childWatch;
                const currentVersion = childWatch.currentVersion ? childWatch.currentVersion.title : "Unknown";
                const type = cluster && cluster.gitOpsRef ? "git" : "ship";
                const gitPath = cluster && cluster.gitOpsRef ? `${cluster.gitOpsRef.owner}/${cluster.gitOpsRef.repo}/${cluster.gitOpsRef.branch}${cluster.gitOpsRef.path}` : "";
                return (
                  <div key={childWatch.id} className="integration flex-column flex1 flex">
                    <div className="flex u-marginBottom--5">
                      <span className={`normal u-marginRight--5 icon clusterType ${type}`}></span>
                      <div className="flex1 justifyContent--center">
                        <div className="flex justifyContent--spaceBetween">
                          <p className="flex1 u-fontWeight--bold u-fontSize--large u-color--tundora">{cluster && cluster.title || "Downstream deployment"}</p>
                          <span className="flex-auto icon u-grayX-icon clickable" onClick={() => toggleDeleteDeploymentModal(childWatch, parentClusterName)}></span>
                        </div>
                        <p className="u-fontWeight--medium u-fontSize--small u-color--dustyGray u-marginTop--5">{type === "git" ? truncateMiddle(gitPath, 22, 22, "...") : "Deployed with Ship"}</p>
                        <Link to={`/watch/${childWatch.slug}/state`} className="replicated-link u-marginTop--5 u-fontSize--small u-lineHeight--normal">View state.json</Link>
                      </div>
                    </div>
                    <div className="u-marginTop--10">
                      <div className="flex flex1">
                        <h2 className="u-fontSize--jumbo2 alignSelf--center u-fontWeight--bold u-color--tuna">{currentVersion}</h2>
                        {currentVersion && childWatch.pendingVersions.length === 1 &&
                          <div className="flex-auto flex flex1 alignItems--center alignSelf--center">
                            <div className="icon exclamationMark-icon u-marginLeft--10"></div>
                            <p className="u-fontSize--normal u-color--orange u-fontWeight--medium u-marginLeft--5">One version behind</p>
                          </div>
                        }
                        {currentVersion && childWatch.pendingVersions.length >= 2 &&
                          <div className="flex-auto flex flex1 alignItems--center alignSelf--center">
                            <div className="icon exclamationMark-icon u-marginLeft--10"></div>
                            <p className="u-fontSize--normal u-color--orange u-fontWeight--medium u-marginLeft--5">Two or more versions behind</p>
                          </div>
                        }
                        {currentVersion && !childWatch.pendingVersions.length &&
                          <div className="flex-auto flex flex1 alignItems--center alignSelf--center">
                            <div className="icon checkmark-icon u-marginLeft--10"></div>
                            <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-marginLeft--5">Up to date</p>
                          </div>
                        }
                      </div>
                      <Link to={`/watch/${childWatch.slug}/history`} className="replicated-link u-fontSize--normal u-lineHeight--normal">See version history</Link>
                    </div>
                    {currentVersion && childWatch.pendingVersions.length >= 1 &&
                      <div className="flex justifyContent--spaceBetween alignItems--center u-marginTop--10">
                        {type === "git" ?
                          <a href={`https://github.com/${cluster.gitOpsRef.owner}/${cluster.gitOpsRef.repo}/pull/${childWatch.pendingVersions[0].pullrequestNumber}`} className="btn green secondary" target="_blank" rel="noopener noreferrer">Review PR to update application</a>
                        :
                          <div className="flex-column">
                            <button onClick={() => this.installLatestVersion(childWatch.id, childWatch.pendingVersions[0].sequence)} className="btn green secondary">Install latest version ({childWatch.pendingVersions[0].title})</button>
                          </div>
                        }
                      </div>
                    }
                    {currentVersion && !childWatch.pendingVersions.length &&
                      <div className="flex justifyContent--spaceBetween alignItems--center  u-marginTop--5">
                        <p className="u-fontWeight--medium u-fontSize--small u-color--dustyGray u-marginTop--5 u-lineHeight--more">When an update is available you will be able to deploy it from here.</p>
                      </div>
                    }
                    <div className="flex flex1 alignItems--flexEnd">
                      <div className="flex u-marginTop--20 u-borderTop--gray u-width--full">
                        <div className="flex1 flex card-action-wrapper u-cursor--pointer">
                          <span className="flex1 u-marginRight--5 u-color--astral card-action u-fontSize--small u-fontWeight--medium u-textAlign--center" onClick={() => { this.downloadAssetsForCluster(childWatch.id) }}>Download assets</span>
                        </div>
                        <div className="flex1 flex card-action-wrapper u-cursor--pointer">
                          <span onClick={this.props.preparingUpdate === childWatch.cluster.id ? () => { return } : () => this.props.onEditApplication(childWatch)} className="flex1 u-marginRight--5 u-color--astral card-action u-fontSize--small u-fontWeight--medium u-textAlign--center">{this.props.preparingUpdate === childWatch.cluster.id ? "Preparing" : "Edit downstream"}</span>
                        </div>
                      </div>
                    </div>
                  </div>
                )
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
                <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--medium u-fontWeight--medium">{parentClusterName} has been configured but still needs to be deployed. Select a cluster you would like to deploy {parentClusterName} to.</p>
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
