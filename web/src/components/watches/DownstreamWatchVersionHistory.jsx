import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import { compose, withApollo, graphql } from "react-apollo";
import classNames from "classnames";
// import ReactTooltip from "react-tooltip"
import { getClusterType, Utilities } from "@src/utilities/utilities";
import { getDownstreamHistory } from "../../queries/WatchQueries";

import "@src/scss/components/watches/WatchVersionHistory.scss";

class DownstreamWatchVersionHistory extends Component {
  render() {
    const { watch, match, data } = this.props;
    const { currentVersion, watches} = watch;
    const _slug = `${match.params.downstreamOwner}/${match.params.downstreamSlug}`;
    const downstreamWatch = watches.find(w => w.slug === _slug );
    const versionHistory = data?.getDownstreamHistory?.length ? data.getDownstreamHistory : [];
    const downstreamSlug = downstreamWatch ? downstreamWatch.cluster?.slug : "";
    const isGit = downstreamWatch?.cluster?.gitOpsRef;
    const clusterIcon = getClusterType(isGit) === "git" ? "icon github-small-size" : "icon ship-small-size";

    return (
      <div className="flex-column u-position--relative u-overflow--auto u-padding--20">
        <p className="flex-auto u-fontSize--larger u-fontWeight--bold u-color--tuna u-paddingBottom--20">Downstream version history: {downstreamSlug}</p>
        <div className="flex alignItems--center u-borderBottom--gray u-paddingBottom--5">
          <p className="u-fontSize--header u-fontWeight--bold u-color--tuna">
            {currentVersion ? currentVersion.title : "---"}
          </p>
          <p className="u-fontSize--large u-fontWeight--medium u-marginLeft--10">{currentVersion ? "Current upstream version" : "No deployments made"}</p>
          <div className="flex flex1 justifyContent--flexEnd">
            <div className="watch-cell flex">
              <div className="flex flex1 cluster-cell-title justifyContent--center alignItems--center u-fontWeight--bold u-color--tuna">
                <span className={classNames(clusterIcon, "flex-auto u-marginRight--5")} />
                <p className="u-fontSize--small u-fontWeight--medium u-color--tuna">
                  {downstreamSlug}
                </p>
              </div>
            </div>
          </div>
        </div>
        <div className="flex-column">
          {versionHistory?.length > 0 && versionHistory.map( version => {
            if (!version) return null;
            const gitRef = downstreamWatch?.cluster?.gitOpsRef;
            const githubLink = gitRef && `https://github.com/${gitRef.owner}/${gitRef.repo}/pull/${version.pullrequestNumber}`;
            return (
              <div
                key={`${version.title}-${version.sequence}`}
                className="flex u-paddingTop--20 u-paddingBottom--20 u-borderBottom--gray">
                <div className="flex alignItems--center u-fontSize--larger u-color--tuna u-fontWeight--bold u-marginLeft--10">
                  Version {version.title}
                  {version.pullrequestNumber &&
                    <div>
                      <span className="icon integration-card-icon-github u-marginLeft--10" />
                      <a
                        className="u-color--astral u-marginLeft--5"
                        href={githubLink}
                        rel="noopener noreferrer"
                        target="_blank">
                          #{version.pullrequestNumber}
                      </a>
                    </div>
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