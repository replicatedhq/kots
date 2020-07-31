import React, { Component, Fragment } from "react";
import classNames from "classnames";
import { withRouter, Switch, Route } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import { Helmet } from "react-helmet";
import Modal from "react-modal";

import withTheme from "@src/components/context/withTheme";
import { listDownstreamsForApp } from "@src/queries/AppsQueries";
import { isVeleroInstalled } from "@src/queries/SnapshotQueries";
import { createKotsDownstream } from "../../mutations/AppsMutations";
import { KotsSidebarItem } from "@src/components/watches/WatchSidebarItem";
import { HelmChartSidebarItem } from "@src/components/watches/WatchSidebarItem";
import NotFound from "../static/NotFound";
import Dashboard from "./Dashboard";
import CodeSnippet from "../shared/CodeSnippet";
import DownstreamTree from "../../components/tree/KotsApplicationTree";
import AppVersionHistory from "./AppVersionHistory";
import { isAwaitingResults, Utilities } from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";
import PreflightResultPage from "../PreflightResultPage";
import AppConfig from "./AppConfig";
import AppLicense from "./AppLicense";
import SubNavBar from "@src/components/shared/SubNavBar";
import SidebarLayout from "../layout/SidebarLayout/SidebarLayout";
import SideBar from "../shared/SideBar";
import Loader from "../shared/Loader";
import AppSettings from "./AppSettings";
import AppGitops from "./AppGitops";
import AppSnapshots from "./AppSnapshots";
import AppSnapshotSchedule from "./AppSnapshotSchedule";
import AppSnapshotDetail from "./AppSnapshotDetail";
import AppSnapshotRestore from "./AppSnapshotRestore";
import TroubleshootContainer from "../troubleshoot/TroubleshootContainer";

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
      isBundleUploading: false,
      app: null,
      loadingApp: true,
      getAppJob: new Repeater(),
    }
  }

  componentDidUpdate(_, lastState) {
    const { getThemeState, setThemeState, match, listApps, history } = this.props;
    const { app, loadingApp } = this.state;

    // Used for a fresh reload
    if (history.location.pathname === "/apps") {
      this.checkForFirstApp();
      return;
    }

    // Refetch app info when switching between apps
    if (app && !loadingApp && match.params.slug != app.slug) {
      this.getApp();
      return;
    }

    // Handle updating the theme state when switching apps.
    const currentApp = listApps?.find(w => w.slug === match.params.slug);
    if (currentApp?.iconUri) {
      const { navbarLogo, ...rest } = getThemeState();
      if (navbarLogo === null || navbarLogo !== currentApp.iconUri) {
        setThemeState({
          ...rest,
          navbarLogo: currentApp.iconUri
        });
      }
    }

    // Enforce initial app configuration (if exists)
    if (app !== lastState.app && app) {
      const downstream = app.downstreams?.length && app.downstreams[0];
      if (downstream?.pendingVersions?.length) {
        const firstVersion = downstream.pendingVersions.find(version => version?.sequence === 0);
        if (firstVersion?.status === "pending_config") {
          this.props.history.push(`/${app.slug}/config`);
          return;
        }
      }
    }
  }

  componentWillUnmount() {
    clearInterval(this.interval);
    this.props.clearThemeState();
    this.state.getAppJob.stop();
  }

  makeCurrentRelease = async (upstreamSlug, sequence) => {
    try {
      await fetch(`${window.env.API_ENDPOINT}/app/${upstreamSlug}/sequence/${sequence}/deploy`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "POST",
      });
      this.refetchData();
    } catch(err) {
      console.log(err);
    }
  }

  toggleDisplayDownloadModal = () => {
    this.setState({ displayDownloadCommandModal: !this.state.displayDownloadCommandModal });
  }

  toggleIsBundleUploading = (isUploading) => {
    this.setState({ isBundleUploading: isUploading });
  }

  getApp = async (slug = this.props.match.params.slug) => {
    if (!slug) {
      return;
    }

    try {
      this.setState({ loadingApp: true });

      const res = await fetch(`${window.env.API_ENDPOINT}/apps/app/${slug}`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });
      if (res.ok && res.status == 200) {
        const app = await res.json();
        this.setState({ app, loadingApp: false });
      } else {
        console.log("failed to get app, unexpected status code", res.status);
        this.setState({ loadingApp: false });
      }
    } catch(err) {
      console.log(err);
      this.setState({ loadingApp: false });
    }
  }

  /**
   * Refetch all the data for this component and all its children
   *
   * @return {undefined}
   */
  refetchData = () => {
    this.getApp();
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
      this.getApp(firstApp.slug);
    } else {
      history.replace("/upload-license");
    }
  }

  componentDidMount() {
    const { history } = this.props;

    if (history.location.pathname === "/apps") {
      this.checkForFirstApp();
      return;
    }

    this.getApp();
  }

  render() {
    const {
      match,
      listApps,
      refetchListApps,
      rootDidInitialAppFetch,
      appName,
      isVeleroInstalled
    } = this.props;

    const {
      app,
      displayDownloadCommandModal,
      isBundleUploading
    } = this.state;

    const centeredLoader = (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );

    if (!rootDidInitialAppFetch) {
      return centeredLoader;
    }

    const downstream = app?.downstreams?.length && app.downstreams[0];
    if (downstream?.currentVersion && isAwaitingResults([downstream.currentVersion])) {
      this.state.getAppJob.start(this.getApp, 2000);
    } else {
      this.state.getAppJob.stop();
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
            {!app
              ? centeredLoader
              : (
                <Fragment>
                  <SubNavBar
                    className="flex"
                    activeTab={match.params.tab || "app"}
                    watch={app}
                    isVeleroInstalled={isVeleroInstalled?.isVeleroInstalled}
                  />
                  <Switch>
                    <Route exact path="/app/:slug" render={() =>
                      <Dashboard
                        app={app}
                        cluster={app.downstreams?.length && app.downstreams[0]?.cluster}
                        refetchListApps={refetchListApps}
                        updateCallback={this.refetchData}
                        onActiveInitSession={this.props.onActiveInitSession}
                        makeCurrentVersion={this.makeCurrentRelease}
                        toggleIsBundleUploading={this.toggleIsBundleUploading}
                        isBundleUploading={isBundleUploading}
                        isVeleroInstalled={isVeleroInstalled?.isVeleroInstalled}
                        refreshAppData={this.getApp}
                        snapshotInProgressApps={this.props.snapshotInProgressApps}
                        ping={this.props.ping}
                      />}
                    />

                    <Route exact path="/app/:slug/tree/:sequence?" render={props => <DownstreamTree {...props} app={app} appNameSpace={this.props.appNameSpace} />} />

                    <Route exact path={["/app/:slug/version-history", "/app/:slug/version-history/diff/:firstSequence/:secondSequence"]} render={() =>
                      <AppVersionHistory
                        app={app}
                        match={this.props.match}
                        makeCurrentVersion={this.makeCurrentRelease}
                        updateCallback={this.refetchData}
                        toggleIsBundleUploading={this.toggleIsBundleUploading}
                        isBundleUploading={isBundleUploading}
                        refreshAppData={this.getApp}
                      />
                    } />
                    <Route exact path="/app/:slug/downstreams/:downstreamSlug/version-history/preflight/:sequence" render={props => <PreflightResultPage logo={app.iconUri} {...props} />} />
                    <Route exact path="/app/:slug/config/:sequence?" render={() =>
                      <AppConfig
                        app={app}
                        refreshAppData={this.getApp}
                      />
                    } />
                    <Route path="/app/:slug/troubleshoot" render={() =>
                      <TroubleshootContainer
                        app={app}
                        appName={appName}
                      />
                    } />
                    <Route exact path="/app/:slug/license" render={() =>
                      <AppLicense
                        app={app}
                        syncCallback={this.refetchData}
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
                        refetch={this.getApp}
                      />
                    } />
                    <Route exact path="/app/:slug/snapshots" render={() =>
                      <AppSnapshots
                        app={app}
                        refetch={this.getApp}
                      />
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
                {`kubectl kots download --namespace ${this.props.appNameSpace} --slug ${this.props.match.params.slug}`}
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
  graphql(isVeleroInstalled, {
    name: "isVeleroInstalled",
    options: {
      fetchPolicy: "no-cache"
    }
  }),
  graphql(createKotsDownstream, {
    props: ({ mutate }) => ({
      createKotsDownstream: (appId, clusterId) => mutate({ variables: { appId, clusterId } })
    })
  }),
)(AppDetailPage);
