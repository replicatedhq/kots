import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import { compose, withApollo, graphql } from "react-apollo";
import classNames from "classnames";
import Loader from "../shared/Loader";
import DownstreamVersionRow from "./DownstreamVersionRow";
import filter from "lodash/filter";

import { getDownstreamHistory } from "../../queries/WatchQueries";
import { getKotsDownstreamHistory } from "../../queries/AppsQueries";

import "@src/scss/components/watches/WatchVersionHistory.scss";
import { isKotsApplication, hasPendingPreflight } from "../../utilities/utilities";

class DownstreamWatchVersionHistory extends Component {

  handleMakeCurrent = async (upstreamSlug, sequence, clusterSlug) => {
    if (this.props.makeCurrentVersion && typeof this.props.makeCurrentVersion === "function") {
      await this.props.makeCurrentVersion(upstreamSlug, sequence, clusterSlug);
      await this.props.data.refetch();
    }
  }

  getActiveDownstreamVersion = versionHistory => {
    if (!versionHistory.length) {
      return null;
    }
    const deployed = filter(versionHistory, version => version.status === "deployed");
    deployed.sort((v1, v2) => v1.sequence > v2.sequence);
    return deployed.length ? deployed[0] : null;
  }

  render() {
    const { watch, match, data } = this.props;
    const { watches, downstreams } = watch;
    const isKots = isKotsApplication(watch);
    const _slug = isKots ? match.params.downstreamSlug : `${match.params.downstreamOwner}/${match.params.downstreamSlug}`;
    const downstreamWatch = isKots ? downstreams.find(w => w.cluster.slug === _slug) : watches.find(w => w.slug === _slug );
    let versionHistory = [];
    if (isKots && data?.getKotsDownstreamHistory?.length) {
      versionHistory = data.getKotsDownstreamHistory;
    } else if (data?.getDownstreamHistory?.length) {
      versionHistory = data.getDownstreamHistory;
    }
    const activeDownstreamVersion = this.getActiveDownstreamVersion(versionHistory);
    const downstreamSlug = downstreamWatch ? downstreamWatch.cluster?.slug : "";
    const isGit = downstreamWatch?.cluster?.gitOpsRef;

    const centeredLoader = (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );

    if (isKots && hasPendingPreflight(versionHistory)) {
      data?.startPolling(2000);
    } else {
      this.props?.refreshAppData();
      data?.stopPolling();
    }

    return (
      <div className="flex-column flex1 u-position--relative u-padding--20 u-overflow--auto">
        <p className="flex-auto u-fontSize--larger u-fontWeight--bold u-color--tuna u-paddingBottom--20">Downstream version history: {downstreamSlug}</p>

        <div className="flex-column flex-auto ActiveRelease-wrapper">
          <div className="flex alignItems--center u-borderBottom--gray u-paddingBottom--5">
            <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna">Active release</p>
          </div>
          <div>
            {activeDownstreamVersion ?
              <DownstreamVersionRow
                key="current-downstream-version"
                downstreamWatch={downstreamWatch}
                version={activeDownstreamVersion}
                isKots={isKots}
                urlParams={match.params}
                handleMakeCurrent={this.handleMakeCurrent}
              />
            :
              <div className="no-current-version u-textAlign--center">
                <p className="u-fontSize--large u-color--tundora u-fontWeight--bold u-lineHeight--normal">No active release found on {downstreamSlug}</p>
                <p className="u-fontSize--normal u-color--dustygray u-fontWeight--medium u-lineHeight--normal">{isGit ? "When a PR is merged" : "When a version has been deployed"}, the current version will be shown here</p>
              </div>
            }
          </div>
        </div>

        <div className="flex1 flex-column u-paddingTop--20 u-marginTop--20">
          <div className="flex alignItems--center u-borderBottom--gray u-paddingBottom--5">
            <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna">All releases</p>
          </div>
          <div className={classNames("flex-column", { "flex1": data.loading })}>
            {data.loading
            ? centeredLoader
            : versionHistory?.length > 0 && versionHistory.map( version => (
              <DownstreamVersionRow
                hasPreflight={watch.hasPreflight}
                key={`${version.title}-${version.sequence}`}
                downstreamWatch={downstreamWatch}
                version={version}
                isKots={isKots}
                urlParams={match.params}
                handleMakeCurrent={this.handleMakeCurrent}
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
  graphql(getKotsDownstreamHistory, {
    skip: props => {
      return props.match.params.downstreamOwner;
    },
    options: ({ match }) => ({
      variables: {
        upstreamSlug: match.params.slug,
        clusterSlug: match.params.downstreamSlug,
      },
      fetchPolicy: "no-cache"
    })
  }),
  graphql(getDownstreamHistory, {
    skip: props => {
      return !props.match.params.downstreamOwner;
    },
    options: ({ match }) => ({
      variables: {
        slug: `${match.params.downstreamOwner}/${match.params.downstreamSlug}`
      },
      fetchPolicy: "no-cache"
    })
  }),
)(DownstreamWatchVersionHistory);