import React from "react";
import { withRouter } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";

import {
  updateWatch,
  deleteWatch
} from "@src/mutations/WatchMutations";

import Loader from "@src/components/shared/Loader";

export function PendingHelmChartDetailPage(props) {
  const { chart } = props;
  console.log(chart);
  if (!chart) {
    return (
      <div className="flex1 flex-column alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );
  }

  return (
    <div className="DetailPageApplication--wrapper flex-column flex1 centered-container alignItems--center u-overflow--auto u-paddingTop--20 u-paddingBottom--20">
      <div className="DetailPageApplication flex flex1">
        <div className="flex1 flex-column u-paddingRight--30">
          <div className="flex">
            <div className="flex flex-auto">
              <span
                style={{ backgroundImage: `url(${chart.helmIcon})` }}
                className="DetailPageApplication--appIcon u-position--relative">
              </span>
            </div>
            <div className="flex-column flex1 justifyContent--center u-marginLeft--10 u-paddingLeft--5">
              <p className="u-fontSize--30 u-color--tuna u-fontWeight--bold">{chart.helmName}</p>
              <div className="u-marginTop--10 flex-column">
                <div className="flex-auto">
                  <button className="btn primary">See values.yaml</button>
                </div>
                <div className="flex-auto">
                  <button className="btn primary">Get chart YAMl</button>
                </div>
              </div>
            </div>
          </div>
          <div className="u-marginTop--30">
            <div className="midstream-banner">
              <p className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-color--nevada">This is a pending watch from an upstream Helm chart. You should unfork it so you can better manage updates directly from the upstream and set up automatic deployments.</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

export default compose(
  withApollo,
  withRouter,
  graphql(updateWatch, {
    props: ({ mutate }) => ({
      updateWatch: (watchId, watchName, iconUri) => mutate({ variables: { watchId, watchName, iconUri } })
    })
  }),
  graphql(deleteWatch, {
    props: ({ mutate }) => ({
      deleteWatch: (watchId, childWatchIds) => mutate({ variables: { watchId, childWatchIds } })
    })
  })
)(PendingHelmChartDetailPage);
