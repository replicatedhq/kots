import { hot } from "react-hot-loader/root";
import React, { Suspense, lazy } from "react";
import { createBrowserHistory } from "history";
import ReactPiwik from "react-piwik";
import { Switch, Route, Redirect, Router } from "react-router-dom";
import { ApolloProvider } from "react-apollo";
import { Utilities } from "./utilities/utilities";
import { ShipClientGQL } from "./ShipClientGQL";
import { Helmet } from "react-helmet";

import Footer from "./components/shared/Footer";
import NavBar from "./components/shared/NavBar";
import Loader from "./components/shared/Loader";

// Import Ship Init component CSS first
import "@replicatedhq/ship-init/dist/styles.css";
import "./scss/index.scss";

const Login = lazy(() => import("./components/Login"));
const Signup = lazy(() => import("./components/Signup"));
const GitHubAuth = lazy(() => import("./components/github_auth/GitHubAuth"));
const GitHubInstall = lazy(() => import("./components/github_install/GitHubInstall"));
const Clusters = lazy(() => import("./components/clusters/Clusters"));
const CreateCluster = lazy(() => import("./components/clusters/CreateCluster"));
const VersionHistory = lazy(() => import("./components/watches/VersionHistory"));
const DiffShipReleases = lazy(() => import("./components/watches/DiffShipReleases"));
const DiffGitHubReleases = lazy(() => import("./components/watches/DiffGitHubReleases"));
const StateFileViewer = lazy(() => import("./components/state/StateFileViewer"));
const Ship = lazy(() => import("./components/Ship"));
const ShipInitPre = lazy(() => import("./components/ShipInitPre"));
const ShipUnfork = lazy(() => import("./components/ShipUnfork"));
const ShipInitCompleted = lazy(() => import("./components/ShipInitCompleted"));
const WatchDetailPage = lazy(() => import("./components/watches/WatchDetailPage"));
const ClusterScope = lazy(() => import("./components/clusterscope/ClusterScope"));
const UnsupportedBrowser = lazy(() => import("./components/static/UnsupportedBrowser"));
const NotFound = lazy(() => import("./components/static/NotFound"));
const ReplicatedGraphiQL = lazy(() => import("./components/ReplicatedGraphiQL"));

const INIT_SESSION_ID_STORAGE_KEY = "initSessionId";

let history = createBrowserHistory();
if(process.env.NODE_ENV === "production") {
  const piwik = new ReactPiwik({
    url: "https://data-2.replicated.com",
    siteId: 6,
    trackErrors: true,
    jsFilename: "js/",
  });
  history = piwik.connectToHistory(history);
}

class ProtectedRoute extends React.Component {
  render() {
    return (
      <Route path={this.props.path} render={(innerProps) => {
        if (Utilities.isLoggedIn()) {
          if (this.props.component) {
            return <this.props.component {...innerProps} />;
          }
          return this.props.render(innerProps);
        }
        return <Redirect to={`/login?next=${this.props.location.pathname}${this.props.location.search}`} />;
      }} />
    )
  }
}

const ThemeContext = React.createContext({
  setThemeState: () => {},
  getThemeState: () => ({}),
  clearThemeState: () => {}
});

class Root extends React.Component {
  state = {
    initSessionId: Utilities.localStorageEnabled()
      ? localStorage.getItem(INIT_SESSION_ID_STORAGE_KEY)
      : "",
    themeState: {
      navbarLogo: null
    }
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
      navbarLogo: null
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

  handleInitCompletion = (history) =>
    () => {
      history.push("/watch/init/complete");
    }
  handleUpdateCompletion = history =>
    () => {
      history.push("/watches");
      this.handleActiveInitSessionCompleted()
    }

  render() {
    const { initSessionId, themeState } = this.state;

    return (
      <div className="flex-column flex1">
        <Helmet>
          <meta httpEquiv="Cache-Control" content="no-cache, no-store, must-revalidate" />
          <meta httpEquiv="Pragma" content="no-cache" />
          <meta httpEquiv="Expires" content="0" />
        </Helmet>
        <ApolloProvider client={ShipClientGQL(window.env.GRAPHQL_ENDPOINT, window.env.REST_ENDPOINT, () => { return Utilities.getToken(); })}>
          <ThemeContext.Provider value={{
            setThemeState: this.setThemeState,
            getThemeState: this.getThemeState,
            clearThemeState: this.clearThemeState
          }}>
            <Router history={history}>
              <div className="flex-column flex1">
                <NavBar logo={themeState.navbarLogo}/>
                <Suspense fallback={<div className="flex-column flex1 alignItems--center justifyContent--center"><Loader size="60" /></div>}>
                  <div className="flex-1-auto flex-column u-overflow--hidden">
                    <Switch>
                      <Route exact path="/" component={() => <Redirect to={Utilities.isLoggedIn() ? "/watches" : "/login"} />} />
                      <Route exact path="/login" component={Login} />
                      <Route exact path="/signup" component={Signup} />
                      <Route path="/auth/github" component={GitHubAuth} />
                      <Route path="/install/github" component={GitHubInstall} />
                      <Route path="/clusterscope" component={ClusterScope} />
                      <Route path="/unsupported" component={UnsupportedBrowser} />
                      {window.env.ENVIRONMENT === "development" &&
                        <ProtectedRoute path="/graphiql" component={ReplicatedGraphiQL} />
                      }
                      <ProtectedRoute path="/clusters" render={(props) => <Clusters {...props} />} />
                      <ProtectedRoute path="/cluster/create" render={(props) => <CreateCluster {...props} />} />
                      <ProtectedRoute path="/watches" render={(props) => <WatchDetailPage {...props} onActiveInitSession={this.handleActiveInitSession} />} />
                      <ProtectedRoute path="/watch/:owner/:slug/history/compare/:org/:repo/:branch/:rootPath/:firstSeqNumber/:secondSeqNumber" component={DiffGitHubReleases} />
                      <ProtectedRoute path="/watch/:owner/:slug/history/compare/:firstSeqNumber/:secondSeqNumber" component={DiffShipReleases} />
                      <ProtectedRoute path="/watch/:owner/:slug/history" component={VersionHistory} />
                      <ProtectedRoute path="/watch/create/init" render={(props) => <ShipInitPre {...props} onActiveInitSession={this.handleActiveInitSession} />} />
                      <ProtectedRoute path="/watch/create/unfork" render={(props) => <ShipUnfork {...props} onActiveInitSession={this.handleActiveInitSession} />} />
                      <ProtectedRoute path="/watch/create/state" component={() =>
                        <StateFileViewer
                          isNew={true}
                          headerText="Add your application's state.json file"
                          subText={<span>Paste in the state.json that was generated by Replicated Ship. If you need help finding your state.json file or you have not initialized your app using Replicated Ship, <a href="https://ship.replicated.com/docs/ship-init/storing-state/" target="_blank" rel="noopener noreferrer" className="replicated-link">check out our docs.</a></span>}
                        />
                      } />
                      <ProtectedRoute
                        path="/watch/init/complete"
                        render={
                          (props) => <ShipInitCompleted
                            {...props}
                            initSessionId={initSessionId}
                            onActiveInitSessionCompleted={this.handleActiveInitSessionCompleted}
                          />
                        }
                      />
                      <ProtectedRoute path="/watch/:owner/:slug/:tab?" render={(props) => <WatchDetailPage {...props} onActiveInitSession={this.handleActiveInitSession} />} />
                      <ProtectedRoute
                        path="/ship/init"
                        render={
                          (props) => <Ship
                            {...props}
                            rootURL={window.env.SHIPINIT_ENDPOINT}
                            initSessionId={initSessionId}
                            onCompletion={this.handleInitCompletion(props.history)}
                          />
                        }
                      />
                      <ProtectedRoute
                        path="/ship/update"
                        render={
                          (props) => <Ship
                            {...props}
                            rootURL={window.env.SHIPUPDATE_ENDPOINT}
                            initSessionId={initSessionId}
                            onCompletion={this.handleUpdateCompletion(props.history)}
                          />
                        }
                      />
                      <ProtectedRoute
                        path="/ship/edit"
                        render={
                          (props) => <Ship
                            {...props}
                            rootURL={window.env.SHIPEDIT_ENDPOINT}
                            initSessionId={initSessionId}
                            onCompletion={this.handleUpdateCompletion(props.history)}
                          />
                        }
                      />
                      <Route component={NotFound} />
                    </Switch>
                  </div>
                </Suspense>
                <div className="flex-auto Footer-wrapper u-width--full">
                  <Footer />
                </div>
              </div>
            </Router>
          </ThemeContext.Provider>
        </ApolloProvider>
      </div>
    );
  }
}
export { ThemeContext };
export default hot(Root);
