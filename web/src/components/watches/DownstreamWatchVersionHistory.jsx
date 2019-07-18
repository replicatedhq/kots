import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import { compose, withApollo, graphql } from "react-apollo";
import classNames from "classnames";
import Loader from "../shared/Loader";
import dayjs from "dayjs";
import { getClusterType, Utilities } from "@src/utilities/utilities";
import { getDownstreamHistory } from "../../queries/WatchQueries";

import "@src/scss/components/watches/WatchVersionHistory.scss";

class DownstreamWatchVersionHistory extends Component {

  handleMakeCurrent = (id, sequence) => {
    if (this.props.makeCurrentVersion && typeof this.props.makeCurrentVersion === "function") {
      this.props.makeCurrentVersion(id, sequence);
    }
  }
  
  render() {
    const { watch, match, data } = this.props;
    const { currentVersion, watches} = watch;
    const _slug = `${match.params.downstreamOwner}/${match.params.downstreamSlug}`;
    const downstreamWatch = watches.find(w => w.slug === _slug );
    const versionHistory = data?.getDownstreamHistory?.length ? data.getDownstreamHistory : [];
    const downstreamSlug = downstreamWatch ? downstreamWatch.cluster?.slug : "";
    const isGit = downstreamWatch?.cluster?.gitOpsRef;
    const clusterIcon = getClusterType(isGit) === "git" ? "icon github-small-size" : "icon ship-small-size";

    const centeredLoader = (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );

    return (
      <div className="flex-column flex1 u-position--relative u-overflow--hidden u-padding--20">
        <p className="flex-auto u-fontSize--larger u-fontWeight--bold u-color--tuna u-paddingBottom--20">Downstream version history: {downstreamSlug}</p>
        <div className="flex alignItems--center u-borderBottom--gray u-paddingBottom--5">
          <p className="u-fontSize--header u-fontWeight--bold u-color--tuna">
            {currentVersion ? currentVersion.title : "---"}
          </p>
          <p className="u-fontSize--large u-fontWeight--medium u-marginLeft--10">{currentVersion ? "Current upstream version" : "No deployments made"}</p>
          <div className="flex flex1 justifyContent--flexEnd">
            <div className="flex">
              <div className="flex flex1 cluster-cell-title justifyContent--center alignItems--center u-fontWeight--bold u-color--tuna">
                <span className={classNames(clusterIcon, "flex-auto u-marginRight--5")} />
                <p className="u-fontSize--small u-fontWeight--medium u-color--tuna">
                  {downstreamSlug}
                </p>
              </div>
            </div>
          </div>
        </div>
        <div className="flex-column flex1 u-overflow--auto">
          {data.loading
          ? centeredLoader
          : versionHistory?.length > 0 && versionHistory.map( version => {
            if (!version) return null;
            const gitRef = downstreamWatch?.cluster?.gitOpsRef;
            const githubLink = gitRef && `https://github.com/${gitRef.owner}/${gitRef.repo}/pull/${version.pullrequestNumber}`;
            let shipInstallnode = null;
            if (!gitRef && version.status === "pending") {
              shipInstallnode = (
                <div className="u-marginLeft--10 flex-column flex-auto flex-verticalCenter">
                  <button className="btn secondary small" onClick={() => this.handleMakeCurrent(downstreamWatch.id, version.sequence)}>Make current version</button>
                </div>
              )
            }
            let deployedAtTextNode;
            if (version.deployedAt) {
              deployedAtTextNode = `${gitRef ? "Merged" : "Deployed"} on ${dayjs(version.deployedAt).format("MMMM D, YYYY")}`;
            } else if (gitRef) {
              deployedAtTextNode = <span className="gh-version-detail-text">Merged on date not available. <a className="replicated-link" href={githubLink} rel="noopener noreferrer" target="_blank">View the PR</a> to see when it was merged.</span>
            } else {
              deployedAtTextNode = "Deployed on date not available.";
            }
            return (
              <div key={`${version.title}-${version.sequence}`} className="flex u-paddingTop--20 u-paddingBottom--20 u-borderBottom--gray">
                <div className="flex-column flex1">
                  <div className="flex alignItems--center u-fontSize--larger u-color--tuna u-fontWeight--bold">
                    Version {version.title}
                    {shipInstallnode}
                  </div>
                  {version.status === "deployed" || version.status === "merged" &&
                    <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginTop--10 flex alignItems--center">
                      {version.pullrequestNumber &&
                        <span className="icon integration-card-icon-github u-marginRight--5" />
                      }
                      {deployedAtTextNode}
                    </p>
                  }
                  {version.pullrequestNumber && (version.status === "opened" || version.status === "pending") &&
                    <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginTop--10 flex alignItems--center">
                      <span className="icon integration-card-icon-github u-marginRight--5" />
                      <span className="gh-version-detail-text"><a className="replicated-link" href={githubLink} rel="noopener noreferrer" target="_blank">View this PR on GitHub</a> to review and merged it in for deployment.</span>
                    </p>
                  }
                </div>
                <div className="flex flex1 justifyContent--flexEnd alignItems--center">
                  <div className="watch-cell">
                    <div className="flex justifyContent--center alignItems--center">
                        <div
                          data-tip={`${version.title}-${version.sequence}`}
                          data-for={`${version.title}-${version.sequence}`}
                          className={classNames("icon", {
                          "checkmark-icon": version.status === "deployed" || version.status === "merged",
                          "exclamationMark--icon": version.status === "opened" || version.status === "pending",
                          "grayCircleMinus--icon": version.status === "closed"
                          })}
                        />
                        <span className={classNames("u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginLeft--5", {
                          "u-color--nevada": version.status === "deployed" || version.status === "merged",
                          "u-color--orange": version.status === "opened" || version.status === "pending",
                          "u-color--dustyGray": version.status === "closed"
                        })}>{Utilities.toTitleCase(version.status)}</span>
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  graphql(getDownstreamHistory, {
    options: ({ match }) => ({
      variables: {
        slug: `${match.params.downstreamOwner}/${match.params.downstreamSlug}`
      }
    })
  })
)(DownstreamWatchVersionHistory);