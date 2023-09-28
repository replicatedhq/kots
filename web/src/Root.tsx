import React, { useReducer, useEffect } from "react";
import { createBrowserHistory } from "history";
import { Route, Routes, Navigate, useNavigate } from "react-router-dom";
import { Helmet } from "react-helmet";
import Modal from "react-modal";
import find from "lodash/find";
import ConnectionTerminated from "./ConnectionTerminated";
import GitOps from "./components/clusters/GitOps";
import PreflightResultPage from "./components/PreflightResultPage";
import AppConfig from "./features/AppConfig/components/AppConfig";
import { AppDetailPage } from "./components/apps/AppDetailPage";
import ClusterNodes from "./components/apps/ClusterNodes";
import UnsupportedBrowser from "./components/static/UnsupportedBrowser";
import NotFound from "./components/static/NotFound";
import { Utilities, parseUpstreamUri } from "./utilities/utilities";
import fetch from "./utilities/fetchWithTimeout";
import { SecureAdminConsole } from "@features/Auth";
import UploadLicenseFile from "./components/UploadLicenseFile";
import BackupRestore from "./components/BackupRestore";
import UploadAirgapBundle from "./components/UploadAirgapBundle";
import RestoreCompleted from "./components/RestoreCompleted";
import Access from "./components/identity/Access";
import SnapshotsWrapper from "./components/snapshots/SnapshotsWrapper";
import { QueryClient, QueryClientProvider } from "react-query";
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

// scss
import "./scss/index.scss";
// tailwind
import "./index.css";
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

// react-query client
const queryClient = new QueryClient();

const INIT_SESSION_ID_STORAGE_KEY = "initSessionId";

let browserHistory = createBrowserHistory();
let history = connectHistory(browserHistory);

// TODO:  pull in the react router hook

const ThemeContext = React.createContext({
  setThemeState: (themeState?: ThemeState) => {
    console.log("setThemeState used before being set", themeState);
  },
  getThemeState: (): ThemeState => ({ navbarLogo: null }),
  clearThemeState: () => {},
});

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
  errLoggingOut: string;
  featureFlags: object;
  fetchingMetadata: boolean;
  initSessionId: string | null;
  isHelmManaged: boolean;
  selectedAppName: string | null;
  snapshotInProgressApps: string[];
  themeState: ThemeState;
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
      connectionTerminated: false,
      errLoggingOut: "",
      featureFlags: {},
      isHelmManaged: false,
      fetchingMetadata: false,
      initSessionId: Utilities.localStorageEnabled()
        ? localStorage.getItem(INIT_SESSION_ID_STORAGE_KEY)
        : "",
      selectedAppName: null,
      snapshotInProgressApps: [],
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

  // TODO: delete if not used
  // const handleActiveInitSessionCompleted = () => {
  //   if (Utilities.localStorageEnabled()) {
  //     localStorage.removeItem(INIT_SESSION_ID_STORAGE_KEY);
  //   }
  //   setState({ initSessionId: "" });
  // };

  const checkIsHelmManaged = async () => {
    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/is-helm-managed`, {
        headers: {
          "Content-Type": "application/json",
        },
        method: "GET",
        credentials: "include",
      });
      if (res.ok && res.status === 200) {
        const response = await res.json();
        setState({ isHelmManaged: response.isHelmManaged });
        return response.isHelmManaged;
      } else {
        setState({ isHelmManaged: false });
      }
      return false;
    } catch (err) {
      console.log(err);
      setState({ isHelmManaged: false });
      return false;
    }
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
      .then(async (result) => {
        if (result.status === 401) {
          Utilities.logoutUser();
          return;
        }
        const body = await result.json();
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

  const onRootMounted = () => {
    fetchKotsAppMetadata();
    if (Utilities.isLoggedIn()) {
      ping();
      checkIsHelmManaged();
      getAppsList().then((appsList) => {
        if (appsList?.length > 0 && window.location.pathname === "/apps") {
          const { slug } = appsList[0];
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
            isGitOpsSupported={isGitOpsSupported()}
            isIdentityServiceSupported={isIdentityServiceSupported()}
            appsList={state.appsList}
            onLogoutError={onLogoutError}
            isSnapshotsSupported={isSnapshotsSupported()}
            errLoggingOut={state.errLoggingOut}
            isHelmManaged={state.isHelmManaged}
          />
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
                    checkIsHelmManaged={checkIsHelmManaged}
                    navigate={navigate}
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
                  />
                }
              />
              <Route
                path="/:slug/config"
                element={
                  <AppConfig
                    fromLicenseFlow={true}
                    refetchAppsList={getAppsList}
                    isHelmManaged={state.isHelmManaged}
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
                  />
                }
              />
              <Route path="/install-with-helm" element={<InstallWithHelm />} />
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
              <Route
                path="/cluster/manage"
                element={<ClusterNodes appName={state.selectedAppName} />}
              />
              <Route
                path="/gitops"
                element={<GitOps appName={state.selectedAppName || ""} />}
              />
              <Route
                path="/access/:tab?"
                element={
                  <Access
                    isKurlEnabled={state.adminConsoleMetadata?.isKurl || false}
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
                    appsList={state.appsList}
                  />
                }
              >
                <Route
                  index
                  element={
                    <Snapshots
                      isKurlEnabled={state.adminConsoleMetadata?.isKurl}
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
                      appsList={state.appsList}
                    />
                  }
                />
                <Route
                  path="details/:id"
                  element={
                    <SnapshotDetails
                      isKurlEnabled={state.adminConsoleMetadata?.isKurl}
                      appsList={state.appsList}
                    />
                  }
                />
                <Route path=":slug/:id/restore" element={<SnapshotRestore />} />
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
                    snapshotInProgressApps={state.snapshotInProgressApps}
                    ping={ping}
                    isHelmManaged={state.isHelmManaged}
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
                    snapshotInProgressApps={state.snapshotInProgressApps}
                    ping={ping}
                    isHelmManaged={state.isHelmManaged}
                  />
                }
              >
                <Route path=":slug" element={<Dashboard />} />
                <Route
                  path=":slug/tree/:sequence?"
                  element={<DownstreamTree />}
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
                    />
                  }
                />
                <Route
                  path=":slug/config/:sequence"
                  element={
                    <AppConfig
                      fromLicenseFlow={false}
                      refetchAppsList={getAppsList}
                      isHelmManaged={state.isHelmManaged}
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
                  <Route index element={<SupportBundleList />} />
                  <Route path="generate" element={<GenerateSupportBundle />} />
                  <Route
                    path="analyze/:bundleSlug"
                    element={<SupportBundleAnalysis />}
                  >
                    <Route index element={<AnalyzerInsights />} />
                    <Route path={"contents/*"} element={<AnalyzerFileTree />} />
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
                <Route path=":slug/license" element={<AppLicense />} />
                <Route
                  path=":slug/registry-settings"
                  element={
                    <AppRegistrySettings
                    // app={selectedApp}
                    // updateCallback={refetchData}
                    />
                  }
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
          <div className="flex-auto Footer-wrapper u-width--full">
            <Footer appsList={state.appsList} />
          </div>
        </ToastProvider>
      </ThemeContext.Provider>
      <Modal
        isOpen={state.connectionTerminated}
        onRequestClose={undefined}
        shouldReturnFocusAfterClose={false}
        contentLabel="Connection terminated modal"
        ariaHideApp={false}
        className="ConnectionTerminated--wrapper Modal DefaultSize"
      >
        <ConnectionTerminated
          connectionTerminated={state.connectionTerminated}
          appLogo={state.appLogo}
          setTerminatedState={(status: boolean) =>
            setState({ connectionTerminated: status })
          }
        />
      </Modal>
    </QueryClientProvider>
  );
};
export { ThemeContext, Root };
