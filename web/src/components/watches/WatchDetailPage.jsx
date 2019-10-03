import React, { Component, Fragment } from "react";
import classNames from "classnames";
import { withRouter, Switch, Route } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import Modal from "react-modal";

import withTheme from "@src/components/context/withTheme";
import { getWatch, getHelmChart } from "@src/queries/WatchQueries";
import { createUpdateSession, deleteWatch, checkForUpdates, deployWatchVersion } from "../../mutations/WatchMutations";
import WatchSidebarItem from "@src/components/watches/WatchSidebarItem";
import { KotsSidebarItem } from "@src/components/watches/WatchSidebarItem";
import { HelmChartSidebarItem } from "@src/components/watches/WatchSidebarItem";
import NotFound from "../static/NotFound";
import PendingHelmChartDetailPage from "./PendingHelmChartDetailPage";
import DetailPageApplication from "./DetailPageApplication";
import DetailPageIntegrations from "./DetailPageIntegrations";
import StateFileViewer from "../state/StateFileViewer";
import DeploymentClusters from "../watches/DeploymentClusters";
import AddClusterModal from "../shared/modals/AddClusterModal";
import WatchVersionHistory from "./WatchVersionHistory";
import DownstreamWatchVersionHistory from "./DownstreamWatchVersionHistory";
import WatchConfig from "./WatchConfig";
import WatchLicense from "./WatchLicense";
import SubNavBar from "@src/components/shared/SubNavBar";
import SidebarLayout from "../layout/SidebarLayout/SidebarLayout";
import SideBar from "../shared/SideBar";
import Loader from "../shared/Loader";
import SupportBundleList from "../troubleshoot/SupportBundleList";
import SupportBundleAnalysis from "../troubleshoot/SupportBundleAnalysis";
import GenerateSupportBundle from "../troubleshoot/GenerateSupportBundle";

import "../../scss/components/watches/WatchDetailPage.scss";

let loadingTextTimer = null;
class WatchDetailPage extends Component {
  constructor(props) {
    super(props);
    this.state = {
      displayRemoveClusterModal: false,
      addNewClusterModal: false,
      preparingUpdate: "",
      clusterParentSlug: "",
      selectedWatchName: "",
      clusterToRemove: {},
      watchToEdit: {},
      existingDeploymentClusters: [],
      checkingForUpdates: false,
      checkingUpdateText: "Checking for updates",
      updateError: false,
    }
  }

  static defaultProps = {
    getWatchQuery: {
      loading: true
    }
  }

  componentDidUpdate(lastProps) {
    const { getThemeState, setThemeState, match, listApps, history, getWatchQuery } = this.props;
    const { search } = this.props.location;
    const slug = `${match.params.owner}/${match.params.slug}`;
    const currentWatch = listApps?.find(w => w.slug === slug);

    // Handle updating the app theme state when a watch changes.
    if (currentWatch?.watchIcon) {
      const { navbarLogo, ...rest } = getThemeState();
      if (navbarLogo === null || navbarLogo !== currentWatch.watchIcon) {

        setThemeState({
          ...rest,
          navbarLogo: currentWatch.watchIcon
        });
      }
    }

    if (getWatchQuery.getWatch && getWatchQuery.getWatch !== lastProps.getWatchQuery.getWatch) {
      const URLParams = new URLSearchParams(search);
      if (URLParams.get("add")) {
        this.handleAddNewClusterClick(getWatchQuery.getWatch);
        history.replace(this.props.location.pathname); // remove query param so refreshing the page doesn't trigger the modal again.
      }
    }

    // Used for a fresh reload
    if (history.location.pathname === "/watches") {
      this.checkForFirstWatch();
    }

  }

  componentWillUnmount() {
    clearInterval(this.interval);
    this.props.clearThemeState();
  }

  makeCurrentRelease = async (watchId, sequence) => {
    await this.props.deployWatchVersion(watchId, sequence).then(() => {
      this.props.getWatchQuery.refetch();
    })
  }

  onCheckForUpdates = async () => {
    const { client, getWatchQuery } = this.props;
    const { getWatch: watch } = getWatchQuery;
    this.setState({ checkingForUpdates: true });
    loadingTextTimer = setTimeout(() => {
      this.setState({ checkingUpdateText: "Almost there, hold tight..." });
    }, 10000);
    await client.mutate({
      mutation: checkForUpdates,
      variables: {
        watchId: watch.id,
      }
    }).catch(() => {
      this.setState({ updateError: true });
    }).finally(() => {
      this.props.getWatchQuery.refetch();
      clearTimeout(loadingTextTimer);
      this.setState({
        checkingForUpdates: false,
        checkingUpdateText: "Checking for updates"
      });
    });
  }
  addClusterToWatch = (clusterId, githubPath) => {
    const { clusterParentSlug } = this.state;
    const upstreamUrl = `ship://ship-cloud/${clusterParentSlug}`;
    this.props.history.push(`/watch/create/init?upstream=${upstreamUrl}&cluster_id=${clusterId}&path=${githubPath}`);
  }

  createDownstreamForCluster = () => {
    const { clusterParentSlug } = this.state;
    localStorage.setItem("clusterRedirect", `/watch/${clusterParentSlug}/downstreams?add=1`);
    this.props.history.push("/cluster/create");
  }

  handleAddNewClusterClick = (watch) => {
    this.setState({
      addNewClusterModal: true,
      clusterParentSlug: watch.slug,
      selectedWatchName: watch.watchName,
      existingDeploymentClusters: watch.watches.map((watch) => watch.cluster.id)
    });
  }

  closeAddClusterModal = () => {
    this.setState({
      addNewClusterModal: false,
      clusterParentSlug: "",
      selectedWatchName: "",
      existingDeploymentClusters: []
    })
  }

  onEditApplicationClicked = (watch) => {
    const { onActiveInitSession } = this.props;

    this.setState({ watchToEdit: watch, preparingUpdate: watch.cluster.id });
    this.props.createUpdateSession(watch.id)
      .then(({ data }) => {
        const { createUpdateSession } = data;
        const { id: initSessionId } = createUpdateSession;
        onActiveInitSession(initSessionId);
        this.props.history.push("/ship/update")
      })
      .catch(() => this.setState({ watchToEdit: null, preparingUpdate: "" }));
  }

  toggleDeleteDeploymentModal = (watch, parentName) => {
    this.setState({
      clusterToRemove: watch,
      selectedWatchName: parentName,
      displayRemoveClusterModal: !this.state.displayRemoveClusterModal
    })
  }

  /**
   * Refetch all the GraphQL data for this component and all its children
   *
   * @return {undefined}
   */
  refetchGraphQLData = () => {
    this.props.getWatchQuery.refetch()
  }

  onDeleteDeployment = async () => {
    const { clusterToRemove } = this.state;
    await this.props.deleteWatch(clusterToRemove.id).then(() => {
      this.setState({
        clusterToRemove: {},
        selectedWatchName: "",
        displayRemoveClusterModal: false
      });
      this.refetchGraphQLData();
    })
  }

  /**
   *  Runs on mount and on update. Also handles redirect logic
   *  if no watches are found, or the first watch is found.
   */
  checkForFirstWatch = () => {
    const { history, rootDidInitialWatchFetch, listApps } = this.props;
    if (!rootDidInitialWatchFetch) {
      return;
    }

    if (listApps.length > 0) {
      history.replace(`/${listApps[0].name ? "app" : "watch"}/${listApps[0].slug}`);
    } else {
      const nextUrl = window.env.NO_APPS_REDIRECT;
      history.replace(nextUrl);
    }
  }

  componentDidMount() {
    const { history } = this.props;

    if (history.location.pathname === "/watches") {
      return this.checkForFirstWatch();
    }

  }

  render() {
    const {
      match,
      getWatchQuery,
      getHelmChartQuery,
      listApps,
      refetchListApps,
      rootDidInitialWatchFetch
    } = this.props;

    const {
      displayRemoveClusterModal,
      addNewClusterModal,
      clusterToRemove,
      checkingUpdateText,
      updateError
    } = this.state;

    const centeredLoader = (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );
    const isHelmChartUrl = match.params.owner === "helm";

    let watch;
    if (!isHelmChartUrl) {
      watch = getWatchQuery?.getWatch;
    } else {
      watch = getHelmChartQuery?.getHelmChart;
    }

    let loading;
    if (isHelmChartUrl) {
      loading = getHelmChartQuery?.loading || !rootDidInitialWatchFetch;
    } else {
      loading = getWatchQuery?.loading || !rootDidInitialWatchFetch;
    }

    if (!rootDidInitialWatchFetch) {
      return centeredLoader;
    }

    return (
      <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
        <SidebarLayout
          className="flex flex1 u-minHeight--full u-overflow--hidden"
          condition={listApps?.length > 1}
          sidebar={(
            <SideBar
              items={listApps?.map((item, idx) => {
                let sidebarItemNode;
                if (item.name) {
                  const slugFromRoute = `${match.params.owner}/${match.params.slug}`;
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
                } else if (item.slug) {
                  const slugFromRoute = `${match.params.owner}/${match.params.slug}`;
                  sidebarItemNode = (
                    <WatchSidebarItem
                      key={idx}
                      className={classNames({
                        selected: (
                          item.slug === slugFromRoute &&
                          match.params.owner !== "helm"
                        )
                      })}
                      watch={item} />
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
          <div className="flex-column flex1 u-width--full u-height--full">
            {loading
              ? centeredLoader
              : (
                <Fragment>
                  <SubNavBar
                    className="flex"
                    activeTab={match.params.tab || "app"}
                    watch={watch}
                  />
                  <Switch>
                    <Route
                      exact
                      path="/watch/helm/:id"
                      render={() =>
                        <PendingHelmChartDetailPage
                          loading={loading}
                          chart={watch}
                          refetchListApps={this.props.refetchListApps}
                          onActiveInitSession={this.props.onActiveInitSession}
                        />
                      }
                    />
                    {watch && !watch.cluster &&
                      <Route exact path="/watch/:owner/:slug" render={() =>
                        <DetailPageApplication
                          watch={watch}
                          refetchListApps={refetchListApps}
                          refetchWatch={this.props.getWatchQuery?.refetch}
                          updateCallback={this.refetchGraphQLData}
                          onActiveInitSession={this.props.onActiveInitSession}
                        />}
                      />
                    }
                    {watch && !watch.cluster &&
                      <Route exact path="/watch/:owner/:slug/downstreams" render={() =>
                        <div className="container">
                          <DeploymentClusters
                            appDetailPage={true}
                            parentWatch={watch}
                            parentClusterName={watch.watchName}
                            preparingUpdate={this.state.preparingUpdate}
                            childWatches={watch.watches}
                            handleAddNewCluster={() => this.handleAddNewClusterClick(watch)}
                            onEditApplication={this.onEditApplicationClicked}
                            installLatestVersion={this.makeCurrentRelease}
                            toggleDeleteDeploymentModal={this.toggleDeleteDeploymentModal}
                          />
                        </div>
                      } />
                    }
                    { /*
                      <Route exact path="/watch/helm/:id" render={() =>
                        <DetailPageHelmChart chart={watch} refetchListApps={refetchListApps} updateCallback={this.refetchGraphQLData} />
                      } />
                    */ }
                    { /* ROUTE UNUSED */}
                    <Route exact path="/watch/:owner/:slug/integrations" render={() => <DetailPageIntegrations watch={watch} />} />

                    <Route exact path="/watch/:owner/:slug/state" render={() => <StateFileViewer watch={watch} headerText="Edit your application’s state.json file" />} />

                    <Route exact path="/watch/:owner/:slug/version-history" render={() =>
                      <WatchVersionHistory
                        watch={watch}
                        match={this.props.match}
                        onCheckForUpdates={this.onCheckForUpdates}
                        checkingForUpdates={this.state.checkingForUpdates}
                        checkingUpdateText={checkingUpdateText}
                        handleAddNewCluster={() => this.handleAddNewClusterClick(watch)}
                        errorCheckingUpdate={updateError}
                      />
                    } />
                    <Route exact path="/watch/:owner/:slug/downstreams/:downstreamOwner/:downstreamSlug/version-history" render={() =>
                      <DownstreamWatchVersionHistory
                        watch={watch}
                        makeCurrentVersion={this.makeCurrentRelease}
                      />
                    } />
                    <Route exact path="/watch/:owner/:slug/config" render={() =>
                      <WatchConfig
                        watch={watch}
                        onActiveInitSession={this.props.onActiveInitSession}
                      />
                    } />
                    <Route exact path="/watch/:owner/:slug/troubleshoot" render={() =>
                      <SupportBundleList
                        watch={watch}
                      />
                    } />
                    <Route exact path="/watch/:owner/:slug/troubleshoot/generate" render={() =>
                      <GenerateSupportBundle
                        watch={watch}
                      />
                    } />
                    <Route path="/watch/:owner/:slug/troubleshoot/analyze/:bundleSlug" render={() =>
                      <SupportBundleAnalysis
                        watch={watch}
                      />
                    } />
                    <Route exact path="/watch/:owner/:slug/license" render={() =>
                      <WatchLicense
                        watch={watch}
                      />
                    } />
                    <Route component={NotFound} />
                  </Switch>
                </Fragment>
              )
            }
          </div>
        </SidebarLayout>
        {addNewClusterModal &&
          <Modal
            isOpen={addNewClusterModal}
            onRequestClose={this.closeAddClusterModal}
            shouldReturnFocusAfterClose={false}
            contentLabel="Add cluster modal"
            ariaHideApp={false}
            className="AddNewClusterModal--wrapper Modal"
          >
            <div className="Modal-body">
              <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Add {this.state.selectedWatchName} to a new downstream</h2>
              <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Select one of your existing downstreams to deploy to.</p>
              <AddClusterModal
                onAddCluster={this.addClusterToWatch}
                watch={watch}
                onRequestClose={this.closeAddClusterModal}
                createDownstreamForCluster={this.createDownstreamForCluster}
                existingDeploymentClusters={this.state.existingDeploymentClusters}
              />
            </div>
          </Modal>
        }
        {displayRemoveClusterModal &&
          <Modal
            isOpen={displayRemoveClusterModal}
            onRequestClose={() => this.toggleDeleteDeploymentModal({}, "")}
            shouldReturnFocusAfterClose={false}
            contentLabel="Add cluster modal"
            ariaHideApp={false}
            className="RemoveClusterFromWatchModal--wrapper Modal"
          >
            <div className="Modal-body">
              <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Remove {this.state.selectedWatchName} from {clusterToRemove.cluster.title}</h2>
              <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">This application will no longer be deployed to {clusterToRemove.cluster.title}.</p>
              <div className="u-marginTop--10 flex">
                <button onClick={() => this.toggleDeleteDeploymentModal({}, "")} className="btn secondary u-marginRight--10">Cancel</button>
                <button onClick={this.onDeleteDeployment} className="btn green primary">Delete deployment</button>
              </div>
            </div>
          </Modal>
        }
      </div>
    );
  }
}

export { WatchDetailPage };
export default compose(
  withApollo,
  withRouter,
  withTheme,
  graphql(getWatch, {
    name: "getWatchQuery",
    skip: props => {
      const { owner, slug } = props.match.params;

      // Skip this query if it's a helm chart
      if (owner === "helm") {
        return true;
      }

      // Skip if no variables (user at "/watches" URL)
      if (!owner && !slug) {
        return true;
      }

      return false;

    },
    options: props => {
      const { owner, slug } = props.match.params;
      return {
        fetchPolicy: "no-cache",
        variables: {
          slug: `${owner}/${slug}`
        }
      }
    }
  }),
  graphql(getHelmChart, {
    name: "getHelmChartQuery",
    skip: props => {
      const { owner } = props.match.params;
      return owner !== "helm";
    },
    options: props => {
      const { slug: id } = props.match.params;
      return {
        variables: {
          id
        }
      };
    }
  }),
  graphql(createUpdateSession, {
    props: ({ mutate }) => ({
      createUpdateSession: (watchId) => mutate({ variables: { watchId } })
    })
  }),
  graphql(deleteWatch, {
    props: ({ mutate }) => ({
      deleteWatch: (watchId) => mutate({ variables: { watchId } })
    })
  }),
  graphql(deployWatchVersion, {
    props: ({ mutate }) => ({
      deployWatchVersion: (watchId, sequence) => mutate({ variables: { watchId, sequence } })
    })
  })
)(WatchDetailPage);
