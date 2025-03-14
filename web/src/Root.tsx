import React, { createContext, useEffect, useReducer, useState } from "react";
import { createBrowserHistory } from "history";
import { Navigate, Route, Routes, useNavigate } from "react-router-dom";
import { Helmet } from "react-helmet";
import Modal from "react-modal";
import find from "lodash/find";
import ConnectionTerminated from "./ConnectionTerminated";
import GitOps from "./components/clusters/GitOps";
import PreflightResultPage from "./components/PreflightResultPage";
import AppConfig from "./features/AppConfig/components/AppConfig";
import { AppDetailPage } from "./components/apps/AppDetailPage";
import KurlClusterManagement from "./components/apps/KurlClusterManagement";
import EmbeddedClusterManagement from "@components/apps/EmbeddedClusterManagement";
import UnsupportedBrowser from "./components/static/UnsupportedBrowser";
import NotFound from "./components/static/NotFound";
import { parseUpstreamUri, Utilities } from "./utilities/utilities";
import fetch from "./utilities/fetchWithTimeout";
import { SecureAdminConsole } from "@features/Auth";
import UploadLicenseFile from "./components/UploadLicenseFile";
import BackupRestore from "./components/BackupRestore";
import UploadAirgapBundle from "./components/UploadAirgapBundle";
import RestoreCompleted from "./components/RestoreCompleted";
import Access from "./components/identity/Access";
import SnapshotsWrapper from "./components/snapshots/SnapshotsWrapper";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { InstallWithHelm } from "@features/AddNewApp";
import DownstreamTree from "./components/tree/KotsApplicationTree";
import { Dashboard } from "@features/Dashboard/components/Dashboard";
import AppVersionHistory from "@components/apps/AppVersionHistory";
import AppLicense from "@components/apps/AppLicense";
import AppRegistrySettings from "@components/apps/AppRegistrySettings";
import AppIdentityServiceSettings from "@components/apps/AppIdentityServiceSettings";
import TroubleshootContainer from "@components/troubleshoot/TroubleshootContainer";

import Footer from "./components/shared/Footer";
import NavBar from "./components/shared/NavBar";
import connectHistory from "./services/matomo";

// types
import { App, Metadata, ThemeState } from "@types";
import { ToastProvider } from "./context/ToastContext";
import Redactors from "@components/redactors/Redactors";
import EditRedactor from "@components/redactors/EditRedactor";
import SupportBundleAnalysis from "@components/troubleshoot/SupportBundleAnalysis";
import GenerateSupportBundle from "@components/troubleshoot/GenerateSupportBundle";
import SupportBundleList from "@components/troubleshoot/SupportBundleList";
import AnalyzerInsights from "@components/troubleshoot/AnalyzerInsights";
import AnalyzerFileTree from "@components/troubleshoot/AnalyzerFileTree";
import AnalyzerRedactorReport from "@components/troubleshoot/AnalyzerRedactorReport";
import Snapshots from "@components/snapshots/Snapshots";
import SnapshotSettings from "@components/snapshots/SnapshotSettings";
import SnapshotDetails from "@components/snapshots/SnapshotDetails";
import SnapshotRestore from "@components/snapshots/SnapshotRestore";
import AppSnapshots from "@components/snapshots/AppSnapshots";
import AppSnapshotRestore from "@components/snapshots/AppSnapshotRestore";
import EmbeddedClusterViewNode from "@components/apps/EmbeddedClusterViewNode";
import UpgradeStatusModal from "@components/modals/UpgradeStatusModal";
import AppLoading from "@components/apps/AppLoading";
import Icon from "@components/Icon";

import "./scss/components/watches/WatchConfig.scss";

// react-query client
const queryClient = new QueryClient();

const INIT_SESSION_ID_STORAGE_KEY = "initSessionId";

let browserHistory = createBrowserHistory();
let history = connectHistory(browserHistory);

// TODO:  pull in the react router hook

const ThemeContext = createContext({
  setThemeState: (themeState?: ThemeState) => {
    console.log("setThemeState used before being set", themeState);
  },
  getThemeState: (): ThemeState => ({ navbarLogo: null }),
  clearThemeState: () => {},
});

type ConfigGroupItem = {
  name: string;
  title: string;
  type: string;
  hidden: boolean;
  validationError: boolean;
  error: boolean;
  when: string;
};

type NavbarConfigGroup = {
  name: string;
  title: string;
  items: ConfigGroupItem[];
  hidden: boolean;
  hasError: boolean;
  when: string;
};

type AppBranding = {
  css?: string[];
  fontFaces?: string[];
  logo: string;
};

type State = {
  app: App | null;
  appBranding: AppBranding | null;
  appLogo: string | null;
  appNameSpace: string | null;
  appsList: App[];
  appSlugFromMetadata: string | null;
  adminConsoleMetadata: Metadata | null;
  connectionTerminated: boolean;
  showUpgradeStatusModal: boolean;
  upgradeStatus?: string;
  upgradeMessage?: string;
  upgradeAppSlug?: string;
  clusterState: string;
  errLoggingOut: string;
  featureFlags: object;
  fetchingMetadata: boolean;
  initSessionId: string | null;
  selectedAppName: string | null;
  snapshotInProgressApps: string[];
  isEmbeddedClusterWaitingForNodes: boolean;
  themeState: ThemeState;
  activeGroups: string[];
};

let interval: ReturnType<typeof setInterval> | undefined;

const Root = () => {
  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      appBranding: null,
      appLogo: null,
      appsList: [],
      appSlugFromMetadata: null,
      appNameSpace: null,
      adminConsoleMetadata: null,
      activeGroups: null,
      connectionTerminated: false,
      showUpgradeStatusModal: false,
      upgradeStatus: "",
      upgradeMessage: "",
      upgradeAppSlug: "",
      clusterState: "",
      errLoggingOut: "",
      featureFlags: {},
      fetchingMetadata: false,
      initSessionId: Utilities.localStorageEnabled()
        ? localStorage.getItem(INIT_SESSION_ID_STORAGE_KEY)
        : "",
      selectedAppName: null,
      snapshotInProgressApps: [],
      isEmbeddedClusterWaitingForNodes: false,
      themeState: {
        navbarLogo: null,
      },
      app: null,
    }
  );

  /**
   * Sets the Theme State for the whole application
   * @param {Object} newThemeState - Object to set for new theme state
   * @param {Function} callback - callback to run like in setState()'s callback
   */
  const setThemeState = (newThemeState?: ThemeState) => {
    if (newThemeState) {
      setState({
        themeState: { ...newThemeState },
      });
    }
  };

  /**
   * Gets the current theme state of the app
   * @return {Object}
   */
  const getThemeState = () => {
    return state.themeState;
  };

  /**
   * Clears the current theme state to nothing
   */
  const clearThemeState = () => {
    /**
     * Reference object to a blank theme state
     */
    const EMPTY_THEME_STATE = {
      navbarLogo: null,
    };

    setState({
      themeState: { ...EMPTY_THEME_STATE },
    });
  };

  const handleActiveInitSession = (initSessionId: string) => {
    if (Utilities.localStorageEnabled()) {
      localStorage.setItem(INIT_SESSION_ID_STORAGE_KEY, initSessionId);
    }
    setState({ initSessionId });
  };

  const getPendingApp = async () => {
    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/pendingapp`, {
        headers: {
          "Content-Type": "application/json",
        },
        method: "GET",
        credentials: "include",
      });
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }
        if (res.status === 404) {
          return;
        }

        console.log(
          "failed to get pending apps, unexpected status code",
          res.status
        );
        return;
      }
      const response = await res.json();
      const app = response.app;
      // TODO: delete if not used
      // setState({
      //   pendingApp: app,
      // });
      return app;
    } catch (err) {
      throw err;
    }
  };

  const getAppsList = async () => {
    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/apps`, {
        headers: {
          "Content-Type": "application/json",
        },
        method: "GET",
        credentials: "include",
      });
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }
        console.log("failed to list apps, unexpected status code", res.status);
        return;
      }
      const response = await res.json();

      const apps = response.apps;
      setState({
        appsList: apps,
      });
      return apps;
    } catch (err) {
      throw err;
    }
  };

  const fetchUpgradeStatus = async (appSlug) => {
    try {
      const url = `${process.env.API_ENDPOINT}/app/${appSlug}/task/upgrade-service`;
      const res = await fetch(url, {
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        method: "GET",
      });
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }
        console.log(
          "failed to get upgrade service status, unexpected status code",
          res.status
        );
        return;
      }
      const response = await res.json();
      const status = response.status;
      if (
        status === "upgrading-cluster" ||
        status === "upgrading-app" ||
        status === "upgrade-failed"
      ) {
        setState({
          showUpgradeStatusModal: true,
          upgradeStatus: status,
          upgradeMessage: response.currentMessage,
          upgradeAppSlug: appSlug,
        });
        return;
      }
      if (state.showUpgradeStatusModal) {
        // upgrade finished, reload the page
        window.location.reload();
        return;
      }
    } catch (err) {
      throw err;
    }
  };

  const fetchKotsAppMetadata = async () => {
    setState({ fetchingMetadata: true });

    fetch(`${process.env.API_ENDPOINT}/metadata`, {
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      method: "GET",
    })
      .then(async (res) => {
        const data = await res.json();
        if (!data) {
          setState({ fetchingMetadata: false });
          return;
        }

        setState({
          appLogo: data.iconUri,
          appBranding: data.branding,
          selectedAppName: data.name,
          app: data,
          appSlugFromMetadata: parseUpstreamUri(data.upstreamUri),
          appNameSpace: data.namespace,
          adminConsoleMetadata: data.adminConsoleMetadata,
          featureFlags: data.consoleFeatureFlags,
          fetchingMetadata: false,
          isEmbeddedClusterWaitingForNodes:
            data.isEmbeddedClusterWaitingForNodes,
        });
      })
      .catch((err) => {
        setState({ fetchingMetadata: false });
        throw err;
      });
  };

  const ping = async (tries = 0) => {
    if (!Utilities.isLoggedIn()) {
      return;
    }
    let apps = state.appsList;
    const appSlugs = apps?.map((a) => a.slug);
    const url = `${process.env.API_ENDPOINT}/ping?slugs=${appSlugs}`;
    await fetch(
      url,
      {
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
      },
      10000
    )
      .then(async (res) => {
        if (!res.ok) {
          if (res.status === 401) {
            Utilities.logoutUser();
            return;
          }
          throw new Error(`Unexpected status code: ${res.status}`);
        }
        const body = await res.json();
        setState({
          connectionTerminated: false,
          snapshotInProgressApps: body.snapshotInProgressApps,
        });
      })
      .catch(() => {
        if (tries < 2) {
          setTimeout(() => {
            ping(tries + 1);
          }, 1000);
          return;
        }
        setState({
          connectionTerminated: true,
          snapshotInProgressApps: [],
        });
      });
  };

  const onRootMounted = async () => {
    fetchKotsAppMetadata();
    if (Utilities.isLoggedIn()) {
      ping();
      getAppsList().then((appsList) => {
        if (!appsList?.length) {
          return;
        }
        const { slug } = appsList[0];
        fetchUpgradeStatus(slug);
        if (window.location.pathname === "/apps") {
          history.replace(`/app/${slug}`);
        }
      });
    }
  };

  useEffect(() => {
    onRootMounted();
    interval = setInterval(ping, 10000);

    return () => {
      clearInterval(interval);
    };
  }, []);

  useEffect(() => {
    if (state.connectionTerminated) {
      clearInterval(interval);
    } else {
      interval = setInterval(ping, 10000);
    }

    return () => {
      clearInterval(interval);
    };
  }, [state.connectionTerminated]);

  const isGitOpsSupported = () => {
    const apps = state.appsList;
    return !!find(apps, (app) => app.isGitOpsSupported);
  };

  const isIdentityServiceSupported = () => {
    const apps = state.appsList;
    return !!find(apps, (app) => app.isIdentityServiceSupported);
  };

  const isGeoaxisSupported = () => {
    const apps = state.appsList;
    return !!find(apps, (app) => app.isGeoaxisSupported);
  };

  const isSnapshotsSupported = () => {
    const apps = state.appsList;
    return !!find(apps, (app) => app.allowSnapshots);
  };

  const onLogoutError = (message: string) => {
    setState({
      errLoggingOut: message,
    });
  };

  const Crashz = () => {
    throw new Error("Crashz!");
  };
  const navigate = useNavigate();

  const [currentStep, setCurrentStep] = useState(0);
  const [navbarConfigGroups, setNavbarConfigGroups] = useState<
    NavbarConfigGroup[]
  >([]);
  const [activeGroups, setActiveGroups] = useState<string[]>([]);

  const getStepProps = (step: number) => {
    if (step < currentStep) {
      return {
        icon: "check-circle-filled",
        textColor: "tw-text-gray-400",
        fontClass: "",
      };
    } else if (step === currentStep) {
      return {
        icon: "check-gray-filled",
        textColor: "tw-text-gray-800",
        fontClass: "tw-font-bold",
      };
    } else {
      return {
        icon: "check-gray-filled",
        textColor: "",
        fontClass: "",
      };
    }
  };

  const toggleActiveGroups = (name: string) => {
    let groupsArr = activeGroups;
    if (groupsArr.includes(name)) {
      let updatedGroupsArr = groupsArr.filter((n) => n !== name);
      setActiveGroups(updatedGroupsArr);
    } else {
      setActiveGroups([...groupsArr, name]);
    }
  };

  const NavGroup = React.memo(
    ({
      group,
      isActive,
      i,
    }: {
      group: NavbarConfigGroup;
      isActive: boolean;
      i: number;
    }) => {
      return (
        <div
          key={`${i}-${group.name}-${group.title}`}
          className={`side-nav-group ${isActive ? "group-open" : ""}`}
          id={`config-group-nav-${group.name}`}
        >
          <div
            className="flex alignItems--center"
            onClick={() => toggleActiveGroups(group.name)}
          >
            <div className="u-lineHeight--normal group-title u-fontSize--normal">
              {group.title}
            </div>
            {/* adding the arrow-down classes, will rotate the icon when clicked */}
            <Icon
              icon="down-arrow"
              className="darkGray-color clickable flex-auto u-marginLeft--5 arrow-down"
              size={12}
              style={{}}
              color={""}
              disableFill={false}
              removeInlineStyle={false}
            />
          </div>
          {group.items ? (
            <div className="side-nav-items">
              {group.items
                ?.filter((item) => item?.type !== "label")
                ?.map((item, j) => {
                  const hash = location.hash.slice(1);
                  if (item.hidden || item.when === "false") {
                    return;
                  }
                  return (
                    <a
                      className={`u-fontSize--normal u-lineHeight--normal
                                ${
                                  item.validationError || item.error
                                    ? "has-error"
                                    : ""
                                }
                                ${
                                  hash === `${item.name}-group`
                                    ? "active-item"
                                    : ""
                                }`}
                      href={`#${item.name}-group`}
                      key={`${j}-${item.name}-${item.title}`}
                    >
                      {item.title}
                    </a>
                  );
                })}
            </div>
          ) : null}
        </div>
      );
    }
  );

  return (
    <QueryClientProvider client={queryClient}>
      <Helmet>
        <meta
          httpEquiv="Cache-Control"
          content="no-cache, no-store, must-revalidate"
        />
        <meta httpEquiv="Pragma" content="no-cache" />
        <meta httpEquiv="Expires" content="0" />
        {state.appLogo && (
          <link rel="icon" type="image/png" href={state.appLogo} />
        )}
        {state.appBranding?.fontFaces &&
          state.appBranding.fontFaces.map((fontFace, index) => (
            <style
              type="text/css"
              id={`kots-branding-font-face-${index}`}
              key={`kots-branding-font-face-${index}`}
            >
              {fontFace}
            </style>
          ))}
        {state.appBranding?.css &&
          state.appBranding.css.map((style, index) => (
            <style
              type="text/css"
              id={`kots-branding-css-${index}`}
              key={`kots-branding-css-${index}`}
            >
              {style}
            </style>
          ))}
      </Helmet>
      <ThemeContext.Provider
        value={{
          setThemeState,
          getThemeState,
          clearThemeState,
        }}
      >
        <ToastProvider>
          {/* eslint-disable-next-line */}
          {/* @ts-ignore */}

          <NavBar
            logo={state.themeState.navbarLogo || state.appLogo}
            refetchAppsList={getAppsList}
            fetchingMetadata={state.fetchingMetadata}
            isKurlEnabled={Boolean(state.adminConsoleMetadata?.isKurl)}
            isEmbeddedClusterEnabled={Boolean(
              state.adminConsoleMetadata?.isEmbeddedCluster
            )}
            isGitOpsSupported={isGitOpsSupported()}
            isIdentityServiceSupported={isIdentityServiceSupported()}
            appsList={state.appsList}
            onLogoutError={onLogoutError}
            isSnapshotsSupported={isSnapshotsSupported()}
            errLoggingOut={state.errLoggingOut}
          />
          <div className="tw-flex flex1" data-testid="root-container">
            {(state.adminConsoleMetadata?.isKurl ||
              state.adminConsoleMetadata?.isEmbeddedCluster) &&
              Utilities.isInitialAppInstall(state.appsList[0]) &&
              Utilities.isLoggedIn() && (
                <div className="tw-w-[400px]  tw-bg-[#F9FBFC]" data-testid="get-started-sidebar">
                  <div className="tw-py-8 tw-pl-8 tw-shadow-[0_1px_0_#c4c8ca]">
                    <span className="tw-text-lg tw-font-semibold  tw-text-gray-800">
                      Let's get you started!
                    </span>
                  </div>
                  <div className="tw-p-8 tw-shadow-[0_1px_0_#c4c8ca] tw-font-medium tw-flex">
                    <Icon
                      icon={getStepProps(0).icon}
                      size={16}
                      className="tw-mr-2"
                    />
                    <span
                      className={`${getStepProps(0).fontClass} ${
                        getStepProps(0).textColor
                      }`}
                    >
                      Secure the Admin Console
                    </span>
                  </div>
                  <div className="tw-p-8 tw-shadow-[0_1px_0_#c4c8ca] tw-font-medium tw-flex">
                    <Icon
                      icon={getStepProps(1).icon}
                      size={16}
                      className="tw-mr-2"
                    />
                    <span
                      className={`${getStepProps(1).fontClass} ${
                        getStepProps(1).textColor
                      }`}
                    >
                      Configure the cluster (optional)
                    </span>
                  </div>
                  <div className="tw-p-8 tw-shadow-[0_1px_0_#c4c8ca] tw-font-medium">
                    <div className="tw-flex">
                      <Icon
                        icon={getStepProps(2).icon}
                        size={16}
                        className="tw-mr-2"
                      />
                      <span
                        className={`${getStepProps(2).fontClass} ${
                          getStepProps(2).textColor
                        }`}
                      >
                        Configure {state.selectedAppName || ""}
                      </span>
                    </div>
                    {navbarConfigGroups.length > 0 && (
                      <div
                        id="configSidebarWrapper"
                        className="config-sidebar-wrapper clickable !tw-bg-[#F9FBFC] tw-pt-4 tw-px-5"
                      >
                        {navbarConfigGroups?.map((group, i) => {
                          if (
                            group.title === "" ||
                            group.title.length === 0 ||
                            group.hidden ||
                            group.when === "false"
                          ) {
                            return;
                          }
                          const isActive =
                            activeGroups.includes(group.name) || group.hasError;

                          return (
                            <NavGroup group={group} isActive={isActive} i={i} />
                          );
                        })}
                      </div>
                    )}
                  </div>
                  <div className="tw-p-8 tw-shadow-[0_1px_0_#c4c8ca] tw-font-medium tw-leading-6 tw-flex">
                    <Icon
                      icon={getStepProps(3).icon}
                      size={16}
                      className="tw-mr-2"
                    />
                    <span
                      className={`${getStepProps(3).fontClass} ${
                        getStepProps(3).textColor
                      }`}
                    >
                      Validate the environment & deploy{" "}
                      {state.selectedAppName || ""}
                    </span>
                  </div>
                </div>
              )}

            <div className="flex1 flex-column u-overflow--auto tw-relative">
              <Routes>
                <Route
                  path="/"
                  element={
                    <Navigate
                      to={Utilities.isLoggedIn() ? "/apps" : "/secure-console"}
                    />
                  }
                />
                <Route path="/crashz" element={<Crashz />} />{" "}
                <Route path="*" element={<NotFound />} />
                <Route
                  path="/secure-console"
                  element={
                    <SecureAdminConsole
                      logo={state.appLogo}
                      appName={state.selectedAppName}
                      pendingApp={getPendingApp}
                      onLoginSuccess={getAppsList}
                      fetchingMetadata={state.fetchingMetadata}
                      navigate={navigate}
                      isEmbeddedClusterWaitingForNodes={
                        state.isEmbeddedClusterWaitingForNodes
                      }
                    />
                  }
                />
                <Route
                  path="/:slug/preflight"
                  element={
                    <PreflightResultPage
                      logo={state.appLogo || ""}
                      fromLicenseFlow={true}
                      refetchAppsList={getAppsList}
                      setCurrentStep={setCurrentStep}
                      isEmbeddedCluster={
                        state.adminConsoleMetadata?.isEmbeddedCluster
                      }
                    />
                  }
                />
                <Route
                  path="/:slug/config"
                  element={
                    <AppConfig
                      fromLicenseFlow={true}
                      refetchAppsList={getAppsList}
                      isEmbeddedCluster={
                        state.adminConsoleMetadata?.isEmbeddedCluster
                      }
                      setCurrentStep={setCurrentStep}
                      setNavbarConfigGroups={setNavbarConfigGroups}
                      setActiveGroups={setActiveGroups}
                    />
                  }
                />
                <Route
                  path="/upload-license"
                  element={
                    <UploadLicenseFile
                      logo={state.appLogo}
                      appsListLength={state.appsList?.length}
                      appName={state.selectedAppName || ""}
                      appSlugFromMetadata={state.appSlugFromMetadata || ""}
                      fetchingMetadata={state.fetchingMetadata}
                      onUploadSuccess={getAppsList}
                      isEmbeddedCluster={Boolean(
                        state.adminConsoleMetadata?.isEmbeddedCluster
                      )}
                    />
                  }
                />
                <Route path="/cluster/loading" element={<AppLoading />} />
                <Route
                  path="/install-with-helm"
                  element={<InstallWithHelm />}
                />
                <Route
                  path="/restore"
                  element={
                    <BackupRestore
                      logo={state.appLogo}
                      appName={state.selectedAppName}
                      appsListLength={state.appsList?.length}
                      fetchingMetadata={state.fetchingMetadata}
                    />
                  }
                />
                <Route
                  path="/:slug/airgap"
                  element={
                    <UploadAirgapBundle
                      showRegistry={true}
                      logo={state.appLogo}
                      appsListLength={state.appsList?.length}
                      appName={state.selectedAppName}
                      onUploadSuccess={getAppsList}
                      fetchingMetadata={state.fetchingMetadata}
                    />
                  }
                />
                <Route
                  path="/:slug/airgap-bundle"
                  element={
                    <UploadAirgapBundle
                      showRegistry={false}
                      logo={state.appLogo}
                      appsListLength={state.appsList?.length}
                      appName={state.selectedAppName}
                      onUploadSuccess={getAppsList}
                      fetchingMetadata={state.fetchingMetadata}
                    />
                  }
                />
                <Route path="/unsupported" element={<UnsupportedBrowser />} />
                {state.adminConsoleMetadata?.isEmbeddedCluster && (
                  <>
                    <Route
                      path="/:slug/cluster/manage"
                      element={
                        <EmbeddedClusterManagement
                          fromLicenseFlow={true}
                          setCurrentStep={setCurrentStep}
                        />
                      }
                    />
                    <Route
                      path="/:slug/cluster/:nodeName"
                      element={<EmbeddedClusterViewNode />}
                    />
                  </>
                )}
                {(state.adminConsoleMetadata?.isKurl ||
                  state.adminConsoleMetadata?.isEmbeddedCluster) && (
                  <Route
                    path="/cluster/manage"
                    element={
                      state.adminConsoleMetadata?.isKurl ? (
                        <KurlClusterManagement />
                      ) : (
                        <EmbeddedClusterManagement
                          setCurrentStep={setCurrentStep}
                        />
                      )
                    }
                  />
                )}
                {state.adminConsoleMetadata?.isEmbeddedCluster && (
                  <Route
                    path="/cluster/:nodeName"
                    element={<EmbeddedClusterViewNode />}
                  />
                )}
                <Route
                  path="/gitops"
                  element={<GitOps appName={state.selectedAppName || ""} />}
                />
                <Route
                  path="/access/:tab?"
                  element={
                    <Access
                      isKurlEnabled={
                        state.adminConsoleMetadata?.isKurl || false
                      }
                      isGeoaxisSupported={isGeoaxisSupported()}
                    />
                  }
                />
                {/* :tab?  */}
                <Route
                  path="/snapshots/*"
                  element={
                    <SnapshotsWrapper
                      appName={state.selectedAppName}
                      isKurlEnabled={state.adminConsoleMetadata?.isKurl}
                      isEmbeddedCluster={
                        state.adminConsoleMetadata?.isEmbeddedCluster
                      }
                      appsList={state.appsList}
                    />
                  }
                >
                  <Route
                    index
                    element={
                      <Snapshots
                        isKurlEnabled={state.adminConsoleMetadata?.isKurl}
                        isEmbeddedCluster={
                          state.adminConsoleMetadata?.isEmbeddedCluster
                        }
                        appsList={state.appsList}
                      />
                    }
                  />
                  <Route
                    path="settings"
                    element={
                      // eslint-disable-next-line
                      // @ts-ignore
                      <SnapshotSettings
                        isKurlEnabled={state.adminConsoleMetadata?.isKurl}
                        isEmbeddedCluster={
                          state.adminConsoleMetadata?.isEmbeddedCluster
                        }
                        appsList={state.appsList}
                      />
                    }
                  />
                  <Route
                    path="details/:id"
                    element={
                      <SnapshotDetails
                        isKurlEnabled={state.adminConsoleMetadata?.isKurl}
                        isEmbeddedCluster={
                          state.adminConsoleMetadata?.isEmbeddedCluster
                        }
                        appsList={state.appsList}
                      />
                    }
                  />
                  <Route
                    path=":slug/:id/restore"
                    element={<SnapshotRestore />}
                  />
                  <Route
                    path="partial/:slug"
                    element={
                      <AppSnapshots
                        appsList={state.appsList}
                        appName={state.selectedAppName}
                      />
                    }
                  />
                  <Route
                    path="partial/:slug/:id"
                    element={
                      <SnapshotDetails
                        appsList={state.appsList}
                        appName={state.selectedAppName}
                      />
                    }
                  />
                  <Route
                    path="partial/:slug/:id/restore"
                    element={<AppSnapshotRestore appsList={state.appsList} />}
                  />
                </Route>
                <Route
                  path="/apps"
                  element={
                    <AppDetailPage
                      refetchAppMetadata={fetchKotsAppMetadata}
                      onActiveInitSession={handleActiveInitSession}
                      appNameSpace={state.appNameSpace}
                      appName={state.selectedAppName}
                      refetchAppsList={getAppsList}
                      refetchUpgradeStatus={fetchUpgradeStatus}
                      snapshotInProgressApps={state.snapshotInProgressApps}
                      ping={ping}
                      isEmbeddedCluster={Boolean(
                        state.adminConsoleMetadata?.isEmbeddedCluster
                      )}
                      showUpgradeStatusModal={state.showUpgradeStatusModal}
                    />
                  }
                />
                <Route
                  path="/app/*"
                  element={
                    <AppDetailPage
                      refetchAppMetadata={fetchKotsAppMetadata}
                      onActiveInitSession={handleActiveInitSession}
                      appNameSpace={state.appNameSpace}
                      appName={state.selectedAppName}
                      refetchAppsList={getAppsList}
                      refetchUpgradeStatus={fetchUpgradeStatus}
                      snapshotInProgressApps={state.snapshotInProgressApps}
                      ping={ping}
                      adminConsoleMetadata={state.adminConsoleMetadata}
                      isEmbeddedCluster={Boolean(
                        state.adminConsoleMetadata?.isEmbeddedCluster
                      )}
                      showUpgradeStatusModal={state.showUpgradeStatusModal}
                    />
                  }
                >
                  <Route
                    path=":slug"
                    element={
                      <Dashboard
                        adminConsoleMetadata={state.adminConsoleMetadata}
                      />
                    }
                  />
                  <Route
                    path=":slug/tree/:sequence?"
                    element={
                      <DownstreamTree
                        isEmbeddedCluster={Boolean(
                          state.adminConsoleMetadata?.isEmbeddedCluster
                        )}
                      />
                    }
                  />

                  <Route
                    path={":slug/version-history"}
                    // eslint-disable-next-line
                    // @ts-ignore
                    element={<AppVersionHistory />}
                  />
                  <Route
                    path={
                      ":slug/version-history/diff/:firstSequence/:secondSequence"
                    }
                    // eslint-disable-next-line
                    // @ts-ignore
                    element={<AppVersionHistory />}
                  />
                  <Route
                    path=":slug/downstreams/:downstreamSlug/version-history/preflight/:sequence"
                    element={
                      <PreflightResultPage
                        logo={state.appLogo || ""}
                        fromLicenseFlow={true}
                        refetchAppsList={getAppsList}
                        setCurrentStep={setCurrentStep}
                        isEmbeddedCluster={
                          state.adminConsoleMetadata?.isEmbeddedCluster
                        }
                      />
                    }
                  />
                  <Route
                    path=":slug/config/:sequence"
                    element={
                      <AppConfig
                        fromLicenseFlow={false}
                        refetchAppsList={getAppsList}
                        isEmbeddedCluster={
                          state.adminConsoleMetadata?.isEmbeddedCluster
                        }
                        setCurrentStep={setCurrentStep}
                        setNavbarConfigGroups={setNavbarConfigGroups}
                        setActiveGroups={setActiveGroups}
                      />
                    }
                  />
                  <Route
                    path=":slug/troubleshoot"
                    element={
                      //@ts-ignore
                      <TroubleshootContainer />
                    }
                  >
                    <Route
                      index
                      element={
                        <SupportBundleList
                          isEmbeddedClusterEnabled={Boolean(
                            state.adminConsoleMetadata?.isEmbeddedCluster
                          )}
                        />
                      }
                    />
                    <Route
                      path="generate"
                      element={
                        <GenerateSupportBundle
                          isEmbeddedClusterEnabled={Boolean(
                            state.adminConsoleMetadata?.isEmbeddedCluster
                          )}
                        />
                      }
                    />
                    <Route
                      path="analyze/:bundleSlug"
                      element={<SupportBundleAnalysis />}
                    >
                      <Route index element={<AnalyzerInsights />} />
                      <Route
                        path={"contents/*"}
                        element={<AnalyzerFileTree />}
                      />
                      <Route
                        path={"redactor/report"}
                        element={<AnalyzerRedactorReport />}
                      />
                    </Route>
                    <Route path="redactors" element={<Redactors />} />
                    <Route path="redactors/new" element={<EditRedactor />} />
                    <Route
                      path="redactors/:redactorSlug"
                      element={<EditRedactor />}
                    />
                    <Route element={<NotFound />} />
                  </Route>
                  <Route
                    path=":slug/license"
                    element={
                      <AppLicense
                        //@ts-ignore
                        isEmbeddedCluster={Boolean(
                          state.adminConsoleMetadata?.isEmbeddedCluster
                        )}
                      />
                    }
                  />
                  <Route
                    path=":slug/registry-settings"
                    element={<AppRegistrySettings />}
                  />
                  {/* WHERE IS SELECTEDAPP */}
                  {state.app?.isAppIdentityServiceSupported && (
                    <Route
                      path=":slug/access"
                      element={<AppIdentityServiceSettings />}
                    />
                  )}
                  {/* snapshots redirects */}
                  <Route
                    path=":slug/snapshots"
                    element={<Navigate to="/snapshots/partial/:slug" />}
                  />
                  <Route
                    path=":slug/snapshots/schedule"
                    element={<Navigate to="/snapshots/settings?:slug" />}
                  />
                  <Route
                    path=":slug/snapshots/:id"
                    element={<Navigate to="/snapshots/partial/:slug/:id" />}
                  />
                  <Route
                    path=":slug/snapshots/:id/restore"
                    element={
                      <Navigate to="/snapshots/partial/:slug/:id/restore" />
                    }
                  />

                  <Route element={<NotFound />} />
                </Route>
                <Route
                  path="/restore-completed"
                  element={
                    <RestoreCompleted
                      logo={state.appLogo}
                      fetchingMetadata={state.fetchingMetadata}
                    />
                  }
                />
              </Routes>
            </div>
          </div>
          <div className="flex-auto Footer-wrapper u-width--full">
            <Footer appsList={state.appsList} />
          </div>
        </ToastProvider>
      </ThemeContext.Provider>

      {state.showUpgradeStatusModal ? (
        <Modal
          isOpen={state.showUpgradeStatusModal}
          onRequestClose={() => {
            // cannot close the modal while upgrading
            if (state.upgradeStatus === "upgrade-failed") {
              setState({ showUpgradeStatusModal: false });
            }
          }}
          shouldReturnFocusAfterClose={false}
          contentLabel="Upgrade status modal"
          ariaHideApp={false}
          className="Modal DefaultSize"
        >
          <UpgradeStatusModal
            status={state.upgradeStatus}
            message={state.upgradeMessage}
            appSlug={state.upgradeAppSlug}
            refetchStatus={fetchUpgradeStatus}
            closeModal={() => setState({ showUpgradeStatusModal: false })}
            connectionTerminated={state.connectionTerminated}
            setTerminatedState={(status: boolean) =>
              setState({ connectionTerminated: status })
            }
          />
        </Modal>
      ) : (
        <Modal
          isOpen={state.connectionTerminated}
          onRequestClose={undefined}
          shouldReturnFocusAfterClose={false}
          contentLabel="Connection terminated modal"
          ariaHideApp={false}
          className="Modal DefaultSize"
        >
          <ConnectionTerminated
            connectionTerminated={state.connectionTerminated}
            appLogo={state.appLogo}
            setTerminatedState={(status: boolean) =>
              setState({ connectionTerminated: status })
            }
          />
        </Modal>
      )}
    </QueryClientProvider>
  );
};
export { ThemeContext, Root };
