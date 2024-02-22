import { Fragment, useReducer, useEffect, useState } from "react";
import classNames from "classnames";
import { Outlet, useNavigate, useParams } from "react-router-dom";
import Modal from "react-modal";
import { useTheme } from "@src/components/context/withTheme";
import {
  HelmChartSidebarItem,
  KotsSidebarItem,
} from "@src/components/watches/WatchSidebarItem";
import { Utilities, isAwaitingResults } from "../../utilities/utilities";

import SubNavBar from "@src/components/shared/SubNavBar";
import SidebarLayout from "../layout/SidebarLayout/SidebarLayout";
import SideBar from "../shared/SideBar";
import Loader from "../shared/Loader";

import ErrorModal from "../modals/ErrorModal";

// Types
import { App, KotsParams, Metadata, Version } from "@types";
import { useApps, useSelectedApp } from "@features/App";

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
  isEmbeddedCluster: boolean;
  setShouldShowClusterUpgradeModal: (
    shouldShowClusterUpgradeModal: boolean
  ) => void;
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

  const navigate = useNavigate();
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
      navigate(`/app/${appsList[0].slug}`, { replace: true });
    } else if (props.isHelmManaged) {
      navigate("/install-with-helm", { replace: true });
    } else {
      navigate("/upload-license", { replace: true });
    }
  };

  // loading state stuff that was in the old getApp() implementation
  useEffect(() => {
    if (appIsFetching && !selectedApp) {
      setState({
        loadingApp: true,
      });
    } else {
      const shouldShowUpgradeModal =
        Utilities.shouldShowClusterUpgradeModal(appsList);
      props.setShouldShowClusterUpgradeModal(shouldShowUpgradeModal);
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
        let gettingAppErrMsg =
          appsError instanceof Error
            ? appsError.message
            : "Unexpected error when fetching apps";
        let displayErrorModal = true;
        if (shouldShowUpgradeModal) {
          // don't show apps error modal if cluster is upgrading
          gettingAppErrMsg = "";
          displayErrorModal = false;
        }
        setState({
          loadingApp: false,
          gettingAppErrMsg: gettingAppErrMsg,
          displayErrorModal: displayErrorModal,
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
            "Content-Type": "application/json",
          },
          credentials: "include",
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
            "Content-Type": "application/json",
          },
          credentials: "include",
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

  useEffect(() => {
    refetchApps();
    if (location.pathname === "/apps") {
      return;
    }
    checkIsVeleroInstalled();
    return () => {
      theme.clearThemeState();
      setAppsRefetchInterval(false);
    };
  }, [location.pathname]);

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
    if (location.pathname === "/apps") {
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
      if (
        firstVersion?.status === "pending_cluster_management" &&
        props.isEmbeddedCluster
      ) {
        navigate(`/${appNeedsConfiguration.slug}/cluster/manage`);
        return;
      }
      if (firstVersion?.status === "pending_config") {
        navigate(`/${appNeedsConfiguration.slug}/config`);
        return;
      }
    }
  }, [selectedApp]);

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

  if (
    appIsFetching &&
    !selectedApp &&
    !Utilities.shouldShowClusterUpgradeModal(appsList)
  ) {
    return centeredLoader;
  }

  // poll version status if it's awaiting results or if the cluster is upgrading
  const downstream = selectedApp?.downstream;
  if (
    (downstream?.currentVersion &&
      isAwaitingResults([downstream.currentVersion])) ||
    Utilities.shouldShowClusterUpgradeModal(appsList)
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

  const context = {
    adminConsoleMetadata: props.adminConsoleMetadata,
    app: selectedApp,
    appName: props.appName,
    appNameSpace: props.appNameSpace,
    cluster: selectedApp?.downstream?.cluster,
    displayErrorModal: state.displayErrorModal,
    isBundleUploading: isBundleUploading,
    isHelmManaged: props.isHelmManaged,
    isEmbeddedCluster: props.isEmbeddedCluster,
    isVeleroInstalled: isVeleroInstalled,
    logo: selectedApp?.iconUri,
    makeCurrentVersion: makeCurrentRelease,
    makingCurrentRelease: state.makingCurrentRelease,
    makingCurrentVersionErrMsg: state.makingCurrentReleaseErrMsg,
    ping: props.ping,
    redeployVersion: redeployVersion,
    redeployVersionErrMsg: state.redeployVersionErrMsg,
    refreshAppData: refetchApps,
    resetMakingCurrentReleaseErrorMessage,
    resetRedeployErrorMessage,
    toggleErrorModal: toggleErrorModal,
    toggleIsBundleUploading: toggleIsBundleUploading,
    updateCallback: refetchData,
  };

  const lastItem = location.pathname.substring(
    location.pathname.lastIndexOf("/") + 1
  );

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
                activeTab={lastItem === params.slug ? "app" : lastItem}
                app={selectedApp}
                isVeleroInstalled={isVeleroInstalled}
                isHelmManaged={props.isHelmManaged}
              />
              <Outlet context={context} />
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
