import React, { Component, Fragment } from "react";
import classNames from "classnames";
import { withRouter, Switch, Route } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import { Helmet } from "react-helmet";
import Modal from "react-modal";
import has from "lodash/has";

import withTheme from "@src/components/context/withTheme";
import { getKotsApp, listDownstreamsForApp } from "@src/queries/AppsQueries";
import { createKotsDownstream, deleteKotsDownstream, deployKotsVersion } from "../../mutations/AppsMutations";
import { KotsSidebarItem } from "@src/components/watches/WatchSidebarItem";
import { HelmChartSidebarItem } from "@src/components/watches/WatchSidebarItem";
import NotFound from "../static/NotFound";
import Dashboard from "./Dashboard";
import CodeSnippet from "../shared/CodeSnippet";
import DownstreamTree from "../../components/tree/KotsApplicationTree";
import AppVersionHistory from "./AppVersionHistory";
import { isAwaitingResults } from "../../utilities/utilities";
import PreflightResultPage from "../PreflightResultPage";
import AppConfig from "./AppConfig";
import AppLicense from "./AppLicense";
import SubNavBar from "@src/components/shared/SubNavBar";
import SidebarLayout from "../layout/SidebarLayout/SidebarLayout";
import SideBar from "../shared/SideBar";
import Loader from "../shared/Loader";
import SupportBundleList from "../troubleshoot/SupportBundleList";
import SupportBundleAnalysis from "../troubleshoot/SupportBundleAnalysis";
import GenerateSupportBundle from "../troubleshoot/GenerateSupportBundle";
import AppSettings from "./AppSettings";
import AppGitops from "./AppGitops";
import AppSnapshots from "./AppSnapshots";
import AppSnapshotSettings from "./AppSnapshotSettings";
import AppSnapshotSchedule from "./AppSnapshotSchedule";
import AppSnapshotDetail from "./AppSnapshotDetail";
import AppSnapshotRestore from "./AppSnapshotRestore";

import "../../scss/components/watches/WatchDetailPage.scss";

class AppDetailPage extends Component {
  constructor(props) {
    super(props);
    this.state = {
      preparingUpdate: "",
      clusterParentSlug: "",
      selectedWatchName: "",
      watchToEdit: {},
      existingDeploymentClusters: [],
      displayDownloadCommandModal: false,
      isBundleUploading: false
    }
  }

  static defaultProps = {
    getKotsAppQuery: {
      loading: true
    }
  }

  componentDidUpdate(lastProps) {
    const { getThemeState, setThemeState, match, listApps, history } = this.props;
    const slug = `${match.params.owner}/${match.params.slug}`;
    const currentWatch = listApps?.find(w => w.slug === slug);

    // Handle updating the app theme state when a watch changes.
    if (currentWatch?.watchIcon) {
      const { navbarLogo, ...rest } = getThemeState();
      if (navbarLogo === null || navbarLogo !== currentWatch.watchIcon) {

        setThemeState({
          ...rest,
          navbarLogo: currentWatch.watchIcon
        });
      }
    }

    // Used for a fresh reload
    if (history.location.pathname === "/apps") {
      this.checkForFirstApp();
    }

    // enforce initial app configuration (if exists)
    const { getKotsAppQuery } = this.props;
    if (getKotsAppQuery?.getKotsApp !== lastProps?.getKotsAppQuery?.getKotsApp && getKotsAppQuery?.getKotsApp) {
      const app = getKotsAppQuery?.getKotsApp;
      const downstream = app.downstreams?.length && app.downstreams[0];
      if (downstream?.pendingVersions?.length) {
        const firstVersion = downstream.pendingVersions.find(version => version?.sequence === 0);
        if (firstVersion?.status === "pending_config") {
          this.props.history.push(`/${app.slug}/config`);
        }
      }
    }
  }

  componentWillUnmount() {
    clearInterval(this.interval);
    this.props.clearThemeState();
  }

  makeCurrentRelease = async (upstreamSlug, sequence, clusterSlug) => {
    await this.props.deployKotsVersion(upstreamSlug, sequence, clusterSlug).then(() => {
      this.refetchGraphQLData();
    })
  }

  toggleDisplayDownloadModal = () => {
    this.setState({ displayDownloadCommandModal: !this.state.displayDownloadCommandModal });
  }

  toggleIsBundleUploading = (isUploading) => {
    this.setState({ isBundleUploading: isUploading });
  }

  createDownstreamForCluster = () => {
    const { clusterParentSlug } = this.state;
    localStorage.setItem("clusterRedirect", `/watch/${clusterParentSlug}/downstreams?add=1`);
    this.props.history.push("/cluster/create");
  }

  handleViewFiles = () => {
    const { slug } = this.props.match.params;
    const currentSequence = this.props.getKotsAppQuery?.getKotsApp?.currentSequence;
    this.props.history.push(`/app/${slug}/tree/${currentSequence}`);
  }

  /**
   * Refetch all the GraphQL data for this component and all its children
   *
   * @return {undefined}
   */
  refetchGraphQLData = () => {
    this.props.getKotsAppQuery.refetch();
    this.props.refetchListApps();
  }

  /**
   *  Runs on mount and on update. Also handles redirect logic
   *  if no apps are found, or the first app is found.
   */
  checkForFirstApp = () => {
    const { history, rootDidInitialAppFetch, listApps } = this.props;
    if (!rootDidInitialAppFetch) {
      return;
    }
    const firstApp = listApps?.find(app => app.name);

    if (firstApp) {
      history.replace(`/app/${firstApp.slug}`);
    } else {
      history.replace("/upload-license");
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
      rootDidInitialAppFetch,
      appName
    } = this.props;
    const {
      displayDownloadCommandModal,
      isBundleUploading
    } = this.state;

    const centeredLoader = (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );

    const app = getKotsAppQuery?.getKotsApp;

    const refreshAppData = getKotsAppQuery.refetch;

    // if there is app, don't render a loader to avoid flickering
    const loading = (getKotsAppQuery?.loading || !rootDidInitialAppFetch) && !app;

    if (!rootDidInitialAppFetch) {
      return centeredLoader;
    }

    const downstream = app?.downstreams?.length && app.downstreams[0];
    if (downstream?.currentVersion && isAwaitingResults([downstream.currentVersion])) {
      getKotsAppQuery?.startPolling(2000);
    } else if (has(getKotsAppQuery, "stopPolling")) {
      getKotsAppQuery?.stopPolling();
    }


    return (
      <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
        <Helmet>
          <title>{`${appName ? `${appName} Admin Console` : "Admin Console"}`}</title>
        </Helmet>
        <SidebarLayout
          className="flex flex1 u-minHeight--full u-overflow--hidden"
          condition={listApps?.length > 1}
          sidebar={(
            <SideBar
              items={listApps?.map((item, idx) => {
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
          <div className="flex-column flex1 u-width--full u-height--full u-overflow--auto">
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
                      <Dashboard
                        app={app}
                        cluster={app.downstreams?.length && app.downstreams[0]?.cluster}
                        refetchListApps={refetchListApps}
                        refetchWatch={this.props.getKotsAppQuery?.refetch}
                        updateCallback={this.refetchGraphQLData}
                        onActiveInitSession={this.props.onActiveInitSession}
                        makeCurrentVersion={this.makeCurrentRelease}
                        toggleIsBundleUploading={this.toggleIsBundleUploading}
                        isBundleUploading={isBundleUploading}
                      />}
                    />

                    <Route exact path="/app/:slug/tree/:sequence" render={props => <DownstreamTree {...props} appNameSpace={this.props.appNameSpace} />} />

                    <Route exact path={["/app/:slug/version-history", "/app/:slug/version-history/diff/:firstSequence/:secondSequence"]} render={() =>
                      <AppVersionHistory
                        app={app}
                        match={this.props.match}
                        makeCurrentVersion={this.makeCurrentRelease}
                        updateCallback={this.refetchGraphQLData}
                        toggleIsBundleUploading={this.toggleIsBundleUploading}
                        isBundleUploading={isBundleUploading}
                        refreshAppData={refreshAppData}
                      />
                    } />
                    <Route exact path="/app/:slug/downstreams/:downstreamSlug/version-history/preflight/:sequence" render={props => <PreflightResultPage logo={app.iconUri} {...props} />} />
                    <Route exact path="/app/:slug/config/:sequence?" render={() =>
                      <AppConfig
                        app={app}
                        refreshAppData={refreshAppData}
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
                      <AppLicense
                        app={app}
                        syncCallback={this.refetchGraphQLData}
                      />
                    } />
                    <Route exact path="/app/:slug/registry-settings" render={() =>
                      <AppSettings
                        app={app}
                      />
                    } />
                    <Route exact path="/app/:slug/gitops" render={() =>
                      <AppGitops
                        app={app}
                        history={this.props.history}
                        refetch={() => this.props.getKotsAppQuery.refetch()}
                      />
                    } />
                    <Route exact path="/app/:slug/snapshots" render={() =>
                      <AppSnapshots
                        app={app}
                        refetch={() => this.props.getKotsAppQuery.refetch()}
                      />
                    } />
                    <Route exact path="/app/:slug/snapshots/settings" render={() =>
                      <AppSnapshotSettings app={app} />
                    } />
                    <Route exact path="/app/:slug/snapshots/schedule" render={() =>
                      <AppSnapshotSchedule app={app} />
                    } />
                    <Route exact path="/app/:slug/snapshots/:id" render={() =>
                      <AppSnapshotDetail app={app} />
                    } />
                    <Route exact path="/app/:slug/snapshots/:id/restore" render={() =>
                      <AppSnapshotRestore app={app} />
                    } />
                    <Route component={NotFound} />
                  </Switch>
                </Fragment>
              )
            }
          </div>
        </SidebarLayout>
        {displayDownloadCommandModal &&
          <Modal
            isOpen={displayDownloadCommandModal}
            onRequestClose={this.toggleDisplayDownloadModal}
            shouldReturnFocusAfterClose={false}
            contentLabel="Download cluster command modal"
            ariaHideApp={false}
            className="DisplayDownloadCommandModal--wrapper Modal"
          >
            <div className="Modal-body">
              <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Download assets</h2>
              <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Run this command in your cluster to download the assets.</p>
              <CodeSnippet
                language="bash"
                canCopy={true}
                onCopyText={<span className="u-color--chateauGreen">Command has been copied to your clipboard</span>}
              >
                kubectl krew install kots
                {`kubectl kots download --namespace ${this.props.appNameSpace} --slug ${this.props.match.params.slug} --dest ~/${this.props.match.params.slug}`}
              </CodeSnippet>
              <div className="u-marginTop--10 flex">
                <button onClick={this.toggleDisplayDownloadModal} className="btn blue primary">Ok, got it!</button>
              </div>
            </div>
          </Modal>
        }
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
  graphql(listDownstreamsForApp, {
    name: "listDownstreamsForAppQuery",
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
  graphql(createKotsDownstream, {
    props: ({ mutate }) => ({
      createKotsDownstream: (appId, clusterId) => mutate({ variables: { appId, clusterId } })
    })
  }),
  graphql(deleteKotsDownstream, {
    props: ({ mutate }) => ({
      deleteKotsDownstream: (slug, clusterId) => mutate({ variables: { slug, clusterId } })
    })
  }),
  graphql(deployKotsVersion, {
    props: ({ mutate }) => ({
      deployKotsVersion: (upstreamSlug, sequence, clusterSlug) => mutate({ variables: { upstreamSlug, sequence, clusterSlug } })
    })
  }),
)(AppDetailPage);
