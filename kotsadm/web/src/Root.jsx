import { hot } from "react-hot-loader/root";
import Favicon from "react-favicon";
import React, { Component } from "react";
import { createBrowserHistory } from "history";
import { Switch, Route, Redirect, Router } from "react-router-dom";
import { ApolloProvider } from "react-apollo";
import { Helmet } from "react-helmet";
import Modal from "react-modal";
import find from "lodash/find";
import ConnectionTerminated from "./ConnectionTerminated";
import GitOps from "././components/clusters/GitOps";
import Snapshots from "./components/snapshots/Snapshots";
import PreflightResultPage from "./components/PreflightResultPage";
import Redactors from "./components/redactors/Redactors";
import EditRedactor from "./components/redactors/EditRedactor";
import AppConfig from "./components/apps/AppConfig";
import AppDetailPage from "./components/apps/AppDetailPage";
import ClusterNodes from "./components/apps/ClusterNodes";
import UnsupportedBrowser from "./components/static/UnsupportedBrowser";
import NotFound from "./components/static/NotFound";
import { Utilities } from "./utilities/utilities";
import { ShipClientGQL } from "./ShipClientGQL";
import SecureAdminConsole from "./components/SecureAdminConsole";
import RestoreCompleted from "./components/RestoreCompleted";

import { listApps } from "@src/queries/AppsQueries";
import Footer from "./components/shared/Footer";
import NavBar from "./components/shared/NavBar";

// Import Ship Init component CSS first
import "@replicatedhq/ship-init/dist/styles.css";
import "./scss/index.scss";
import UploadLicenseFile from "./components/UploadLicenseFile";
import UploadAirgapBundle from "./components/UploadAirgapBundle";
import connectHistory from "./services/matomo";

const INIT_SESSION_ID_STORAGE_KEY = "initSessionId";

let browserHistory = createBrowserHistory();
let history = connectHistory(browserHistory);

/**
 * Create our GraphQL Client
 */
const GraphQLClient = ShipClientGQL(
  window.env.GRAPHQL_ENDPOINT,
  window.env.API_ENDPOINT,
  () => Utilities.getToken()
);

class ProtectedRoute extends Component {
  render() {
    const redirectURL = `/secure-console?next=${this.props.location.pathname}${this.props.location.search}`;

    return (
      <Route path={this.props.path} render={(innerProps) => {
        if (Utilities.isLoggedIn()) {
          if (this.props.component) {
            return <this.props.component {...innerProps} />;
          }
          return this.props.render(innerProps);
        }
        return <Redirect to={redirectURL} />;
      }} />
    );
  }
}

const ThemeContext = React.createContext({
  setThemeState: () => { },
  getThemeState: () => ({}),
  clearThemeState: () => { }
});

class Root extends Component {
  state = {
    listApps: [],
    appLogo: null,
    selectedAppName: null,
    appNameSpace: null,
    fetchingMetadata: false,
    initSessionId: Utilities.localStorageEnabled()
      ? localStorage.getItem(INIT_SESSION_ID_STORAGE_KEY)
      : "",
    themeState: {
      navbarLogo: null,
    },
    rootDidInitialWatchFetch: false,
    connectionTerminated: false,
  };
  /**
   * Sets the Theme State for the whole application
   * @param {Object} newThemeState - Object to set for new theme state
   * @param {Function} callback - callback to run like in setState()'s callback
   */
  setThemeState = (newThemeState, callback) => {
    this.setState({
      themeState: { ...newThemeState }
    }, callback);
  }

  /**
   * Gets the current theme state of the app
   * @return {Object}
   */
  getThemeState = () => {
    return this.state.themeState;
  }

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
      themeState: { ...EMPTY_THEME_STATE }
    });
  }

  handleActiveInitSession = (initSessionId) => {
    if (Utilities.localStorageEnabled()) {
      localStorage.setItem(INIT_SESSION_ID_STORAGE_KEY, initSessionId)
    }
    this.setState({ initSessionId })
  }

  handleActiveInitSessionCompleted = () => {
    if (Utilities.localStorageEnabled()) {
      localStorage.removeItem(INIT_SESSION_ID_STORAGE_KEY);
    }
    this.setState({ initSessionId: "" });
  }

  refetchListApps = async () => {
    const apps = await GraphQLClient.query({
      query: listApps,
      fetchPolicy: "no-cache"
    }).catch(error => {
      throw error;
    });

    this.setState({
      listApps: apps.data.listApps.kotsApps,
      rootDidInitialWatchFetch: true
    });

    return apps.data.listApps.kotsApps;
  }

  fetchKotsAppMetadata = async () => {
    this.setState({ fetchingMetadata: true });

    fetch(`${window.env.API_ENDPOINT}/metadata`, {
      headers: {
        "Content-Type": "application/json",
        "Accept": "application/json",
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
          selectedAppName: data.name,
          appNameSpace: data.namespace,
          isKurlEnabled: data.isKurlEnabled,
          fetchingMetadata: false
        });
      })
      .catch((err) => {
        this.setState({ fetchingMetadata: false });
        throw err;
      });

  }

  ping = async (tries = 0) => {
    await fetch(`${window.env.API_ENDPOINT}/ping`, {
    }).then(() => {
      this.setState({ connectionTerminated: false });
    }).catch(() => {
      if (tries < 2) {
        setTimeout(() => {
          this.ping(tries + 1);
        }, 1000);
        return;
      }
      this.setState({ connectionTerminated: true });
    });
  }

  onRootMounted = () => {
    this.fetchKotsAppMetadata();
    this.ping();

    if (Utilities.isLoggedIn()) {
      this.refetchListApps().then(listApps => {
        if (listApps.length > 0 && window.location.pathname === "/apps") {
          const { slug } = listApps[0];
          history.replace(`/app/${slug}`);
        }
      });
    }
  }

  componentDidMount = async () => {
    this.onRootMounted();
    this.interval = setInterval(async () => await this.ping(), 10000);
  }

  componentDidUpdate = async (lastProps, lastState) => {
    if (this.state.connectionTerminated !== lastState.connectionTerminated) {
      if (this.interval) {
        clearInterval(this.interval);
      }
      if (!this.state.connectionTerminated) {
        this.interval = setInterval(async () => await this.ping(), 10000);
      }
    }
  }

  componentWillUnmount() {
    clearInterval(this.interval);
  }

  isGitOpsSupported = () => {
    const apps = this.state.listApps;
    return !!find(apps, app => app.isGitOpsSupported);
  }

  isSnapshotsSupported = () => {
    const apps = this.state.listApps;
    return !!find(apps, app => app.allowSnapshots);
  }

  render() {
    const {
      themeState,
      listApps,
      rootDidInitialWatchFetch,
      connectionTerminated,
    } = this.state;

    return (
      <div className="flex-column flex1">
        <Helmet>
          <meta httpEquiv="Cache-Control" content="no-cache, no-store, must-revalidate" />
          <meta httpEquiv="Pragma" content="no-cache" />
          <meta httpEquiv="Expires" content="0" />
          {this.state.appLogo &&
            <link rel="shortcut icon" href={this.state.appLogo} />
          }
        </Helmet>
        <ApolloProvider client={GraphQLClient}>
          <ThemeContext.Provider value={{
            setThemeState: this.setThemeState,
            getThemeState: this.getThemeState,
            clearThemeState: this.clearThemeState
          }}>
            <Router history={history}>
              <div className="flex-column flex1">
                <NavBar
                  logo={themeState.navbarLogo || this.state.appLogo}
                  refetchListApps={this.refetchListApps}
                  fetchingMetadata={this.state.fetchingMetadata}
                  isKurlEnabled={this.state.isKurlEnabled}
                  isGitOpsSupported={this.isGitOpsSupported()}
                  listApps={listApps}
                  isSnapshotsSupported={this.isSnapshotsSupported()}
                />
                <div className="flex1 flex-column u-overflow--auto">
                  <Switch>

                    <Route exact path="/" component={() => <Redirect to={Utilities.isLoggedIn() ? "/apps" : "/secure-console"} />} />
                    <Route exact path="/crashz" render={() => {
                      const Crashz = () => {
                        throw new Error("Crashz!");
                      };
                      return <Crashz />;

                    }} />
                    <ProtectedRoute path="/preflight" render={props => <PreflightResultPage {...props} logo={this.state.appLogo} appName={this.state.selectedAppName} fromLicenseFlow={true} refetchListApps={this.refetchListApps} />} />
                    <ProtectedRoute exact path="/:slug/config" render={props => <AppConfig {...props} fromLicenseFlow={true} refetchListApps={this.refetchListApps} />} />
                    <Route exact path="/secure-console" render={props => <SecureAdminConsole {...props} logo={this.state.appLogo} appName={this.state.selectedAppName} onLoginSuccess={this.refetchListApps} fetchingMetadata={this.state.fetchingMetadata} />} />
                    <ProtectedRoute exact path="/upload-license" render={props => <UploadLicenseFile {...props} logo={this.state.appLogo} appsListLength={listApps?.length} appName={this.state.selectedAppName} fetchingMetadata={this.state.fetchingMetadata} onUploadSuccess={this.refetchListApps} />} />
                    <ProtectedRoute exact path="/:slug/airgap" render={props => <UploadAirgapBundle {...props} showRegistry={true} logo={this.state.appLogo} appsListLength={listApps?.length} appName={this.state.selectedAppName} onUploadSuccess={this.refetchListApps} fetchingMetadata={this.state.fetchingMetadata} />} />
                    <ProtectedRoute exact path="/:slug/airgap-bundle" render={props => <UploadAirgapBundle {...props} showRegistry={false} logo={this.state.appLogo} appsListLength={listApps?.length} appName={this.state.selectedAppName} onUploadSuccess={this.refetchListApps} fetchingMetadata={this.state.fetchingMetadata} />} />
                    <Route path="/unsupported" component={UnsupportedBrowser} />
                    <ProtectedRoute path="/cluster/manage" render={(props) => <ClusterNodes {...props} appName={this.state.selectedAppName} />} />
                    <ProtectedRoute path="/gitops" render={(props) => <GitOps {...props} appName={this.state.selectedAppName} />} />
                    <ProtectedRoute path="/snapshots" render={(props) => <Snapshots {...props} appName={this.state.selectedAppName} />} />
                    {/* <ProtectedRoute exact path="/redactors" render={(props) => <Redactors {...props} appName={this.state.selectedAppName} />} />
                    <ProtectedRoute exact path="/redactors/new" render={(props) => <EditRedactor {...props} appName={this.state.selectedAppName} />} />
                    <ProtectedRoute exact path="/redactors/:slug" render={(props) => <EditRedactor {...props} appName={this.state.selectedAppName} />} /> */}
                    <ProtectedRoute
                      path={["/apps", "/app/:slug/:tab?"]}
                      render={
                        props => (
                          <AppDetailPage
                            {...props}
                            rootDidInitialAppFetch={rootDidInitialWatchFetch}
                            listApps={listApps}
                            refetchListApps={this.refetchListApps}
                            onActiveInitSession={this.handleActiveInitSession}
                            appNameSpace={this.state.appNameSpace}
                            appName={this.state.selectedAppName}
                          />
                        )
                      }
                    />
                    <Route exact path="/restore-completed" render={props => <RestoreCompleted {...props} logo={this.state.appLogo} fetchingMetadata={this.state.fetchingMetadata} />} />
                    <Route component={NotFound} />
                  </Switch>
                </div>
                <div className="flex-auto Footer-wrapper u-width--full">
                  <Footer />
                </div>
              </div>
            </Router>
          </ThemeContext.Provider>
        </ApolloProvider>
        {connectionTerminated &&
          <Modal
            isOpen={connectionTerminated}
            onRequestClose={undefined}
            shouldReturnFocusAfterClose={false}
            contentLabel="Connection terminated modal"
            ariaHideApp={false}
            className="ConnectionTerminated--wrapper Modal DefaultSize"
          >
            <ConnectionTerminated gqlClient={GraphQLClient} connectionTerminated={this.state.connectionTerminated} appLogo={this.state.appLogo} setTerminatedState={(status) => this.setState({ connectionTerminated: status })} />
          </Modal>
        }
      </div>
    );
  }
}
export { ThemeContext };
export default hot(Root);
