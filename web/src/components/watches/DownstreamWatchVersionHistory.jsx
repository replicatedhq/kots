import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import { compose, withApollo, graphql } from "react-apollo";
import classNames from "classnames";
import Loader from "../shared/Loader";
import DownstreamVersionRow from "./DownstreamVersionRow";

import { getClusterType } from "@src/utilities/utilities";
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
    const { watches} = watch;
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

        <div className="flex-column flex-auto">
          <div className="flex alignItems--center u-borderBottom--gray u-paddingBottom--5">
            <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna">Active release</p>
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
          <div>
            <DownstreamVersionRow
              downstreamWatch={downstreamWatch}
              version={downstreamWatch.currentVersion} 
            />
          </div>
        </div>

        <div className="flex1 flex-column u-paddingTop--20 u-marginTop--20">
          <div className="flex alignItems--center u-borderBottom--gray u-paddingBottom--5">
            <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna">All releases</p>
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
            : versionHistory?.length > 0 && versionHistory.map( version => (
              <DownstreamVersionRow
                key={`${version.title}-${version.sequence}`}
                downstreamWatch={downstreamWatch}
                version={version} 
              />
            ))}
          </div>
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
      },
      fetchPolicy: "no-cache"
    })
  })
)(DownstreamWatchVersionHistory);