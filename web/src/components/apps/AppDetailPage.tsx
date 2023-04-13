import React, { Fragment, useReducer, useEffect, useState } from "react";
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
import { KotsSidebarItem } from "@src/components/watches/WatchSidebarItem";
import { HelmChartSidebarItem } from "@src/components/watches/WatchSidebarItem";
import NotFound from "../static/NotFound";
import { Dashboard } from "@features/Dashboard";
import DownstreamTree from "../../components/tree/KotsApplicationTree";
import AppVersionHistory from "./AppVersionHistory";
import { isAwaitingResults, Utilities } from "../../utilities/utilities";
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

import "../../scss/components/watches/WatchDetailPage.scss";
import { useApps, useSelectedApp } from "@features/App";

// Types
import { App, Metadata, KotsParams, Version } from "@types";

type Props = {
  adminConsoleMetadata?: Metadata;
  appNameSpace: string | null;
  appName: string | null;
  isHelmManaged: boolean;
  onActiveInitSession: (session: string) => void;
  ping: () => void;
  // TODO: remove this after adding app hook to Root-
  // right now the footer needs to update when the app list is updated by the AppDetailPage
  refetchAppsList: () => void;
  refetchAppMetadata: () => void;
  snapshotInProgressApps: string[];
};

type State = {
  clusterParentSlug: string;
  displayErrorModal: boolean;
  displayRequiredKotsUpdateModal: boolean;
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

function AppDetailPage(props: Props) {
  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      clusterParentSlug: "",
      displayErrorModal: false,
      displayRequiredKotsUpdateModal: false,
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
    }
  );

  const history = useHistory();
  const params = useParams<KotsParams>();
  const selectedApp = useSelectedApp();
  const [appsRefetchInterval, setAppsRefetchInterval] = useState<
    number | false
  >(false);
  const {
    data: appsData,
    error: appsError,
    isError: appsIsError,
    isFetching: appIsFetching,
    refetch: refetchApps,
  } = useApps({ refetchInterval: appsRefetchInterval });

  const { apps: appsList } = appsData || {};

  /**
   *  Runs on mount and on update. Also handles redirect logic
   *  if no apps are found, or the first app is found.
   */
  const redirectToFirstAppOrInstall = () => {
    // navigate to first app if available
    if (appsList && appsList?.length > 0) {
      history.replace(`/app/${appsList[0].slug}`);
    } else if (props.isHelmManaged) {
      history.replace("/install-with-helm");
    } else {
      history.replace("/upload-license");
    }
  };

  // loading state stuff that was in the old getApp() implementation
  useEffect(() => {
    if (appIsFetching && !selectedApp) {
      setState({
        loadingApp: true,
      });
    } else {
      if (!appsIsError) {
        if (appsList?.length === 0 || !params.slug) {
          redirectToFirstAppOrInstall();
          return;
        }
        if (
          state.loadingApp ||
          state.gettingAppErrMsg ||
          state.displayErrorModal
        ) {
          setState({
            loadingApp: false,
            gettingAppErrMsg: "",
            displayErrorModal: false,
          });
        }
      } else {
        setState({
          loadingApp: false,
          gettingAppErrMsg:
            appsError instanceof Error
              ? appsError.message
              : "Unexpected error when fetching apps",
          displayErrorModal: true,
        });
      }
    }
  }, [appsList, appIsFetching, appsIsError]);

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

  const checkIsVeleroInstalled = async () => {
    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/velero`, {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
        credentials: "include",
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
    refetchApps();
    props.refetchAppsList();
    props.refetchAppMetadata();
    checkIsVeleroInstalled();
  };

  const makeCurrentRelease = async (
    upstreamSlug: string,
    version: Version,
    isSkipPreflights: boolean,
    continueWithFailedPreflights = false
  ) => {
    try {
      setState({ makingCurrentReleaseErrMsg: "" });

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
        setState({ makingCurrentReleaseErrMsg: "" });
        refetchData();
      } else {
        const response = await res.json();
        setState({
          makingCurrentReleaseErrMsg: `Unable to deploy release ${version.versionLabel}, sequence ${version.sequence}: ${response.error}`,
        });
      }
    } catch (err) {
      console.log(err);
      if (err instanceof Error) {
        setState({
          makingCurrentReleaseErrMsg: `Unable to deploy release ${version.versionLabel}, sequence ${version.sequence}: ${err.message}`,
        });
      } else {
        setState({
          makingCurrentReleaseErrMsg: "Something went wrong, please try again.",
        });
      }
    }
  };

  const redeployVersion = async (
    upstreamSlug: string,
    version: Version | null
  ) => {
    try {
      setState({ redeployVersionErrMsg: "" });

      const res = await fetch(
        `${process.env.API_ENDPOINT}/app/${upstreamSlug}/sequence/${version?.sequence}/redeploy`,
        {
          headers: {
            Authorization: Utilities.getToken(),
            "Content-Type": "application/json",
          },
          method: "POST",
        }
      );
      if (res.ok && res.status === 204) {
        setState({ redeployVersionErrMsg: "" });
        refetchData();
      } else {
        setState({
          redeployVersionErrMsg: `Unable to redeploy release ${version?.versionLabel}, sequence ${version?.sequence}: Unexpected status code: ${res.status}`,
        });
      }
    } catch (err) {
      console.log(err);
      if (err instanceof Error) {
        setState({
          redeployVersionErrMsg: `Unable to deploy release ${version?.versionLabel}, sequence ${version?.sequence}: ${err.message}`,
        });
      } else {
        setState({
          redeployVersionErrMsg: "Something went wrong, please try again.",
        });
      }
    }
  };

  // Enforce initial app configuration (if exists)
  useEffect(() => {
    // Handle updating the theme state when switching apps.
    if (selectedApp?.iconUri) {
      const { navbarLogo, ...rest } = theme.getThemeState();
      if (navbarLogo === null || navbarLogo !== selectedApp.iconUri) {
        theme.setThemeState({
          ...rest,
          navbarLogo: selectedApp.iconUri,
        });
      }
    }
    // Refetch app info when switching between apps
    if (selectedApp && !appIsFetching && params.slug !== selectedApp.slug) {
      refetchApps();
      checkIsVeleroInstalled();
      return;
    }

    // Handle updating the theme state when switching apps.
    // Used for a fresh reload
    if (history.location.pathname === "/apps") {
      // updates state but does not cause infinite loop because app navigates away from /apps
      return;
    }

    // find if any app needs configuration and redirect to its configuration flow
    const appNeedsConfiguration = appsList?.find((app) => {
      return app?.downstream?.pendingVersions?.length > 0;
    });
    if (appNeedsConfiguration) {
      const downstream = appNeedsConfiguration.downstream;
      const firstVersion = downstream.pendingVersions.find(
        (version: Version) => version?.sequence === 0
      );
      if (firstVersion?.status === "pending_config") {
        history.push(`/${appNeedsConfiguration.slug}/config`);
        return;
      }
    }
  }, [selectedApp]);

  useEffect(() => {
    refetchApps();
    if (history.location.pathname === "/apps") {
      return;
    }
    // getApp();
    checkIsVeleroInstalled();
    return () => {
      theme.clearThemeState();
      setAppsRefetchInterval(false);
    };
  }, [history.location.pathname]);

  const { appName } = props;

  const {
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

  if (appIsFetching && !selectedApp) {
    return centeredLoader;
  }

  // poll version status if it's awaiting results
  const downstream = selectedApp?.downstream;
  if (
    downstream?.currentVersion &&
    isAwaitingResults([downstream.currentVersion])
  ) {
    if (appsRefetchInterval === false) {
      setAppsRefetchInterval(2000);
    }
  } else {
    if (appsRefetchInterval) {
      setAppsRefetchInterval(false);
    }
  }

  const resetRedeployErrorMessage = () => {
    setState({
      redeployVersionErrMsg: "",
    });
  };

  const resetMakingCurrentReleaseErrorMessage = () => {
    setState({
      makingCurrentReleaseErrMsg: "",
    });
  };

  return (
    <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
      <SidebarLayout
        className="flex flex1 u-minHeight--full u-overflow--hidden"
        condition={appsList && appsList?.length > 1}
        sidebar={
          <SideBar
            items={appsList
              ?.sort(
                (a: App, b: App) =>
                  new Date(b.createdAt).getTime() -
                  new Date(a.createdAt).getTime()
              )
              .map((item: App, idx: number) => {
                let sidebarItemNode;
                if (item.name) {
                  const slugFromRoute = params.slug;
                  sidebarItemNode = (
                    <KotsSidebarItem
                      key={idx}
                      className={classNames({
                        selected:
                          item.slug === slugFromRoute &&
                          params.owner !== "helm",
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
          {!selectedApp ? (
            centeredLoader
          ) : (
            <Fragment>
              <SubNavBar
                className="flex"
                activeTab={params.tab || "app"}
                app={selectedApp}
                isVeleroInstalled={isVeleroInstalled}
                isHelmManaged={props.isHelmManaged}
              />
              <Switch>
                <Route
                  exact
                  path="/app/:slug"
                  render={() => (
                    <Dashboard
                      app={selectedApp}
                      cluster={selectedApp.downstream?.cluster}
                      updateCallback={refetchData}
                      toggleIsBundleUploading={toggleIsBundleUploading}
                      makeCurrentVersion={makeCurrentRelease}
                      redeployVersion={redeployVersion}
                      isBundleUploading={isBundleUploading}
                      isVeleroInstalled={isVeleroInstalled}
                      refreshAppData={refetchApps}
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
                      app={selectedApp}
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
                      app={selectedApp}
                      match={{ match: { params: params } }}
                      makeCurrentVersion={makeCurrentRelease}
                      makingCurrentVersionErrMsg={
                        state.makingCurrentReleaseErrMsg
                      }
                      updateCallback={refetchData}
                      toggleIsBundleUploading={toggleIsBundleUploading}
                      isBundleUploading={isBundleUploading}
                      isHelmManaged={props.isHelmManaged}
                      refreshAppData={refetchApps}
                      displayErrorModal={state.displayErrorModal}
                      toggleErrorModal={toggleErrorModal}
                      makingCurrentRelease={state.makingCurrentRelease}
                      redeployVersion={redeployVersion}
                      redeployVersionErrMsg={state.redeployVersionErrMsg}
                      resetRedeployErrorMessage={resetRedeployErrorMessage}
                      resetMakingCurrentReleaseErrorMessage={
                        resetMakingCurrentReleaseErrorMessage
                      }
                      adminConsoleMetadata={props.adminConsoleMetadata}
                    />
                  )}
                />
                <Route
                  exact
                  path="/app/:slug/downstreams/:downstreamSlug/version-history/preflight/:sequence"
                  render={(renderProps) => (
                    <PreflightResultPage
                      logo={selectedApp.iconUri}
                      {...renderProps}
                    />
                  )}
                />
                <Route
                  exact
                  path="/app/:slug/config/:sequence?"
                  render={() => (
                    <AppConfig
                      app={selectedApp}
                      refreshAppData={refetchApps}
                      fromLicenseFlow={false}
                      isHelmManaged={props.isHelmManaged}
                    />
                  )}
                />
                <Route
                  path="/app/:slug/troubleshoot"
                  render={() => (
                    <TroubleshootContainer
                      app={selectedApp}
                      appName={appName || ""}
                    />
                  )}
                />
                <Route
                  exact
                  path="/app/:slug/license"
                  render={() => (
                    <AppLicense
                      app={selectedApp}
                      syncCallback={refetchData}
                      changeCallback={refetchData}
                      isHelmManaged={props.isHelmManaged}
                    />
                  )}
                />
                <Route
                  exact
                  path="/app/:slug/registry-settings"
                  render={() => (
                    <AppRegistrySettings
                      app={selectedApp}
                      updateCallback={refetchData}
                    />
                  )}
                />
                {selectedApp.isAppIdentityServiceSupported && (
                  <Route
                    exact
                    path="/app/:slug/access"
                    render={() => (
                      <AppIdentityServiceSettings
                        app={selectedApp}
                        refetch={refetchApps}
                      />
                    )}
                  />
                )}
                {/* snapshots redirects */}
                <Route
                  path="/app/:slug/snapshots"
                  render={() => <Redirect to="/snapshots/partial/:slug" />}
                />
                <Route
                  path="/app/:slug/snapshots/schedule"
                  render={() => <Redirect to="/snapshots/settings?:slug" />}
                />
                <Route
                  path="/app/:slug/snapshots/:id"
                  render={() => <Redirect to="/snapshots/partial/:slug/:id" />}
                />
                <Route
                  path="/app/:slug/snapshots/:id/restore"
                  render={() => (
                    <Redirect to="/snapshots/partial/:slug/:id/restore" />
                  )}
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
              This version of {selectedApp?.name} requires a version of KOTS
              that is different from what you currently have installed.
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
          tryAgain={() => refetchApps()}
          err="Failed to get application"
          loading={state.loadingApp}
        />
      )}
    </div>
  );
}

export { AppDetailPage };
