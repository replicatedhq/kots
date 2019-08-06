import { hot } from "react-hot-loader/root";
import React, { Component } from "react";
import { createBrowserHistory } from "history";
import ReactPiwik from "react-piwik";
import { Switch, Route, Redirect, Router } from "react-router-dom";
import { ApolloProvider } from "react-apollo";
import { Helmet } from "react-helmet";
import Login from "./components/Login";
import Signup from "./components/Signup";
import GitHubAuth from "./components/github_auth/GitHubAuth";
import GitHubInstall from "./components/github_install/GitHubInstall";
import Clusters from "./components/clusters/Clusters";
import CreateCluster from "./components/clusters/CreateCluster";
import VersionHistory from "./components/watches/VersionHistory";
import DiffShipReleases from "./components/watches/DiffShipReleases";
import DiffGitHubReleases from "./components/watches/DiffGitHubReleases";
import StateFileViewer from "./components/state/StateFileViewer";
import PreflightCheckPage from "./components/PreflightCheckPage";
import Ship from "./components/Ship";
import ShipInitPre from "./components/ShipInitPre";
import ShipUnfork from "./components/ShipUnfork";
import ShipInitCompleted from "./components/ShipInitCompleted";
import WatchDetailPage from "./components/watches/WatchDetailPage";
import ClusterScope from "./components/clusterscope/ClusterScope";
import DownstreamTree from "./components/tree/ApplicationTree";
import UnsupportedBrowser from "./components/static/UnsupportedBrowser";
import SecureAdminConsole from "./components/SecureAdminConsole";
import NotFound from "./components/static/NotFound";
import { Utilities } from "./utilities/utilities";
import { ShipClientGQL } from "./ShipClientGQL";

import { listWatches, listPendingInit, listHelmCharts } from "@src/queries/WatchQueries";
import Footer from "./components/shared/Footer";
import NavBar from "./components/shared/NavBar";

// Import Ship Init component CSS first
import "@replicatedhq/ship-init/dist/styles.css";
import "./scss/index.scss";

const INIT_SESSION_ID_STORAGE_KEY = "initSessionId";

let history = createBrowserHistory();
if(process.env.NODE_ENV === "production") {
  const piwik = new ReactPiwik({
    url: "https://data-2.replicated.com",
    siteId: 6,
    trackErrors: true,
    jsFilename: "js/",
    phpFilename: "js/"
  });
  history = piwik.connectToHistory(history);
}

/**
 * Create our GraphQL Client
 */
const GraphQLClient = ShipClientGQL(
  window.env.GRAPHQL_ENDPOINT,
  window.env.REST_ENDPOINT,
  () => Utilities.getToken()
);

class ProtectedRoute extends Component {
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

class Root extends Component {
  state = {
    listWatches: [],
    initSessionId: Utilities.localStorageEnabled()
      ? localStorage.getItem(INIT_SESSION_ID_STORAGE_KEY)
      : "",
    themeState: {
      navbarLogo: null
    },
    rootDidInitialWatchFetch: false
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

  handleUpdateCompletion = history => () => {
    history.push("/watches");
    this.handleActiveInitSessionCompleted();
  }

  refetchListWatches = async () => {
    // @TODO: Turn all of these queries into just 1 call
    // @TODO: Lean out the query to only fetch the data that
    //        We need

    // Fetch list of your watches
    const watchList = await GraphQLClient.query({
      query: listWatches,
      fetchPolicy: "no-cache"
    }).catch( error => {
      throw error;
    });

    // Fetch list of pending inits
    const pendingInits = await GraphQLClient.query({
      query: listPendingInit,
      fetchPolicy: "no-cache"
    }).catch( error => {
      throw error;
    });

    // Fetch list of pending unforks
    const pendingUnforks = await GraphQLClient.query({
      query: listHelmCharts,
      fetchPolicy: "no-cache"
    }).catch( error => {
      throw error;
    });

    const allWatches = watchList.data.listWatches.concat(
      pendingInits.data.listPendingInitSessions,
      pendingUnforks.data.listHelmCharts
    );

    this.setState({
      listWatches: allWatches,
      rootDidInitialWatchFetch: true
    });

    return allWatches;
  }

  onRootMounted = () => {
    if (Utilities.isLoggedIn()) {
      this.refetchListWatches().then(listWatches => {
        if (listWatches.length > 0 && window.location.pathname === "/watches") {
          const { slug } = listWatches[0];
          history.replace(`/watch/${slug}`);
        }
      });
    }
  }

  componentDidMount() {
    this.onRootMounted();
  }

  render() {
    const {
      initSessionId,
      themeState,
      listWatches,
      rootDidInitialWatchFetch
    } = this.state;

    return (
      <div className="flex-column flex1">
        <Helmet>
          <meta httpEquiv="Cache-Control" content="no-cache, no-store, must-revalidate" />
          <meta httpEquiv="Pragma" content="no-cache" />
          <meta httpEquiv="Expires" content="0" />
        </Helmet>
        <ApolloProvider client={GraphQLClient}>
          <ThemeContext.Provider value={{
            setThemeState: this.setThemeState,
            getThemeState: this.getThemeState,
            clearThemeState: this.clearThemeState
          }}>
            <Router history={history}>
              <div className="flex-column flex1">
                <NavBar logo={themeState.navbarLogo}/>
                <div className="flex1 flex-column u-overflow--hidden">
                  <Switch>

                    <Route exact path="/" component={() => <Redirect to={Utilities.isLoggedIn() ? "/watches" : "/login"} />} />
                    <Route exact path="/crashz" render={() => {
                      const Crashz = () => {
                        throw new Error("Crashz!");
                      };
                      return <Crashz />;

                    }}/>
                    <Route exact path="/login" render={props => (<Login {...props} onLoginSuccess={this.refetchListWatches} />) } />
                    <Route exact path="/signup" component={Signup} />
                    <Route exact path="/secure-console" render={props => <SecureAdminConsole {...props} onLoginSuccess={this.refetchListWatches} />} />
                    <Route path="/auth/github" render={props => (<GitHubAuth {...props} refetchListWatches={this.refetchListWatches}/>)} />
                    <Route path="/install/github" component={GitHubInstall} />
                    <Route path="/clusterscope" component={ClusterScope} />
                    <Route path="/unsupported" component={UnsupportedBrowser} />
                    <Route path="/preflight/:owner/:name/:downstream" component={PreflightCheckPage}/>
                    <ProtectedRoute path="/clusters" render={(props) => <Clusters {...props} />} />
                    <ProtectedRoute path="/cluster/create" render={(props) => <CreateCluster {...props} />} />
                    <ProtectedRoute
                      path="/watches"
                      render={
                        props => (
                          <WatchDetailPage
                            {...props}
                            rootDidInitialWatchFetch={rootDidInitialWatchFetch}
                            listWatches={listWatches}
                            refetchListWatches={this.refetchListWatches}
                            onActiveInitSession={this.handleActiveInitSession}
                          />
                        )
                      }
                    />
                    <ProtectedRoute path="/watch/:owner/:slug/history/compare/:org/:repo/:branch/:rootPath/:firstSeqNumber/:secondSeqNumber" component={DiffGitHubReleases} />
                    <ProtectedRoute path="/watch/:owner/:slug/history/compare/:firstSeqNumber/:secondSeqNumber" component={DiffShipReleases} />
                    <ProtectedRoute path="/watch/:owner/:slug/history" component={VersionHistory} />
                    <ProtectedRoute path="/watch/:owner/:slug/tree/:sequence" render={props => <DownstreamTree {...props} />} />
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
                          refetchListWatches={this.refetchListWatches}
                          initSessionId={initSessionId}
                          onActiveInitSessionCompleted={this.handleActiveInitSessionCompleted}
                        />
                      }
                    />
                    <ProtectedRoute
                      path="/watch/:owner/:slug/:tab?"
                      render={
                        props => (
                          <WatchDetailPage
                            {...props}
                            rootDidInitialWatchFetch={rootDidInitialWatchFetch}
                            listWatches={listWatches}
                            refetchListWatches={this.refetchListWatches}
                            onActiveInitSession={this.handleActiveInitSession}
                          />
                        )
                      }
                    />
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
