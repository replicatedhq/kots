import React, { Component, Fragment } from "react";
import classNames from "classnames";
import { withRouter, Switch, Route } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";

import withTheme from "@src/components/context/withTheme";
import { getKotsApp } from "@src/queries/AppsQueries";
import { checkForUpdates } from "../../mutations/WatchMutations";
import WatchSidebarItem from "@src/components/watches/WatchSidebarItem";
import { KotsSidebarItem } from "@src/components/watches/WatchSidebarItem";
import { HelmChartSidebarItem } from "@src/components/watches/WatchSidebarItem";
import NotFound from "../static/NotFound";
import DetailPageApplication from "../watches/DetailPageApplication";
import DetailPageIntegrations from "../watches/DetailPageIntegrations";
import DeploymentClusters from "../watches/DeploymentClusters";
import DownstreamTree from "../../components/tree/KotsApplicationTree";
import WatchVersionHistory from "../watches/WatchVersionHistory";
import DownstreamWatchVersionHistory from "../watches/DownstreamWatchVersionHistory";
import WatchConfig from "../watches/WatchConfig";
import WatchLicense from "../watches/WatchLicense";
import SubNavBar from "@src/components/shared/SubNavBar";
import SidebarLayout from "../layout/SidebarLayout/SidebarLayout";
import SideBar from "../shared/SideBar";
import Loader from "../shared/Loader";
import SupportBundleList from "../troubleshoot/SupportBundleList";
import SupportBundleAnalysis from "../troubleshoot/SupportBundleAnalysis";
import GenerateSupportBundle from "../troubleshoot/GenerateSupportBundle";

import "../../scss/components/watches/WatchDetailPage.scss";

let loadingTextTimer = null;
class AppDetailPage extends Component {
  constructor(props) {
    super(props);
    this.state = {
      preparingUpdate: "",
      clusterParentSlug: "",
      selectedWatchName: "",
      clusterToRemove: {},
      watchToEdit: {},
      existingDeploymentClusters: [],
      checkingForUpdates: false,
      checkingUpdateText: "Checking for updates",
      updateError: false,
    }
  }

  static defaultProps = {
    getKotsAppQuery: {
      loading: true
    }
  }

  componentDidUpdate(lastProps) {
    const { getThemeState, setThemeState, match, listApps, history, getKotsAppQuery } = this.props;
    const { search } = this.props.location;
    const slug = `${match.params.owner}/${match.params.slug}`;
    const currentWatch = listApps?.find(w => w.slug === slug);

    // Handle updating the app theme state when a watch changes.
    if (currentWatch ?.watchIcon) {
      const { navbarLogo, ...rest } = getThemeState();
      if (navbarLogo === null || navbarLogo !== currentWatch.watchIcon) {

        setThemeState({
          ...rest,
          navbarLogo: currentWatch.watchIcon
        });
      }
    }

    if (getKotsAppQuery.getKotsApp && getKotsAppQuery.getKotsApp !== lastProps.getKotsAppQuery.getKotsApp) {
      const URLParams = new URLSearchParams(search);
      if (URLParams.get("add")) {
        this.handleAddNewClusterClick(getKotsAppQuery.getKotsApp);
        history.replace(this.props.location.pathname); // remove query param so refreshing the page doesn't trigger the modal again.
      }
    }

    // Used for a fresh reload
    if (history.location.pathname === "/apps") {
      this.checkForFirstApp();
    }

  }

  componentWillUnmount() {
    clearInterval(this.interval);
    this.props.clearThemeState();
  }

  makeCurrentRelease = async (watchId, sequence) => {
    await this.props.deployWatchVersion(watchId, sequence).then(() => {
      this.props.getKotsAppQuery.refetch();
    })
  }

  onCheckForUpdates = async () => {
    const { client, getKotsAppQuery } = this.props;
    const { getKotsApp: watch } = getKotsAppQuery;
    this.setState({ checkingForUpdates: true });
    loadingTextTimer = setTimeout(() => {
      this.setState({ checkingUpdateText: "Almost there, hold tight..." });
    }, 10000);
    await client.mutate({
      mutation: checkForUpdates,
      variables: {
        watchId: watch.id,
      }
    }).catch(() => {
      this.setState({ updateError: true });
    }).finally(() => {
      this.props.getKotsAppQuery.refetch();
      clearTimeout(loadingTextTimer);
      this.setState({
        checkingForUpdates: false,
        checkingUpdateText: "Checking for updates"
      });
    });
  }
  addClusterToWatch = (clusterId, githubPath) => {
    const { clusterParentSlug } = this.state;
    const upstreamUrl = `ship://ship-cloud/${clusterParentSlug}`;
    this.props.history.push(`/watch/create/init?upstream=${upstreamUrl}&cluster_id=${clusterId}&path=${githubPath}`);
  }

  createDownstreamForCluster = () => {
    const { clusterParentSlug } = this.state;
    localStorage.setItem("clusterRedirect", `/watch/${clusterParentSlug}/downstreams?add=1`);
    this.props.history.push("/cluster/create");
  }

  onEditApplicationClicked = (watch) => {
    const { onActiveInitSession } = this.props;

    this.setState({ watchToEdit: watch, preparingUpdate: watch.cluster.id });
    this.props.createUpdateSession(watch.id)
      .then(({ data }) => {
        const { createUpdateSession } = data;
        const { id: initSessionId } = createUpdateSession;
        onActiveInitSession(initSessionId);
        this.props.history.push("/ship/update")
      })
      .catch(() => this.setState({ watchToEdit: null, preparingUpdate: "" }));
  }

  /**
   * Refetch all the GraphQL data for this component and all its children
   *
   * @return {undefined}
   */
  refetchGraphQLData = () => {
    this.props.getKotsAppQuery.refetch()
  }

  /**
   *  Runs on mount and on update. Also handles redirect logic
   *  if no watches are found, or the first watch is found.
   */
  checkForFirstApp = () => {
    const { history, rootDidInitialAppFetch, listApps } = this.props;
    if (!rootDidInitialAppFetch) {
      return;
    }

    if (listApps.length > 0) {
      const firstApp = listApps.find(app => app.name);
      history.replace(`/app/${firstApp.slug}`);
    } else {
      history.replace("/watch/create/init");
    }
  }

  componentDidMount() {
    const { history } = this.props;

    if (history.location.pathname === "/apps") {
      return this.checkForFirstApp();
    }

  }

  render() {
    const {
      match,
      getKotsAppQuery,
      listApps,
      refetchListApps,
      rootDidInitialAppFetch
    } = this.props;

    const {
      checkingUpdateText,
      updateError
    } = this.state;

    const centeredLoader = (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );

    const app = getKotsAppQuery?.getKotsApp;
    const loading = getKotsAppQuery?.loading || !rootDidInitialAppFetch;

    if (!rootDidInitialAppFetch) {
      return centeredLoader;
    }

    return (
      <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
        <SidebarLayout
          className="flex flex1 u-minHeight--full u-overflow--hidden"
          condition={listApps ?.length > 1}
          sidebar={(
            <SideBar
              items={listApps ?.map((item, idx) => {
                let sidebarItemNode;
                if (item.name) {
                  const slugFromRoute = match.params.slug;
                  sidebarItemNode = (
                    <KotsSidebarItem
                      key={idx}
                      className={classNames({
                        selected: (
                          item.slug === slugFromRoute &&
                          match.params.owner !== "helm"
                        )
                      })}
                      app={item} />
                  );
                } else if (item.slug) {
                  const slugFromRoute = `${match.params.owner}/${match.params.slug}`;
                  sidebarItemNode = (
                    <WatchSidebarItem
                      key={idx}
                      className={classNames({
                        selected: (
                          item.slug === slugFromRoute &&
                          match.params.owner !== "helm"
                        )
                      })}
                      watch={item} />
                  );
                } else if (item.helmName) {
                  sidebarItemNode = (
                    <HelmChartSidebarItem
                      key={idx}
                      className={classNames({ selected: item.id === match.params.slug })}
                      helmChart={item} />
                  );
                }
                return sidebarItemNode;
              })}
            />
          )}>
          <div className="flex-column flex1 u-width--full u-height--full">
            {loading
              ? centeredLoader
              : (
                <Fragment>
                  <SubNavBar
                    className="flex"
                    activeTab={match.params.tab || "app"}
                    watch={app}
                  />
                  <Switch>
                    <Route exact path="/app/:slug" render={() =>
                      <DetailPageApplication
                        watch={app}
                        refetchListApps={refetchListApps}
                        refetchWatch={this.props.getKotsAppQuery ?.refetch}
                        updateCallback={this.refetchGraphQLData}
                        onActiveInitSession={this.props.onActiveInitSession}
                      />}
                    />
                    <Route exact path="/app/:slug/downstreams" render={() =>
                      <div className="container">
                        <DeploymentClusters
                          appDetailPage={true}
                          parentWatch={app}
                          parentClusterName={app.name}
                          preparingUpdate={this.state.preparingUpdate}
                          childWatches={app.watches || []}
                          handleAddNewCluster={() => this.handleAddNewClusterClick(app)}
                          onEditApplication={this.onEditApplicationClicked}
                          installLatestVersion={this.makeCurrentRelease}
                          toggleDeleteDeploymentModal={undefined}
                        />
                      </div>
                    } />
                    <Route exact path="/app/:slug/integrations" render={() => <DetailPageIntegrations watch={app} />} />

                    <Route exact path="/app/:slug/tree/:sequence" render={props => <DownstreamTree {...props} />} />

                    <Route exact path="/app/:slug/version-history" render={() =>
                      <WatchVersionHistory
                        watch={app}
                        match={this.props.match}
                        onCheckForUpdates={this.onCheckForUpdates}
                        checkingForUpdates={this.state.checkingForUpdates}
                        checkingUpdateText={checkingUpdateText}
                        handleAddNewCluster={() => this.handleAddNewClusterClick(app)}
                        errorCheckingUpdate={updateError}
                      />
                    } />
                    <Route exact path="/app/:slug/downstreams/:downstreamOwner/:downstreamSlug/version-history" render={() =>
                      <DownstreamWatchVersionHistory
                        watch={app}
                        makeCurrentVersion={this.makeCurrentRelease}
                      />
                    } />
                    <Route exact path="/app/:slug/config" render={() =>
                      <WatchConfig
                        watch={app}
                        onActiveInitSession={this.props.onActiveInitSession}
                      />
                    } />
                    <Route exact path="/app/:slug/troubleshoot" render={() =>
                      <SupportBundleList
                        watch={app}
                      />
                    } />
                    <Route exact path="/app/:slug/troubleshoot/generate" render={() =>
                      <GenerateSupportBundle
                        watch={app}
                      />
                    } />
                    <Route path="/app/:slug/troubleshoot/analyze/:bundleSlug" render={() =>
                      <SupportBundleAnalysis
                        watch={app}
                      />
                    } />
                    <Route exact path="/app/:slug/license" render={() =>
                      <WatchLicense
                        watch={app}
                      />
                    } />
                    <Route component={NotFound} />
                  </Switch>
                </Fragment>
              )
            }
          </div>
        </SidebarLayout>
      </div>
    );
  }
}

export { AppDetailPage };
export default compose(
  withApollo,
  withRouter,
  withTheme,
  graphql(getKotsApp, {
    name: "getKotsAppQuery",
    skip: props => {
      const { slug } = props.match.params;

      // Skip if no variables (user at "/watches" URL)
      if (!slug) {
        return true;
      }

      return false;

    },
    options: props => {
      const { slug } = props.match.params;
      return {
        fetchPolicy: "no-cache",
        variables: {
          slug: slug
        }
      }
    }
  }),
)(AppDetailPage);
