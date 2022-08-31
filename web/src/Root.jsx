import React, { Component, PureComponent } from "react";
import { createBrowserHistory } from "history";
import { Switch, Route, Redirect, Router } from "react-router-dom";
import { Helmet } from "react-helmet";
import Modal from "react-modal";
import find from "lodash/find";
import ConnectionTerminated from "./ConnectionTerminated";
import GitOps from "././components/clusters/GitOps";
import PreflightResultPage from "./components/PreflightResultPage";
import AppConfig from "./features/AppConfig/components/AppConfig";
import AppDetailPage from "./components/apps/AppDetailPage";
import ClusterNodes from "./components/apps/ClusterNodes";
import UnsupportedBrowser from "./components/static/UnsupportedBrowser";
import NotFound from "./components/static/NotFound";
import { Utilities, parseUpstreamUri } from "./utilities/utilities";
import fetch from "./utilities/fetchWithTimeout";
import SecureAdminConsole from "./components/SecureAdminConsole";
import UploadLicenseFile from "./components/UploadLicenseFile";
import BackupRestore from "./components/BackupRestore";
import UploadAirgapBundle from "./components/UploadAirgapBundle";
import RestoreCompleted from "./components/RestoreCompleted";
import Access from "./components/identity/Access";
import SnapshotsWrapper from "./components/snapshots/SnapshotsWrapper";
import { QueryClient, QueryClientProvider } from "react-query";

import Footer from "./components/shared/Footer";
import NavBar from "./components/shared/NavBar";

import "./scss/index.scss";
import connectHistory from "./services/matomo";

// react-query client
const queryClient = new QueryClient();

const INIT_SESSION_ID_STORAGE_KEY = "initSessionId";

let browserHistory = createBrowserHistory();
let history = connectHistory(browserHistory);

class ProtectedRoute extends Component {
  render() {
    const redirectURL = `/secure-console?next=${this.props.location.pathname}${this.props.location.search}`;

    return (
      <Route
        path={this.props.path}
        render={(innerProps) => {
          if (Utilities.isLoggedIn()) {
            if (this.props.component) {
              return <this.props.component {...innerProps} />;
            }
            return this.props.render(innerProps);
          }
          return <Redirect to={redirectURL} />;
        }}
      />
    );
  }
}

const ThemeContext = React.createContext({
  setThemeState: () => {},
  getThemeState: () => ({}),
  clearThemeState: () => {},
});

class Root extends PureComponent {
  state = {
    appsList: [],
    appLogo: null,
    appBrandingCss: "",
    selectedAppName: null,
    appNameSpace: null,
    appSlugFromMetadata: null,
    fetchingMetadata: false,
    initSessionId: Utilities.localStorageEnabled()
      ? localStorage.getItem(INIT_SESSION_ID_STORAGE_KEY)
      : "",
    themeState: {
      navbarLogo: null,
    },
    rootDidInitialWatchFetch: false,
    connectionTerminated: false,
    snapshotInProgressApps: [],
    errLoggingOut: "",
    isHelmManaged: false,
  };

  /**
   * Sets the Theme State for the whole application
   * @param {Object} newThemeState - Object to set for new theme state
   * @param {Function} callback - callback to run like in setState()'s callback
   */
  setThemeState = (newThemeState, callback) => {
    this.setState(
      {
        themeState: { ...newThemeState },
      },
      callback
    );
  };

  /**
   * Gets the current theme state of the app
   * @return {Object}
   */
  getThemeState = () => {
    return this.state.themeState;
  };

  /**
   * Clears the current theme state to nothing
   */
  clearThemeState = () => {
    /**
     * Reference object to a blank theme state
     */
    const EMPTY_THEME_STATE = {
      navbarLogo: null,
    };

    this.setState({
      themeState: { ...EMPTY_THEME_STATE },
    });
  };

  handleActiveInitSession = (initSessionId) => {
    if (Utilities.localStorageEnabled()) {
      localStorage.setItem(INIT_SESSION_ID_STORAGE_KEY, initSessionId);
    }
    this.setState({ initSessionId });
  };

  handleActiveInitSessionCompleted = () => {
    if (Utilities.localStorageEnabled()) {
      localStorage.removeItem(INIT_SESSION_ID_STORAGE_KEY);
    }
    this.setState({ initSessionId: "" });
  };

  checkIsHelmManaged = async () => {
    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/is-helm-managed`, {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });
      if (res.ok && res.status === 200) {
        const response = await res.json();
        this.setState({ isHelmManaged: response.isHelmManaged });
      } else {
        this.setState({ isHelmManaged: false });
      }
    } catch (err) {
      console.log(err);
      this.setState({ isHelmManaged: false });
    }
  };

  getPendingApp = async () => {
    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/pendingapp`, {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
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
      this.setState({
        pendingapp: app,
      });
      return app;
    } catch (err) {
      throw err;
    }
  };

  getAppsList = async () => {
    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/apps`, {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
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
      this.setState({
        appsList: apps,
        rootDidInitialWatchFetch: true,
      });
      return apps;
    } catch (err) {
      throw err;
    }
  };

  fetchKotsAppMetadata = async () => {
    this.setState({ fetchingMetadata: true });

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
          this.setState({ fetchingMetadata: false });
          return;
        }

        this.setState({
          appLogo: data.iconUri,
          appBrandingCss: data.brandingCss,
          selectedAppName: data.name,
          appSlugFromMetadata: parseUpstreamUri(data.upstreamUri),
          appNameSpace: data.namespace,
          adminConsoleMetadata: data.adminConsoleMetadata,
          featureFlags: data.consoleFeatureFlags,
          fetchingMetadata: false,
        });
      })
      .catch((err) => {
        this.setState({ fetchingMetadata: false });
        throw err;
      });
  };

  ping = async (tries = 0) => {
    let apps = this.state.appsList;
    const appSlugs = apps?.map((a) => a.slug);
    const url = `${process.env.API_ENDPOINT}/ping?slugs=${appSlugs}`;
    await fetch(
      url,
      {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
      },
      10000
    )
      .then(async (result) => {
        if (result.status === 401) {
          Utilities.logoutUser();
          return;
        }
        const body = await result.json();
        this.setState({
          connectionTerminated: false,
          snapshotInProgressApps: body.snapshotInProgressApps,
        });
      })
      .catch(() => {
        if (tries < 2) {
          setTimeout(() => {
            this.ping(tries + 1);
          }, 1000);
          return;
        }
        this.setState({
          connectionTerminated: true,
          snapshotInProgressApps: [],
        });
      });
  };

  onRootMounted = () => {
    this.fetchKotsAppMetadata();
    this.ping();
    this.checkIsHelmManaged();

    if (Utilities.isLoggedIn()) {
      this.getAppsList().then((appsList) => {
        if (appsList.length > 0 && window.location.pathname === "/apps") {
          const { slug } = appsList[0];
          history.replace(`/app/${slug}`);
        }
      });
    }
  };

  componentDidMount = async () => {
    this.onRootMounted();
    this.interval = setInterval(async () => await this.ping(), 10000);
  };

  componentDidUpdate = async (lastProps, lastState) => {
    if (this.state.connectionTerminated !== lastState.connectionTerminated) {
      if (this.interval) {
        clearInterval(this.interval);
      }
      if (!this.state.connectionTerminated) {
        this.interval = setInterval(async () => await this.ping(), 10000);
      }
    }
  };

  componentWillUnmount() {
    clearInterval(this.interval);
  }

  isGitOpsSupported = () => {
    const apps = this.state.appsList;
    return !!find(apps, (app) => app.isGitOpsSupported);
  };

  isIdentityServiceSupported = () => {
    const apps = this.state.appsList;
    return !!find(apps, (app) => app.isIdentityServiceSupported);
  };

  isGeoaxisSupported = () => {
    const apps = this.state.appsList;
    return !!find(apps, (app) => app.isGeoaxisSupported);
  };

  isSnapshotsSupported = () => {
    const apps = this.state.appsList;
    return !!find(apps, (app) => app.allowSnapshots);
  };

  onLogoutError = (message) => {
    this.setState({
      errLoggingOut: message,
    });
  };

  render() {
    const {
      themeState,
      appsList,
      rootDidInitialWatchFetch,
      connectionTerminated,
      errLoggingOut,
      isHelmManaged,
    } = this.state;

    return (
      <QueryClientProvider client={queryClient}>
        <Helmet>
          <meta
            httpEquiv="Cache-Control"
            content="no-cache, no-store, must-revalidate"
          />
          <meta httpEquiv="Pragma" content="no-cache" />
          <meta httpEquiv="Expires" content="0" />
          {this.state.appLogo && (
            <link rel="icon" type="image/png" href={this.state.appLogo} />
          )}
          {this.state.appBrandingCss && (
            <style rel="stylesheet" type="text/css" id="kots-branding-css">{ this.state.appBrandingCss }</style>
          )}

        </Helmet>
        <ThemeContext.Provider
          value={{
            setThemeState: this.setThemeState,
            getThemeState: this.getThemeState,
            clearThemeState: this.clearThemeState,
          }}
        >
          <Router history={history}>
            <NavBar
              logo={themeState.navbarLogo || this.state.appLogo}
              refetchAppsList={this.getAppsList}
              fetchingMetadata={this.state.fetchingMetadata}
              isKurlEnabled={this.state.adminConsoleMetadata?.isKurl}
              isGitOpsSupported={this.isGitOpsSupported()}
              isIdentityServiceSupported={this.isIdentityServiceSupported()}
              appsList={appsList}
              onLogoutError={this.onLogoutError}
              isSnapshotsSupported={this.isSnapshotsSupported()}
              errLoggingOut={errLoggingOut}
              isHelmManaged={isHelmManaged}
            />
            <div className="flex1 flex-column u-overflow--auto">
              <Switch>
                <Route
                  exact
                  path="/"
                  component={() => (
                    <Redirect
                      to={Utilities.isLoggedIn() ? "/apps" : "/secure-console"}
                    />
                  )}
                />
                <Route
                  exact
                  path="/crashz"
                  render={() => {
                    const Crashz = () => {
                      throw new Error("Crashz!");
                    };
                    return <Crashz />;
                  }}
                />
                <ProtectedRoute
                  path="/:slug/preflight"
                  render={(props) => (
                    <PreflightResultPage
                      {...props}
                      logo={this.state.appLogo}
                      appName={this.state.selectedAppName}
                      appsList={appsList}
                      fromLicenseFlow={true}
                      refetchAppsList={this.getAppsList}
                    />
                  )}
                />
                <ProtectedRoute
                  exact
                  path="/:slug/config"
                  render={(props) => (
                    <AppConfig
                      {...props}
                      fromLicenseFlow={true}
                      refetchAppsList={this.getAppsList}
                    />
                  )}
                />
                <Route
                  exact
                  path="/secure-console"
                  render={(props) => (
                    <SecureAdminConsole
                      {...props}
                      logo={this.state.appLogo}
                      appName={this.state.selectedAppName}
                      pendingApp={this.getPendingApp}
                      onLoginSuccess={this.getAppsList}
                      fetchingMetadata={this.state.fetchingMetadata}
                      checkIsHelmManaged={this.checkIsHelmManaged}
                    />
                  )}
                />
                <ProtectedRoute
                  exact
                  path="/upload-license"
                  render={(props) => (
                    <UploadLicenseFile
                      {...props}
                      logo={this.state.appLogo}
                      appsListLength={appsList?.length}
                      appName={this.state.selectedAppName}
                      appSlugFromMetadata={this.state.appSlugFromMetadata}
                      fetchingMetadata={this.state.fetchingMetadata}
                      onUploadSuccess={this.getAppsList}
                    />
                  )}
                />
                <ProtectedRoute
                  exact
                  path="/restore"
                  render={(props) => (
                    <BackupRestore
                      {...props}
                      logo={this.state.appLogo}
                      appName={this.state.selectedAppName}
                      appsListLength={appsList?.length}
                      fetchingMetadata={this.state.fetchingMetadata}
                    />
                  )}
                />
                <ProtectedRoute
                  exact
                  path="/:slug/airgap"
                  render={(props) => (
                    <UploadAirgapBundle
                      {...props}
                      showRegistry={true}
                      logo={this.state.appLogo}
                      appsListLength={appsList?.length}
                      appName={this.state.selectedAppName}
                      onUploadSuccess={this.getAppsList}
                      fetchingMetadata={this.state.fetchingMetadata}
                    />
                  )}
                />
                <ProtectedRoute
                  exact
                  path="/:slug/airgap-bundle"
                  render={(props) => (
                    <UploadAirgapBundle
                      {...props}
                      showRegistry={false}
                      logo={this.state.appLogo}
                      appsListLength={appsList?.length}
                      appName={this.state.selectedAppName}
                      onUploadSuccess={this.getAppsList}
                      fetchingMetadata={this.state.fetchingMetadata}
                    />
                  )}
                />
                <Route path="/unsupported" component={UnsupportedBrowser} />
                <ProtectedRoute
                  path="/cluster/manage"
                  render={(props) => (
                    <ClusterNodes
                      {...props}
                      appName={this.state.selectedAppName}
                    />
                  )}
                />
                <ProtectedRoute
                  path="/gitops"
                  render={(props) => (
                    <GitOps {...props} appName={this.state.selectedAppName} />
                  )}
                />
                <ProtectedRoute
                  path="/access/:tab?"
                  render={(props) => (
                    <Access
                      {...props}
                      appName={this.state.selectedAppName}
                      isKurlEnabled={this.state.adminConsoleMetadata?.isKurl}
                      isGeoaxisSupported={this.isGeoaxisSupported()}
                    />
                  )}
                />
                <ProtectedRoute
                  path={["/snapshots/:tab?"]}
                  render={(props) => (
                    <SnapshotsWrapper
                      {...props}
                      appName={this.state.selectedAppName}
                      isKurlEnabled={this.state.adminConsoleMetadata?.isKurl}
                      appsList={appsList}
                    />
                  )}
                />
                <ProtectedRoute
                  path={["/apps", "/app/:slug/:tab?"]}
                  render={(props) => (
                    <AppDetailPage
                      {...props}
                      rootDidInitialAppFetch={rootDidInitialWatchFetch}
                      appsList={appsList}
                      refetchAppsList={this.getAppsList}
                      onActiveInitSession={this.handleActiveInitSession}
                      appNameSpace={this.state.appNameSpace}
                      appName={this.state.selectedAppName}
                      snapshotInProgressApps={this.state.snapshotInProgressApps}
                      featureFlags={this.state.featureFlags}
                      ping={this.ping}
                      isHelmManaged={isHelmManaged}
                    />
                  )}
                />
                <Route
                  exact
                  path="/restore-completed"
                  render={(props) => (
                    <RestoreCompleted
                      {...props}
                      logo={this.state.appLogo}
                      fetchingMetadata={this.state.fetchingMetadata}
                    />
                  )}
                />
                <Route component={NotFound} />
              </Switch>
            </div>
            <div className="flex-auto Footer-wrapper u-width--full">
              <Footer appsList={appsList} />
            </div>
          </Router>
        </ThemeContext.Provider>
        <Modal
          isOpen={connectionTerminated}
          onRequestClose={undefined}
          shouldReturnFocusAfterClose={false}
          contentLabel="Connection terminated modal"
          ariaHideApp={false}
          className="ConnectionTerminated--wrapper Modal DefaultSize"
        >
          <ConnectionTerminated
            connectionTerminated={this.state.connectionTerminated}
            appLogo={this.state.appLogo}
            setTerminatedState={(status) =>
              this.setState({ connectionTerminated: status })
            }
          />
        </Modal>
      </QueryClientProvider>
    );
  }
}
export { ThemeContext };
export default Root;
