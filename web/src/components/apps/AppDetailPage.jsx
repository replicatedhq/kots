import React, { Component, Fragment } from "react";
import classNames from "classnames";
import { withRouter, Switch, Route, Redirect } from "react-router-dom";
import { Helmet } from "react-helmet";
import Modal from "react-modal";

import withTheme from "@src/components/context/withTheme";
import { KotsSidebarItem } from "@src/components/watches/WatchSidebarItem";
import { HelmChartSidebarItem } from "@src/components/watches/WatchSidebarItem";
import NotFound from "../static/NotFound";
import Dashboard from "./Dashboard";
import DashboardClassic from "./DashboardClassic";
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
import AppRegistrySettings from "./AppRegistrySettings";
import AppIdentityServiceSettings from "./AppIdentityServiceSettings";
import AppGitops from "./AppGitops";
import TroubleshootContainer from "../troubleshoot/TroubleshootContainer";
import ErrorModal from "../modals/ErrorModal";

import "../../scss/components/watches/WatchDetailPage.scss";
import AppVersionHistoryClassic from "./AppVersionHistoryClassic";

class AppDetailPage extends Component {
  constructor(props) {
    super(props);
    this.state = {
      preparingUpdate: "",
      clusterParentSlug: "",
      selectedWatchName: "",
      watchToEdit: {},
      existingDeploymentClusters: [],
      displayRequiredKotsUpdateModal: false,
      isBundleUploading: false,
      app: null,
      loadingApp: true,
      getAppJob: new Repeater(),
      gettingAppErrMsg: "",
      makingCurrentReleaseErrMsg: "",
      makingCurrentRelease: false,
      displayErrorModal: false,
      isVeleroInstalled: false,
      redeployVersionErrMsg: ""
    }
  }

  componentDidUpdate(_, lastState) {
    const { getThemeState, setThemeState, match, appsList, history } = this.props;
    const { app, loadingApp } = this.state;

    // Used for a fresh reload
    if (history.location.pathname === "/apps") {
      this.checkForFirstApp();
      return;
    }

    // Refetch app info when switching between apps
    if (app && !loadingApp && match.params.slug != app.slug) {
      this.getApp();
      this.checkIsVeleroInstalled();
      return;
    }

    // Handle updating the theme state when switching apps.
    const currentApp = appsList?.find(w => w.slug === match.params.slug);
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

  makeCurrentRelease = async (upstreamSlug, version, isSkipPreflights, continueWithFailedPreflights = false) => {
    try {
      this.setState({ makingCurrentReleaseErrMsg: "" });

      const res = await fetch(`${process.env.API_ENDPOINT}/app/${upstreamSlug}/sequence/${version.sequence}/deploy`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "POST",
        body: JSON.stringify({ 
          isSkipPreflights: isSkipPreflights ,
          continueWithFailedPreflights: continueWithFailedPreflights,
          isCLI: false
        }),
      });
      if (!res.ok && res.status === 409) {
        const response = await res.json();
        this.toggleDisplayRequiredKotsUpdateModal(response);
      }
      if (res.ok && res.status === 204) {
        this.setState({ makingCurrentReleaseErrMsg: "" });
        this.refetchData();
      } else {
        this.setState({
          makingCurrentReleaseErrMsg: `Unable to deploy release ${version.versionLabel}, sequence ${version.sequence}: Unexpected status code: ${res.status}`,
        });
      }
    } catch (err) {
      console.log(err)
      this.setState({
        makingCurrentReleaseErrMsg: err ? `Unable to deploy release ${version.versionLabel}, sequence ${version.sequence}: ${err.message}` : "Something went wrong, please try again.",
      });
    }
  }

  redeployVersion = async (upstreamSlug, version) => {
    try {
      this.setState({ redeployVersionErrMsg: "" });

      const res = await fetch(`${process.env.API_ENDPOINT}/app/${upstreamSlug}/sequence/${version.sequence}/redeploy`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "POST",
      });
      if (res.ok && res.status === 204) {
        this.setState({ redeployVersionErrMsg: "" });
        this.refetchData();
      } else {
        this.setState({
          redeployVersionErrMsg: `Unable to redeploy release ${version.versionLabel}, sequence ${version.sequence}: Unexpected status code: ${res.status}`
        });
      }
    } catch (err) {
      console.log(err)
      this.setState({
        redeployVersionErrMsg: err ? `Unable to deploy release ${version.versionLabel}, sequence ${version.sequence}: ${err.message}` : "Something went wrong, please try again."
      });
    }
  }

  toggleDisplayRequiredKotsUpdateModal = (resObj) => {
    this.setState({
      displayRequiredKotsUpdateModal: !this.state.displayRequiredKotsUpdateModal,
      requiredKotsUpdateObj: resObj,
    });
  }

  toggleIsBundleUploading = (isUploading) => {
    this.setState({ isBundleUploading: isUploading });
  }

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  }

  /**
   * Refetch all the data for this component and all its children
   *
   * @return {undefined}
   */
  refetchData = () => {
    this.getApp();
    this.props.refetchAppsList();
    this.checkIsVeleroInstalled();
  }

  /**
   *  Runs on mount and on update. Also handles redirect logic
   *  if no apps are found, or the first app is found.
   */
  checkForFirstApp = () => {
    const { history, rootDidInitialAppFetch, appsList } = this.props;
    if (!rootDidInitialAppFetch) {
      return;
    }
    const firstApp = appsList?.find(app => app.name);

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
    this.checkIsVeleroInstalled();
  }

  getApp = async (slug = this.props.match.params.slug) => {
    if (!slug) {
      return;
    }

    try {
      this.setState({ loadingApp: true });

      const res = await fetch(`${process.env.API_ENDPOINT}/app/${slug}`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });
      if (res.ok && res.status == 200) {
        const app = await res.json();
        this.setState({ app, loadingApp: false, gettingAppErrMsg: "", displayErrorModal: false });
      } else {
        this.setState({ loadingApp: false, gettingAppErrMsg: `Unexpected status code: ${res.status}`, displayErrorModal: true });
      }
    } catch (err) {
      console.log(err)
      this.setState({ loadingApp: false, gettingAppErrMsg: err ? err.message : "Something went wrong, please try again.", displayErrorModal: true });
    }
  }

  checkIsVeleroInstalled = async () => {
    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/velero`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });
      if (res.ok && res.status == 200) {
        const response = await res.json();
        this.setState({ isVeleroInstalled: response.isVeleroInstalled })
      } else {
        this.setState({ isVeleroInstalled: false });
      }
    } catch (err) {
      console.log(err)
      this.setState({ isVeleroInstalled: false });
    }
  }

  render() {
    const {
      match,
      appsList,
      rootDidInitialAppFetch,
      appName,
    } = this.props;

    const {
      app,
      displayRequiredKotsUpdateModal,
      isBundleUploading,
      requiredKotsUpdateObj,
      gettingAppErrMsg,
      isVeleroInstalled
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
          condition={appsList?.length > 1}
          sidebar={(
            <SideBar
              items={appsList?.map((item, idx) => {
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
                    app={app}
                    isVeleroInstalled={isVeleroInstalled}
                  />
                  <Switch>
                    <Route exact path="/app/:slug" render={() =>
                    app.isNewKotsUiEnabled ?
                      <Dashboard
                        app={app}
                        cluster={app.downstreams?.length && app.downstreams[0]?.cluster}
                        updateCallback={this.refetchData}
                        onActiveInitSession={this.props.onActiveInitSession}
                        toggleIsBundleUploading={this.toggleIsBundleUploading}
                        makeCurrentVersion={this.makeCurrentRelease}
                        isBundleUploading={isBundleUploading}
                        isVeleroInstalled={isVeleroInstalled}
                        refreshAppData={this.getApp}
                        snapshotInProgressApps={this.props.snapshotInProgressApps}
                        ping={this.props.ping}
                      />
                    :
                      <DashboardClassic
                        app={app}
                        cluster={app.downstreams?.length && app.downstreams[0]?.cluster}
                        updateCallback={this.refetchData}
                        onActiveInitSession={this.props.onActiveInitSession}
                        toggleIsBundleUploading={this.toggleIsBundleUploading}
                        isBundleUploading={isBundleUploading}
                        isVeleroInstalled={isVeleroInstalled}
                        refreshAppData={this.getApp}
                        snapshotInProgressApps={this.props.snapshotInProgressApps}
                        ping={this.props.ping}
                      />
                    }
                    />

                    <Route exact path="/app/:slug/tree/:sequence?" render={props => <DownstreamTree {...props} app={app} appNameSpace={this.props.appNameSpace} />} />

                    <Route exact path={["/app/:slug/version-history", "/app/:slug/version-history/diff/:firstSequence/:secondSequence"]} render={() =>
                    app.isNewKotsUiEnabled ?
                      <AppVersionHistory
                        app={app}
                        match={this.props.match}
                        makeCurrentVersion={this.makeCurrentRelease}
                        makingCurrentVersionErrMsg={this.state.makingCurrentReleaseErrMsg}
                        updateCallback={this.refetchData}
                        toggleIsBundleUploading={this.toggleIsBundleUploading}
                        isBundleUploading={isBundleUploading}
                        refreshAppData={this.getApp}
                        displayErrorModal={this.state.displayErrorModal}
                        toggleErrorModal={this.toggleErrorModal}
                        makingCurrentRelease={this.state.makingCurrentRelease}
                        redeployVersion={this.redeployVersion}
                        redeployVersionErrMsg={this.state.redeployVersionErrMsg}
                      />
                    :
                      <AppVersionHistoryClassic
                        app={app}
                        match={this.props.match}
                        makeCurrentVersion={this.makeCurrentRelease}
                        makingCurrentVersionErrMsg={this.state.makingCurrentReleaseErrMsg}
                        updateCallback={this.refetchData}
                        toggleIsBundleUploading={this.toggleIsBundleUploading}
                        isBundleUploading={isBundleUploading}
                        refreshAppData={this.getApp}
                        displayErrorModal={this.state.displayErrorModal}
                        toggleErrorModal={this.toggleErrorModal}
                        makingCurrentRelease={this.state.makingCurrentRelease}
                        redeployVersion={this.redeployVersion}
                        redeployVersionErrMsg={this.state.redeployVersionErrMsg}
                      />
                    } />
                    <Route exact path="/app/:slug/downstreams/:downstreamSlug/version-history/preflight/:sequence" render={props => <PreflightResultPage logo={app.iconUri} {...props} />} />
                    <Route exact path="/app/:slug/config/:sequence?" render={() =>
                      <AppConfig
                        app={app}
                        refreshAppData={this.getApp}
                        fromLicenseFlow={false}
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
                      <AppRegistrySettings
                        app={app}
                        updateCallback={this.refetchData}
                      />
                    } />
                    <Route exact path="/app/:slug/gitops" render={() =>
                      <AppGitops
                        app={app}
                        history={this.props.history}
                        refetch={this.getApp}
                      />
                    } />
                    {app.isAppIdentityServiceSupported &&
                      <Route exact path="/app/:slug/access" render={() =>
                        <AppIdentityServiceSettings
                          app={app}
                          refetch={this.getApp}
                        />
                      } />
                    }
                    {/* snapshots redirects */}
                    <Redirect exact from="/app/:slug/snapshots" to="/snapshots/partial/:slug" />
                    <Redirect exact from="/app/:slug/snapshots/schedule" to="/snapshots/settings?:slug" />
                    <Redirect exact from="/app/:slug/snapshots/:id" to="/snapshots/partial/:slug/:id" />
                    <Redirect exact from="/app/:slug/snapshots/:id/restore" to="/snapshots/partial/:slug/:id/restore" />

                    <Route component={NotFound} />
                  </Switch>
                </Fragment>
              )
            }
          </div>
        </SidebarLayout>
        {displayRequiredKotsUpdateModal &&
          <Modal
            isOpen={displayRequiredKotsUpdateModal}
            onRequestClose={() => this.toggleDisplayRequiredKotsUpdateModal(null)}
            shouldReturnFocusAfterClose={false}
            contentLabel="Required KOTS Update modal"
            ariaHideApp={false}
            className="DisplayRequiredKotsUpdateModal--wrapper Modal"
          >
            <div className="Modal-body">
              <h2 className="u-fontSize--largest u-textColor--primary u-fontWeight--bold u-lineHeight--normal">You must update KOTS to deploy this version</h2>
              <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">This version of {app?.name} requires a version of KOTS that is different from what you currently have installed.</p>
                <p className="u-fontSize--normal u-textColor--error u-fontWeight--medium u-lineHeight--normal u-marginBottom--20">{requiredKotsUpdateObj?.error}</p>
              <div className="u-marginTop--10 flex">
                <button onClick={() => this.toggleDisplayRequiredKotsUpdateModal(null)} className="btn blue primary">Ok, got it!</button>
              </div>
            </div>
          </Modal>
        }
        {gettingAppErrMsg &&
          <ErrorModal
            errorModal={this.state.displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            errMsg={gettingAppErrMsg}
            tryAgain={() => this.getApp(this.props.match.params.slug)}
            err="Failed to get application"
            loading={this.state.loadingApp}
          />}
      </div>
    );
  }
}

export { AppDetailPage };
export default withTheme(withRouter(AppDetailPage));
