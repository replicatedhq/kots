import React from "react";
import truncateMiddle from "truncate-middle";
import { Link } from "react-router-dom";
import "../../scss/components/clusters/ClusterCard.scss";
import "../../scss/components/watches/WatchCard.scss";

export default class DownstreamCard extends React.Component {

  render() {
    const {
      childWatch,
      installLatestVersion,
      downloadAssetsForCluster,
      toggleDeleteDeploymentModal,
      isDownloadingAssets,
      parentWatch,
     } = this.props;
    const { cluster } = childWatch;
    const currentVersion = childWatch.currentVersion ? childWatch.currentVersion.title : "Unknown";
    const type = cluster && cluster.gitOpsRef ? "git" : "ship";
    const gitPath = cluster && cluster.gitOpsRef ? `${cluster.gitOpsRef.owner}/${cluster.gitOpsRef.repo}/${cluster.gitOpsRef.branch}${cluster.gitOpsRef.path}` : "";

    return (
      <div className="integration flex-column flex1 flex">
        <div className="flex u-marginBottom--5">
          <span className={`normal u-marginRight--5 icon clusterType ${type}`}></span>
          <div className="flex1 justifyContent--center">
            <div className="flex justifyContent--spaceBetween">
              <p className="flex1 u-fontWeight--bold u-fontSize--large u-color--tundora u-paddingRight--5">{cluster && cluster.title || "Downstream deployment"}</p>
              <span className="flex-auto icon u-grayX-icon clickable" onClick={() => toggleDeleteDeploymentModal(childWatch, parentWatch.watchName)}></span>
            </div>
            <p className="u-fontWeight--medium u-fontSize--small u-color--dustyGray u-marginTop--5" title={gitPath}>{type === "git" ? truncateMiddle(gitPath, 22, 22, "...") : "Deployed with Ship"}</p>
            <div className="cluster-actions-wrapper u-fontSize--small u-lineHeight--normal">
              <span>
                <Link
                  to={`/watch/${childWatch.slug}/state`}
                  className="replicated-link u-marginTop--5">
                  View state.json
                </Link>
              </span>
              <span className="replicated-link" onClick={isDownloadingAssets ? null : () => { downloadAssetsForCluster(childWatch.id) }}>{isDownloadingAssets ? "Downloading" : "Download assets"}</span>
            </div>
          </div>
        </div>
        <div className="u-marginTop--10">
          <div className="flex flex1">
            <h2 className="u-fontSize--jumbo2 alignSelf--center u-fontWeight--bold u-color--tuna">{currentVersion}</h2>
            {!childWatch.currentVersion &&
              <div className="flex-auto flex flex1 alignItems--center alignSelf--center">
                <div className="icon blueCircleMinus--icon u-marginLeft--10"></div>
                <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-marginLeft--5">No deployments made</p>
              </div>
            }
            {childWatch.currentVersion && childWatch.pendingVersions.length === 1 &&
              <div className="flex-auto flex flex1 alignItems--center alignSelf--center">
                <div className="icon exclamationMark--icon u-marginLeft--10"></div>
                <p className="u-fontSize--normal u-color--orange u-fontWeight--medium u-marginLeft--5">One version behind</p>
              </div>
            }
            {childWatch.currentVersion && childWatch.pendingVersions.length >= 2 &&
              <div className="flex-auto flex flex1 alignItems--center alignSelf--center">
                <div className="icon exclamationMark--icon u-marginLeft--10"></div>
                <p className="u-fontSize--normal u-color--orange u-fontWeight--medium u-marginLeft--5">Two or more versions behind</p>
              </div>
            }
            {childWatch.currentVersion && !childWatch.pendingVersions.length &&
              <div className="flex-auto flex flex1 alignItems--center alignSelf--center">
                <div className="icon checkmark-icon u-marginLeft--10"></div>
                <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-marginLeft--5">Up to date</p>
              </div>
            }
          </div>
          <Link to={`/watch/${parentWatch.slug}/downstreams/${childWatch.slug}/version-history`} className="replicated-link u-fontSize--normal u-lineHeight--normal">See version history</Link>
        </div>
        {currentVersion && childWatch.pendingVersions.length >= 1 &&
          <div className="flex justifyContent--spaceBetween alignItems--center u-marginTop--10">
            {type === "git" ?
              <a href={`https://github.com/${cluster.gitOpsRef.owner}/${cluster.gitOpsRef.repo}/pull/${childWatch.pendingVersions[0].pullrequestNumber}`} className="btn green secondary" target="_blank" rel="noopener noreferrer">Review PR to update application</a>
            :
              <div className="flex-column">
                <button onClick={() => installLatestVersion(childWatch.id, childWatch.pendingVersions[0].sequence)} className="btn green secondary">Install latest version ({childWatch.pendingVersions[0].title})</button>
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
              <span className="flex1 u-marginRight--5 u-color--astral card-action u-fontSize--small u-fontWeight--medium u-textAlign--center" onClick={() => { this.props.history.push(`/watch/${childWatch.slug}/tree/${childWatch.currentVersion?.sequence || 0}`) }}>View file contents</span>
            </div>
            <div className="flex1 flex card-action-wrapper u-cursor--pointer">
              <span onClick={this.props.preparingUpdate === childWatch.cluster.id ? () => { return } : () => this.props.onEditApplication(childWatch)} className="flex1 u-marginRight--5 u-color--astral card-action u-fontSize--small u-fontWeight--medium u-textAlign--center">{this.props.preparingUpdate === childWatch.cluster.id ? "Preparing" : "Edit downstream"}</span>
            </div>
          </div>
        </div>
      </div>
    );
  }
}
