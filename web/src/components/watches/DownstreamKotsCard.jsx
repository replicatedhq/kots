import React from "react";
import truncateMiddle from "truncate-middle";
import { Link } from "react-router-dom";
import "../../scss/components/clusters/ClusterCard.scss";
import "../../scss/components/watches/WatchCard.scss";

export default class DownstreamKotsCard extends React.Component {

  render() {
    const {
      downstream,
      toggleDeleteDeploymentModal,
      displayDownloadCommand,
      viewFiles,
      isDownloadingAssets,
      appSlug
     } = this.props;

    const cluster = downstream?.cluster;
    const type = cluster?.gitOpsRef ? "git" : "ship";
    const gitPath = cluster?.gitOpsRef ? `${cluster.gitOpsRef.owner}/${cluster.gitOpsRef.repo}/${cluster.gitOpsRef.branch}${cluster.gitOpsRef.path}` : "";
    const hasDeployments = downstream?.currentVersion && downstream.currentVersion.status !== "failed";

    return (
      <div className="integration flex-column flex1 flex">
        <div className="flex u-marginBottom--5">
          <span className={`normal u-marginRight--5 icon clusterType ${type}`}></span>
          <div className="flex1 justifyContent--center">
            <div className="flex justifyContent--spaceBetween">
              <p className="flex1 u-fontWeight--bold u-fontSize--large u-color--tundora u-paddingRight--5">{cluster?.title || "Downstream deployment"}</p>
              <span className="flex-auto icon u-grayX-icon clickable" onClick={() => toggleDeleteDeploymentModal(downstream)}></span>
            </div>
            <p className="u-fontWeight--medium u-fontSize--small u-color--dustyGray u-marginTop--5" title={gitPath}>{type === "git" ? truncateMiddle(gitPath, 22, 22, "...") : "Deployed with kotsadm"}</p>
          </div>
        </div>
        <div className="u-marginTop--10">
          <div className="flex flex1">
            {hasDeployments && <h2 className="u-fontSize--jumbo2 alignSelf--center u-fontWeight--bold u-color--tuna">{downstream.currentVersion?.title}</h2>}
            {!hasDeployments &&
              <div className="flex-auto flex flex1 alignItems--center alignSelf--center">
                <div className="icon blueCircleMinus--icon"></div>
                <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-marginLeft--5">No deployments made</p>
              </div>
            }
            {hasDeployments && downstream?.pendingVersions?.length === 1 &&
              <div className="flex-auto flex flex1 alignItems--center alignSelf--center">
                <div className="icon exclamationMark--icon u-marginLeft--10"></div>
                <p className="u-fontSize--normal u-color--orange u-fontWeight--medium u-marginLeft--5">One version behind</p>
              </div>
            }
            {hasDeployments && downstream?.pendingVersions?.length >= 2 &&
              <div className="flex-auto flex flex1 alignItems--center alignSelf--center">
                <div className="icon exclamationMark--icon u-marginLeft--10"></div>
                <p className="u-fontSize--normal u-color--orange u-fontWeight--medium u-marginLeft--5">Two or more versions behind</p>
              </div>
            }
            {hasDeployments && !downstream?.pendingVersions?.length &&
              <div className="flex-auto flex flex1 alignItems--center alignSelf--center">
                <div className="icon checkmark-icon u-marginLeft--10"></div>
                <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-marginLeft--5">Up to date</p>
              </div>
            }
          </div>
          <Link to={`/app/${appSlug}/downstreams/${cluster?.slug}/version-history`} className="replicated-link u-fontSize--normal u-lineHeight--normal">See version history</Link>
        </div>
        {downstream?.currentVersion && downstream.pendingVersions?.length >= 1 &&
          <div className="flex justifyContent--spaceBetween alignItems--center u-marginTop--10">
            {type === "git" ?
              <a href="" className="btn green secondary" target="_blank" rel="noopener noreferrer">Review PR to update application</a>
            :
              <div className="flex-column">
                <button onClick={undefined} className="btn green secondary">Install latest version</button>
              </div>
            }
          </div>
        }
        {downstream?.currentVersion && !downstream?.pendingVersions?.length &&
          <div className="flex justifyContent--spaceBetween alignItems--center  u-marginTop--5">
            <p className="u-fontWeight--medium u-fontSize--small u-color--dustyGray u-marginTop--5 u-lineHeight--more">When an update is available you will be able to deploy it from here.</p>
          </div>
        }
        <div className="flex flex1 alignItems--flexEnd">
          <div className="flex u-marginTop--20 u-borderTop--gray u-width--full">
            <div className="flex1 flex card-action-wrapper u-cursor--pointer">
              <span className="flex1 u-marginRight--5 u-color--astral card-action u-fontSize--small u-fontWeight--medium u-textAlign--center" onClick={isDownloadingAssets ? null : displayDownloadCommand }>{isDownloadingAssets ? "Downloading" : "Download assets"}</span>
            </div>
            <div className="flex1 flex card-action-wrapper u-cursor--pointer">
              <span onClick={viewFiles} className="flex1 u-marginRight--5 u-color--astral card-action u-fontSize--small u-fontWeight--medium u-textAlign--center">View files</span>
            </div>
          </div>
        </div>
      </div>
    );
  }
}
