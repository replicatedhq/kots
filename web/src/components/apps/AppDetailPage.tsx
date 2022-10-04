<<<<<<< HEAD
import React, { Fragment, useReducer, useEffect } from "react";
import classNames from "classnames";
import {
  Switch,
  Route,
  Redirect,
  useHistory,
  useParams,
} from "react-router-dom";
import Modal from "react-modal";
import { useTheme } from "@src/components/context/withTheme";
=======
import React, { Component, Fragment } from "react";
import classNames from "classnames";
import { withRouter, Switch, Route, Redirect } from "react-router-dom";
import Modal from "react-modal";
import withTheme from "@src/components/context/withTheme";
>>>>>>> c78ee9a2d0570aea46b7b4c5645388316ca8ce72
import { KotsSidebarItem } from "@src/components/watches/WatchSidebarItem";
import { HelmChartSidebarItem } from "@src/components/watches/WatchSidebarItem";
import NotFound from "../static/NotFound";
import Dashboard from "./Dashboard";
import DownstreamTree from "../../components/tree/KotsApplicationTree";
import AppVersionHistory from "./AppVersionHistory";
import { isAwaitingResults, Utilities } from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";
import PreflightResultPage from "../PreflightResultPage";
import AppConfig from "../../features/AppConfig/components/AppConfig";
import AppLicense from "./AppLicense";
import SubNavBar from "@src/components/shared/SubNavBar";
import SidebarLayout from "../layout/SidebarLayout/SidebarLayout";
import SideBar from "../shared/SideBar";
import Loader from "../shared/Loader";
import AppRegistrySettings from "./AppRegistrySettings";
import AppIdentityServiceSettings from "./AppIdentityServiceSettings";
import TroubleshootContainer from "../troubleshoot/TroubleshootContainer";
import ErrorModal from "../modals/ErrorModal";
<<<<<<< HEAD
import { useCurrentApp } from "@features/App";
=======
>>>>>>> c78ee9a2d0570aea46b7b4c5645388316ca8ce72

import "../../scss/components/watches/WatchDetailPage.scss";

// Types
<<<<<<< HEAD
import { App, Metadata, KotsParams, Version } from "@types";

type Props = {
  adminConsoleMetadata?: Metadata;
  appsList: App[];
  appNameSpace: string | null;
  appName: string | null;
=======
import { RouteComponentProps } from "react-router";
import { App, Metadata, KotsParams, ThemeState, Version } from "@types";

type Props = {
  adminConsoleMetadata: Metadata;
  appsList: App[];
  appNameSpace: boolean;
  appName: string;
  clearThemeState: () => void;
  getThemeState: () => ThemeState;
>>>>>>> c78ee9a2d0570aea46b7b4c5645388316ca8ce72
  isHelmManaged: boolean;
  onActiveInitSession: (session: string) => void;
  ping: () => void;
  refetchAppsList: () => void;
  refetchAppMetadata: () => void;
  rootDidInitialAppFetch: boolean;
<<<<<<< HEAD
  snapshotInProgressApps: string[];
};
=======
  setThemeState: (theme: ThemeState) => void;
  snapshotInProgressApps: boolean;
} & RouteComponentProps<KotsParams>;
>>>>>>> c78ee9a2d0570aea46b7b4c5645388316ca8ce72

type State = {
  app: App | null;
  checkForFirstAppJob: Repeater;
  clusterParentSlug: string;
  displayErrorModal: boolean;
  displayRequiredKotsUpdateModal: boolean;
  getAppJob: Repeater;
  gettingAppErrMsg: string;
  isBundleUploading: boolean;
  isVeleroInstalled: boolean;
  loadingApp: boolean;
  makingCurrentRelease: boolean;
  makingCurrentReleaseErrMsg: string;
  preparingUpdate: string;
  requiredKotsUpdateMessage: string;
  redeployVersionErrMsg: string;
  selectedWatchName: string;
};

<<<<<<< HEAD
function AppDetailPage(props: Props) {
  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
=======
class AppDetailPage extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
>>>>>>> c78ee9a2d0570aea46b7b4c5645388316ca8ce72
      app: null,
      checkForFirstAppJob: new Repeater(),
      clusterParentSlug: "",
      displayErrorModal: false,
      displayRequiredKotsUpdateModal: false,
      getAppJob: new Repeater(),
      gettingAppErrMsg: "",
      isBundleUploading: false,
      isVeleroInstalled: false,
      loadingApp: true,
      makingCurrentRelease: false,
      makingCurrentReleaseErrMsg: "",
      preparingUpdate: "",
      redeployVersionErrMsg: "",
      requiredKotsUpdateMessage: "",
      selectedWatchName: "",
<<<<<<< HEAD
    }
  );

  const history = useHistory();
  const params = useParams<KotsParams>();
  const { currentApp } = useCurrentApp();
  const theme = useTheme();

  const toggleDisplayRequiredKotsUpdateModal = (message: string) => {
    setState({
      displayRequiredKotsUpdateModal: !state.displayRequiredKotsUpdateModal,
      requiredKotsUpdateMessage: message,
    });
  };

  const toggleIsBundleUploading = (isUploading: boolean) => {
    setState({ isBundleUploading: isUploading });
  };

  const toggleErrorModal = () => {
    setState({ displayErrorModal: !state.displayErrorModal });
  };

  const getApp = async (slug = params.slug) => {
    if (!slug) {
      return;
    }

    try {
      setState({ loadingApp: true });

      const res = await fetch(`${process.env.API_ENDPOINT}/app/${slug}`, {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });
      if (res.ok && res.status == 200) {
        const app = await res.json();
        setState({
          app,
          loadingApp: false,
          gettingAppErrMsg: "",
          displayErrorModal: false,
        });
      } else {
        setState({
          loadingApp: false,
          gettingAppErrMsg: `Unexpected status code: ${res.status}`,
          displayErrorModal: true,
        });
      }
    } catch (err) {
      console.log(err);
      if (err instanceof Error) {
        setState({
          loadingApp: false,
          gettingAppErrMsg: err.message,
          displayErrorModal: true,
        });
      } else {
        setState({
          loadingApp: false,
          gettingAppErrMsg: "Something went wrong, please try again.",
          displayErrorModal: true,
        });
      }
    }
  };

  const checkIsVeleroInstalled = async () => {
    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/velero`, {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });
      if (res.ok && res.status == 200) {
        const response = await res.json();
        setState({ isVeleroInstalled: response.isVeleroInstalled });
      } else {
        setState({ isVeleroInstalled: false });
      }
    } catch (err) {
      console.log(err);
      setState({ isVeleroInstalled: false });
    }
  };

  /**
   * Refetch all the data for this component and all its children
   *
   * @return {undefined}
   */
  const refetchData = () => {
    getApp();
    props.refetchAppsList();
    props.refetchAppMetadata();
    checkIsVeleroInstalled();
  };

  const makeCurrentRelease = async (
=======
    };
  }

  componentDidUpdate(_: Props, lastState: State) {
    const { getThemeState, setThemeState, match, appsList, history } =
      this.props;
    const { app, loadingApp } = this.state;

    // Used for a fresh reload
    if (history.location.pathname === "/apps") {
      this.checkForFirstApp();
      // updates state but does not cause infinite loop because app navigates away from /apps
      return;
    }

    // Refetch app info when switching between apps
    if (app && !loadingApp && match.params.slug != app.slug) {
      this.getApp();
      this.checkIsVeleroInstalled();
      return;
    }

    // Handle updating the theme state when switching apps.
    const currentApp = appsList?.find((w) => w.slug === match.params.slug);
    if (currentApp?.iconUri) {
      const { navbarLogo, ...rest } = getThemeState();
      if (navbarLogo === null || navbarLogo !== currentApp.iconUri) {
        setThemeState({
          ...rest,
          navbarLogo: currentApp.iconUri,
        });
      }
    }

    // Enforce initial app configuration (if exists)
    if (app !== lastState.app && app) {
      const downstream = app?.downstream;
      if (downstream?.pendingVersions?.length) {
        const firstVersion = downstream.pendingVersions.find(
          (version) => version?.sequence === 0
        );
        if (firstVersion?.status === "pending_config") {
          this.props.history.push(`/${app.slug}/config`);
          return;
        }
      }
    }
  }

  componentWillUnmount() {
    this.props.clearThemeState();
    this.state.getAppJob.stop();
    this.state.checkForFirstAppJob?.stop?.();
  }

  makeCurrentRelease = async (
>>>>>>> c78ee9a2d0570aea46b7b4c5645388316ca8ce72
    upstreamSlug: string,
    version: Version,
    isSkipPreflights: boolean,
    continueWithFailedPreflights = false
  ) => {
    try {
<<<<<<< HEAD
      setState({ makingCurrentReleaseErrMsg: "" });
=======
      this.setState({ makingCurrentReleaseErrMsg: "" });
>>>>>>> c78ee9a2d0570aea46b7b4c5645388316ca8ce72

      const res = await fetch(
        `${process.env.API_ENDPOINT}/app/${upstreamSlug}/sequence/${version.sequence}/deploy`,
        {
          headers: {
            Authorization: Utilities.getToken(),
            "Content-Type": "application/json",
          },
          method: "POST",
          body: JSON.stringify({
            isSkipPreflights: isSkipPreflights,
            continueWithFailedPreflights: continueWithFailedPreflights,
            isCLI: false,
          }),
        }
      );
      if (res.ok && res.status < 300) {
<<<<<<< HEAD
        setState({ makingCurrentReleaseErrMsg: "" });
        refetchData();
      } else {
        const response = await res.json();
        setState({
=======
        this.setState({ makingCurrentReleaseErrMsg: "" });
        this.refetchData();
      } else {
        const response = await res.json();
        this.setState({
>>>>>>> c78ee9a2d0570aea46b7b4c5645388316ca8ce72
          makingCurrentReleaseErrMsg: `Unable to deploy release ${version.versionLabel}, sequence ${version.sequence}: ${response.error}`,
        });
      }
    } catch (err) {
      console.log(err);
      if (err instanceof Error) {
<<<<<<< HEAD
        setState({
          makingCurrentReleaseErrMsg: `Unable to deploy release ${version.versionLabel}, sequence ${version.sequence}: ${err.message}`,
        });
      } else {
        setState({
=======
        this.setState({
          makingCurrentReleaseErrMsg: `Unable to deploy release ${version.versionLabel}, sequence ${version.sequence}: ${err.message}`,
        });
      } else {
        this.setState({
>>>>>>> c78ee9a2d0570aea46b7b4c5645388316ca8ce72
          makingCurrentReleaseErrMsg: "Something went wrong, please try again.",
        });
      }
    }
  };

<<<<<<< HEAD
  const redeployVersion = async (upstreamSlug: string, version: Version) => {
    try {
      setState({ redeployVersionErrMsg: "" });
=======
  redeployVersion = async (upstreamSlug: string, version: Version) => {
    try {
      this.setState({ redeployVersionErrMsg: "" });
>>>>>>> c78ee9a2d0570aea46b7b4c5645388316ca8ce72

      const res = await fetch(
        `${process.env.API_ENDPOINT}/app/${upstreamSlug}/sequence/${version.sequence}/redeploy`,
        {
          headers: {
            Authorization: Utilities.getToken(),
            "Content-Type": "application/json",
          },
          method: "POST",
        }
      );
      if (res.ok && res.status === 204) {
<<<<<<< HEAD
        setState({ redeployVersionErrMsg: "" });
        refetchData();
      } else {
        setState({
=======
        this.setState({ redeployVersionErrMsg: "" });
        this.refetchData();
      } else {
        this.setState({
>>>>>>> c78ee9a2d0570aea46b7b4c5645388316ca8ce72
          redeployVersionErrMsg: `Unable to redeploy release ${version.versionLabel}, sequence ${version.sequence}: Unexpected status code: ${res.status}`,
        });
      }
    } catch (err) {
      console.log(err);
      if (err instanceof Error) {
<<<<<<< HEAD
        setState({
          redeployVersionErrMsg: `Unable to deploy release ${version.versionLabel}, sequence ${version.sequence}: ${err.message}`,
        });
      } else {
        setState({
=======
        this.setState({
          redeployVersionErrMsg: `Unable to deploy release ${version.versionLabel}, sequence ${version.sequence}: ${err.message}`,
        });
      } else {
        this.setState({
>>>>>>> c78ee9a2d0570aea46b7b4c5645388316ca8ce72
          redeployVersionErrMsg: "Something went wrong, please try again.",
        });
      }
    }
  };

<<<<<<< HEAD
=======
  toggleDisplayRequiredKotsUpdateModal = (message: string) => {
    this.setState({
      displayRequiredKotsUpdateModal:
        !this.state.displayRequiredKotsUpdateModal,
      requiredKotsUpdateMessage: message,
    });
  };

  toggleIsBundleUploading = (isUploading: boolean) => {
    this.setState({ isBundleUploading: isUploading });
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  };

  /**
   * Refetch all the data for this component and all its children
   *
   * @return {undefined}
   */
  refetchData = () => {
    this.getApp();
    this.props.refetchAppsList();
    this.props.refetchAppMetadata();
    this.checkIsVeleroInstalled();
  };

>>>>>>> c78ee9a2d0570aea46b7b4c5645388316ca8ce72
  /**
   *  Runs on mount and on update. Also handles redirect logic
   *  if no apps are found, or the first app is found.
   */
<<<<<<< HEAD
  const checkForFirstApp = async () => {
    const { rootDidInitialAppFetch, appsList } = props;
    if (!rootDidInitialAppFetch) {
      return;
    }
    state.checkForFirstAppJob?.stop?.();
=======
  checkForFirstApp = async () => {
    const { history, rootDidInitialAppFetch, appsList } = this.props;
    if (!rootDidInitialAppFetch) {
      return;
    }
    this.state.checkForFirstAppJob?.stop?.();
>>>>>>> c78ee9a2d0570aea46b7b4c5645388316ca8ce72
    const firstApp = appsList?.find((app) => app.name);

    if (firstApp) {
      history.replace(`/app/${firstApp.slug}`);
<<<<<<< HEAD
      getApp(firstApp.slug);
    } else if (props.isHelmManaged) {
=======
      this.getApp(firstApp.slug);
    } else if (this.props.isHelmManaged) {
>>>>>>> c78ee9a2d0570aea46b7b4c5645388316ca8ce72
      history.replace("/install-with-helm");
    } else {
      history.replace("/upload-license");
    }
  };

<<<<<<< HEAD
  // Enforce initial app configuration (if exists)
  useEffect(() => {
    const { app } = state;

    // Refetch app info when switching between apps
    if (app && !state.loadingApp && params.slug != app.slug) {
      getApp();
      checkIsVeleroInstalled();
      return;
    }

    // Handle updating the theme state when switching apps.
    // Used for a fresh reload
    if (history.location.pathname === "/apps") {
      checkForFirstApp();
      // updates state but does not cause infinite loop because app navigates away from /apps
      return;
    }

    if (app) {
      const downstream = app?.downstream;
      if (downstream?.pendingVersions?.length) {
        const firstVersion = downstream.pendingVersions.find(
          (version) => version?.sequence === 0
        );
        if (firstVersion?.status === "pending_config") {
          history.push(`/${app.slug}/config`);
          return;
        }
      }
    }
  }, [state.app]);

  useEffect(() => {
    if (history.location.pathname === "/apps") {
      state.checkForFirstAppJob.start(checkForFirstApp, 2000);
      return;
    }
    getApp();
    checkIsVeleroInstalled();
    return () => {
      theme.clearThemeState();
      state.getAppJob.stop();
      state.checkForFirstAppJob?.stop?.();
    };
  }, []);

  // Handle updating the theme state when switching apps.
  useEffect(() => {
    if (currentApp?.iconUri) {
      const { navbarLogo, ...rest } = theme.getThemeState();
      if (navbarLogo === null || navbarLogo !== currentApp.iconUri) {
        theme.setThemeState({
          ...rest,
          navbarLogo: currentApp.iconUri,
        });
      }
    }
  }, [currentApp]);

  const { appsList, rootDidInitialAppFetch, appName } = props;

  const {
    app,
    displayRequiredKotsUpdateModal,
    isBundleUploading,
    requiredKotsUpdateMessage,
    gettingAppErrMsg,
    isVeleroInstalled,
  } = state;

  const centeredLoader = (
    <div className="flex-column flex1 alignItems--center justifyContent--center">
      <Loader size="60" />
    </div>
  );

  if (!rootDidInitialAppFetch) {
    return centeredLoader;
  }

  // poll version status if it's awaiting results
  const downstream = app?.downstream;
  if (
    downstream?.currentVersion &&
    isAwaitingResults([downstream.currentVersion])
  ) {
    state.getAppJob.start(getApp, 2000);
  } else {
    state.getAppJob.stop();
  }

  return (
    <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
      <SidebarLayout
        className="flex flex1 u-minHeight--full u-overflow--hidden"
        condition={appsList?.length > 1}
        sidebar={
          <SideBar
            items={appsList?.map((item, idx) => {
              let sidebarItemNode;
              if (item.name) {
                const slugFromRoute = params.slug;
                sidebarItemNode = (
                  <KotsSidebarItem
                    key={idx}
                    className={classNames({
                      selected:
                        item.slug === slugFromRoute && params.owner !== "helm",
                    })}
                    app={item}
                  />
                );
              } else if (item.helmName) {
                sidebarItemNode = (
                  <HelmChartSidebarItem
                    key={idx}
                    className={classNames({
                      selected: item.id === params.slug,
                    })}
                    helmChart={item}
                  />
                );
              }
              return sidebarItemNode;
            })}
          />
        }
      >
        <div className="flex-column flex1 u-width--full u-height--full u-overflow--auto">
          {!app ? (
            centeredLoader
          ) : (
            <Fragment>
              <SubNavBar
                className="flex"
                activeTab={params.tab || "app"}
                app={app}
                isVeleroInstalled={isVeleroInstalled}
                isHelmManaged={props.isHelmManaged}
              />
              <Switch>
                <Route
                  exact
                  path="/app/:slug"
                  render={() => (
                    <Dashboard
                      app={app}
                      cluster={app.downstream?.cluster}
                      updateCallback={refetchData}
                      onActiveInitSession={props.onActiveInitSession}
                      toggleIsBundleUploading={toggleIsBundleUploading}
                      makeCurrentVersion={makeCurrentRelease}
                      redeployVersion={redeployVersion}
                      redeployVersionErrMsg={state.redeployVersionErrMsg}
                      isBundleUploading={isBundleUploading}
                      isVeleroInstalled={isVeleroInstalled}
                      refreshAppData={getApp}
                      snapshotInProgressApps={props.snapshotInProgressApps}
                      ping={props.ping}
                      isHelmManaged={props.isHelmManaged}
                    />
                  )}
                />

                <Route
                  exact
                  path="/app/:slug/tree/:sequence?"
                  render={(renderProps) => (
                    <DownstreamTree
                      {...renderProps}
                      app={app}
                      appNameSpace={props.appNameSpace}
                      isHelmManaged={props.isHelmManaged}
                    />
                  )}
                />

                <Route
                  exact
                  path={[
                    "/app/:slug/version-history",
                    "/app/:slug/version-history/diff/:firstSequence/:secondSequence",
                  ]}
                  render={() => (
                    <AppVersionHistory
                      app={app}
                      match={{ match: { params: params } }}
                      makeCurrentVersion={makeCurrentRelease}
                      makingCurrentVersionErrMsg={
                        state.makingCurrentReleaseErrMsg
                      }
                      updateCallback={refetchData}
                      toggleIsBundleUploading={toggleIsBundleUploading}
                      isBundleUploading={isBundleUploading}
                      isHelmManaged={props.isHelmManaged}
                      refreshAppData={getApp}
                      displayErrorModal={state.displayErrorModal}
                      toggleErrorModal={toggleErrorModal}
                      makingCurrentRelease={state.makingCurrentRelease}
                      redeployVersion={redeployVersion}
                      redeployVersionErrMsg={state.redeployVersionErrMsg}
                      adminConsoleMetadata={props.adminConsoleMetadata}
                    />
                  )}
                />
                <Route
                  exact
                  path="/app/:slug/downstreams/:downstreamSlug/version-history/preflight/:sequence"
                  render={(renderProps) => (
                    <PreflightResultPage
                      logo={app.iconUri}
                      app={app}
                      {...renderProps}
                    />
                  )}
                />
                <Route
                  exact
                  path="/app/:slug/config/:sequence?"
                  render={() => (
                    <AppConfig
                      app={app}
                      refreshAppData={getApp}
                      fromLicenseFlow={false}
                      isHelmManaged={props.isHelmManaged}
                    />
                  )}
                />
                <Route
                  path="/app/:slug/troubleshoot"
                  render={() => (
                    <TroubleshootContainer app={app} appName={appName} />
                  )}
                />
                <Route
                  exact
                  path="/app/:slug/license"
                  render={() => (
                    <AppLicense
                      app={app}
                      syncCallback={refetchData}
                      changeCallback={refetchData}
                    />
                  )}
                />
                <Route
                  exact
                  path="/app/:slug/registry-settings"
                  render={() => (
                    <AppRegistrySettings
                      app={app}
                      updateCallback={refetchData}
                    />
                  )}
                />
                {true && (
                  <Route
                    exact
                    path="/app/:slug/access"
                    render={() => (
                      <AppIdentityServiceSettings app={app} refetch={getApp} />
                    )}
                  />
                )}
                {/* snapshots redirects */}
                <Redirect
                  exact
                  from="/app/:slug/snapshots"
                  to="/snapshots/partial/:slug"
                />
                <Redirect
                  exact
                  from="/app/:slug/snapshots/schedule"
                  to="/snapshots/settings?:slug"
                />
                <Redirect
                  exact
                  from="/app/:slug/snapshots/:id"
                  to="/snapshots/partial/:slug/:id"
                />
                <Redirect
                  exact
                  from="/app/:slug/snapshots/:id/restore"
                  to="/snapshots/partial/:slug/:id/restore"
                />

                <Route component={NotFound} />
              </Switch>
            </Fragment>
          )}
        </div>
      </SidebarLayout>
      {displayRequiredKotsUpdateModal && (
        <Modal
          isOpen={displayRequiredKotsUpdateModal}
          onRequestClose={() => toggleDisplayRequiredKotsUpdateModal("")}
          shouldReturnFocusAfterClose={false}
          contentLabel="Required KOTS Update modal"
          ariaHideApp={false}
          className="DisplayRequiredKotsUpdateModal--wrapper Modal"
        >
          <div className="Modal-body">
            <h2 className="u-fontSize--largest u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
              You must update KOTS to deploy this version
            </h2>
            <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">
              This version of {app?.name} requires a version of KOTS that is
              different from what you currently have installed.
            </p>
            <p className="u-fontSize--normal u-textColor--error u-fontWeight--medium u-lineHeight--normal u-marginBottom--20">
              {requiredKotsUpdateMessage}
            </p>
            <div className="u-marginTop--10 flex">
              <button
                onClick={() => toggleDisplayRequiredKotsUpdateModal("")}
                className="btn blue primary"
              >
                Ok, got it!
              </button>
            </div>
          </div>
        </Modal>
      )}
      {gettingAppErrMsg && (
        <ErrorModal
          errorModal={state.displayErrorModal}
          toggleErrorModal={toggleErrorModal}
          errMsg={gettingAppErrMsg}
          tryAgain={() => getApp(params.slug)}
          err="Failed to get application"
          loading={state.loadingApp}
        />
      )}
    </div>
  );
}

export { AppDetailPage };
=======
  componentDidMount() {
    const { history } = this.props;

    if (history.location.pathname === "/apps") {
      this.state.checkForFirstAppJob.start(this.checkForFirstApp, 2000);
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
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });
      if (res.ok && res.status == 200) {
        const app = await res.json();
        this.setState({
          app,
          loadingApp: false,
          gettingAppErrMsg: "",
          displayErrorModal: false,
        });
      } else {
        this.setState({
          loadingApp: false,
          gettingAppErrMsg: `Unexpected status code: ${res.status}`,
          displayErrorModal: true,
        });
      }
    } catch (err) {
      console.log(err);
      if (err instanceof Error) {
        this.setState({
          loadingApp: false,
          gettingAppErrMsg: err.message,
          displayErrorModal: true,
        });
      } else {
        this.setState({
          loadingApp: false,
          gettingAppErrMsg: "Something went wrong, please try again.",
          displayErrorModal: true,
        });
      }
    }
  };

  checkIsVeleroInstalled = async () => {
    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/velero`, {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });
      if (res.ok && res.status == 200) {
        const response = await res.json();
        this.setState({ isVeleroInstalled: response.isVeleroInstalled });
      } else {
        this.setState({ isVeleroInstalled: false });
      }
    } catch (err) {
      console.log(err);
      this.setState({ isVeleroInstalled: false });
    }
  };

  render() {
    const { match, appsList, rootDidInitialAppFetch, appName } = this.props;

    const {
      app,
      displayRequiredKotsUpdateModal,
      isBundleUploading,
      requiredKotsUpdateMessage,
      gettingAppErrMsg,
      isVeleroInstalled,
    } = this.state;

    const centeredLoader = (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );

    if (!rootDidInitialAppFetch) {
      return centeredLoader;
    }

    // poll version status if it's awaiting results
    const downstream = app?.downstream;
    if (
      downstream?.currentVersion &&
      isAwaitingResults([downstream.currentVersion])
    ) {
      this.state.getAppJob.start(this.getApp, 2000);
    } else {
      this.state.getAppJob.stop();
    }

    return (
      <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
        <SidebarLayout
          className="flex flex1 u-minHeight--full u-overflow--hidden"
          condition={appsList?.length > 1}
          sidebar={
            <SideBar
              items={appsList?.map((item, idx) => {
                let sidebarItemNode;
                if (item.name) {
                  const slugFromRoute = match.params.slug;
                  sidebarItemNode = (
                    <KotsSidebarItem
                      key={idx}
                      className={classNames({
                        selected:
                          item.slug === slugFromRoute &&
                          match.params.owner !== "helm",
                      })}
                      app={item}
                    />
                  );
                } else if (item.helmName) {
                  sidebarItemNode = (
                    <HelmChartSidebarItem
                      key={idx}
                      className={classNames({
                        selected: item.id === match.params.slug,
                      })}
                      helmChart={item}
                    />
                  );
                }
                return sidebarItemNode;
              })}
            />
          }
        >
          <div className="flex-column flex1 u-width--full u-height--full u-overflow--auto">
            {!app ? (
              centeredLoader
            ) : (
              <Fragment>
                <SubNavBar
                  className="flex"
                  activeTab={match.params.tab || "app"}
                  app={app}
                  isVeleroInstalled={isVeleroInstalled}
                  isHelmManaged={this.props.isHelmManaged}
                />
                <Switch>
                  <Route
                    exact
                    path="/app/:slug"
                    render={() => (
                      <Dashboard
                        app={app}
                        cluster={app.downstream?.cluster}
                        updateCallback={this.refetchData}
                        onActiveInitSession={this.props.onActiveInitSession}
                        toggleIsBundleUploading={this.toggleIsBundleUploading}
                        makeCurrentVersion={this.makeCurrentRelease}
                        redeployVersion={this.redeployVersion}
                        redeployVersionErrMsg={this.state.redeployVersionErrMsg}
                        isBundleUploading={isBundleUploading}
                        isVeleroInstalled={isVeleroInstalled}
                        refreshAppData={this.getApp}
                        snapshotInProgressApps={
                          this.props.snapshotInProgressApps
                        }
                        ping={this.props.ping}
                        isHelmManaged={this.props.isHelmManaged}
                      />
                    )}
                  />

                  <Route
                    exact
                    path="/app/:slug/tree/:sequence?"
                    render={(props) => (
                      <DownstreamTree
                        {...props}
                        app={app}
                        appNameSpace={this.props.appNameSpace}
                        isHelmManaged={this.props.isHelmManaged}
                      />
                    )}
                  />

                  <Route
                    exact
                    path={[
                      "/app/:slug/version-history",
                      "/app/:slug/version-history/diff/:firstSequence/:secondSequence",
                    ]}
                    render={() => (
                      <AppVersionHistory
                        app={app}
                        match={this.props.match}
                        makeCurrentVersion={this.makeCurrentRelease}
                        makingCurrentVersionErrMsg={
                          this.state.makingCurrentReleaseErrMsg
                        }
                        updateCallback={this.refetchData}
                        toggleIsBundleUploading={this.toggleIsBundleUploading}
                        isBundleUploading={isBundleUploading}
                        isHelmManaged={this.props.isHelmManaged}
                        refreshAppData={this.getApp}
                        displayErrorModal={this.state.displayErrorModal}
                        toggleErrorModal={this.toggleErrorModal}
                        makingCurrentRelease={this.state.makingCurrentRelease}
                        redeployVersion={this.redeployVersion}
                        redeployVersionErrMsg={this.state.redeployVersionErrMsg}
                        adminConsoleMetadata={this.props.adminConsoleMetadata}
                      />
                    )}
                  />
                  <Route
                    exact
                    path="/app/:slug/downstreams/:downstreamSlug/version-history/preflight/:sequence"
                    render={(props) => (
                      <PreflightResultPage
                        logo={app.iconUri}
                        app={app}
                        {...props}
                      />
                    )}
                  />
                  <Route
                    exact
                    path="/app/:slug/config/:sequence?"
                    render={() => (
                      <AppConfig
                        app={app}
                        refreshAppData={this.getApp}
                        fromLicenseFlow={false}
                        isHelmManaged={this.props.isHelmManaged}
                      />
                    )}
                  />
                  <Route
                    path="/app/:slug/troubleshoot"
                    render={() => (
                      <TroubleshootContainer app={app} appName={appName} />
                    )}
                  />
                  <Route
                    exact
                    path="/app/:slug/license"
                    render={() => (
                      <AppLicense
                        app={app}
                        syncCallback={this.refetchData}
                        changeCallback={this.refetchData}
                      />
                    )}
                  />
                  <Route
                    exact
                    path="/app/:slug/registry-settings"
                    render={() => (
                      <AppRegistrySettings
                        app={app}
                        updateCallback={this.refetchData}
                      />
                    )}
                  />
                  {true && (
                    <Route
                      exact
                      path="/app/:slug/access"
                      render={() => (
                        <AppIdentityServiceSettings
                          app={app}
                          refetch={this.getApp}
                        />
                      )}
                    />
                  )}
                  {/* snapshots redirects */}
                  <Redirect
                    exact
                    from="/app/:slug/snapshots"
                    to="/snapshots/partial/:slug"
                  />
                  <Redirect
                    exact
                    from="/app/:slug/snapshots/schedule"
                    to="/snapshots/settings?:slug"
                  />
                  <Redirect
                    exact
                    from="/app/:slug/snapshots/:id"
                    to="/snapshots/partial/:slug/:id"
                  />
                  <Redirect
                    exact
                    from="/app/:slug/snapshots/:id/restore"
                    to="/snapshots/partial/:slug/:id/restore"
                  />

                  <Route component={NotFound} />
                </Switch>
              </Fragment>
            )}
          </div>
        </SidebarLayout>
        {displayRequiredKotsUpdateModal && (
          <Modal
            isOpen={displayRequiredKotsUpdateModal}
            onRequestClose={() => this.toggleDisplayRequiredKotsUpdateModal("")}
            shouldReturnFocusAfterClose={false}
            contentLabel="Required KOTS Update modal"
            ariaHideApp={false}
            className="DisplayRequiredKotsUpdateModal--wrapper Modal"
          >
            <div className="Modal-body">
              <h2 className="u-fontSize--largest u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
                You must update KOTS to deploy this version
              </h2>
              <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">
                This version of {app?.name} requires a version of KOTS that is
                different from what you currently have installed.
              </p>
              <p className="u-fontSize--normal u-textColor--error u-fontWeight--medium u-lineHeight--normal u-marginBottom--20">
                {requiredKotsUpdateMessage}
              </p>
              <div className="u-marginTop--10 flex">
                <button
                  onClick={() => this.toggleDisplayRequiredKotsUpdateModal("")}
                  className="btn blue primary"
                >
                  Ok, got it!
                </button>
              </div>
            </div>
          </Modal>
        )}
        {gettingAppErrMsg && (
          <ErrorModal
            errorModal={this.state.displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            errMsg={gettingAppErrMsg}
            tryAgain={() => this.getApp(this.props.match.params.slug)}
            err="Failed to get application"
            loading={this.state.loadingApp}
          />
        )}
      </div>
    );
  }
}

export { AppDetailPage };
export default withTheme(withRouter(AppDetailPage));
>>>>>>> c78ee9a2d0570aea46b7b4c5645388316ca8ce72
