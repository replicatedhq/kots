import React, { Component } from "react";
import classNames from "classnames";
import { withRouter } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";

import withTheme from "@src/components/context/withTheme";
import { listWatches, listPendingInit, listHelmCharts } from "@src/queries/WatchQueries";
import WatchSidebarItem from "@src/components/watches/WatchSidebarItem";
import HelmChartSidebarItem from "@src/components/watches/WatchSidebarItem/HelmChartSidebarItem";
import SubNavBar from "@src/components/shared/SubNavBar";
import SidebarLayout from "../layout/SidebarLayout/SidebarLayout";
import SideBar from "../shared/SideBar";
import Loader from "../shared/Loader";

import "../../scss/components/watches/WatchDetailPage.scss";

class HelmChartDetailPage extends Component {
  constructor(props) {
    super(props);
    this.state = {
      displayRemoveClusterModal: false,
      addNewClusterModal: false,
      preparingUpdate: "",
      clusterParentSlug: "",
      selectedWatchName: "",
      clusterToRemove: {},
      watchToEdit: {},
      existingDeploymentClusters: []
    }
  }

  componentDidUpdate(/* lastProps */) {
    const { getThemeState, setThemeState, match, listWatchesQuery } = this.props;
    const slug = `${match.params.owner}/${match.params.slug}`;

    const currentWatch = listWatchesQuery?.listWatches?.find( w => w.slug === slug);

    // Handle updating the navbar logo when a watch changes.
    if (currentWatch?.watchIcon) {
      const { navbarLogo } = getThemeState();
      if (navbarLogo === null || navbarLogo !== currentWatch.watchIcon) {
        setThemeState({
          navbarLogo: currentWatch.watchIcon
        });
      }
    }
  }

  componentWillUnmount() {
    clearInterval(this.interval);
    this.props.clearThemeState();
  }

  render() {
    const watch = {};
    const allWatches = [];
    const slug = "";
    return (
      <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
        <SidebarLayout
          className="flex u-minHeight--full u-overflow--hidden"
          sidebar={(
            <SideBar
              className="flex flex1"
              aggressive={true}
              loading={this.props.listWatchesQuery.loading}
              items={allWatches?.map( (item, idx) => {
                let sidebarNode;
                if (item.slug) {
                  sidebarNode = (
                    <WatchSidebarItem
                      key={idx}
                      className={classNames({ selected: item.slug === watch.slug})}
                      watch={item} />
                  );
                } else if (item.helmName) {
                  sidebarNode = (
                    <HelmChartSidebarItem
                      key={idx}
                      className={classNames({ selected: item.slug === watch.slug})}
                      watch={item} />
                  );
                }
                return sidebarNode;
              })}
              currentWatch={watch?.watchName}
            />
          )}>
          <div className="flex-column flex3 u-width--full u-height--full">
            <SubNavBar
              className="flex"
              activeTab={this.props.match.params.tab || "app"}
              slug={slug}
              watch={watch}
            />
            {
              this.props.listWatchesQuery.loading ? (
                <div className="flex-column flex1 alignItems--center justifyContent--center">
                  <Loader size="60" />
                </div>
              ) :
              (
                <div>
                  helm chart page
                </div>
              )
            }
          </div>
        </SidebarLayout>
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  withTheme,
  graphql(listWatches, {
    name: "listWatchesQuery",
    options: {
      fetchPolicy: "cache-and-network"
    }
  }),
  graphql(listPendingInit, {
    name: "listPendingInitQuery",
    options: {
      fetchPolicy: "cache-and-network"
    }
  }),
  graphql(listHelmCharts, {
    name: "listHelmChartsQuery",
    options: {
      fetchPolicy: "cache-and-network"
    }
  })
)(HelmChartDetailPage);
