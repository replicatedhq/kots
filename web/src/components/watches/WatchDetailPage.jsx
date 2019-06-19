import React, { Component, Suspense, lazy } from "react";
import classNames from "classnames";
import { withRouter, Switch, Route, Redirect } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import Modal from "react-modal";

import withTheme from "@src/components/context/withTheme";
import { listWatches } from "@src/queries/WatchQueries";
import { createUpdateSession, deleteWatch } from "../../mutations/WatchMutations";
import WatchSidebarItem from "@src/components/watches/WatchSidebarItem";
import SubNavBar from "@src/components/shared/SubNavBar";
import SidebarLayout from "../layout/SidebarLayout/SidebarLayout";
import SideBar from "../shared/SideBar";
import Loader from "../shared/Loader";

import "../../scss/components/watches/WatchDetailPage.scss";

const NotFound = lazy(() => import("../static/NotFound"));
const DetailPageApplication = lazy(() => import("./DetailPageApplication"));
const DetailPageIntegrations = lazy(() => import("./DetailPageIntegrations"));
const StateFileViewer = lazy(() => import("../state/StateFileViewer"));
const DeploymentClusters = lazy(() => import("../watches/DeploymentClusters"));
const AddClusterModal = lazy(() => import("../shared/modals/AddClusterModal"));
const WatchVersionHistory = lazy (() => import("./WatchVersionHistory"));
const WatchConfig = lazy ( () => import("./WatchConfig"));
const WatchTroubleshoot = lazy(() => import("./WatchTroubleshoot"));
const WatchLicense = lazy(() => import("./WatchLicense"));

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
      existingDeploymentClusters: []
    }
  }

  componentDidUpdate(/* lastProps */) {
    const { getThemeState, setThemeState, match, listWatchesQuery } = this.props;
    const slug = `${match.params.owner}/${match.params.slug}`;

    const currentWatch = listWatchesQuery?.listWatches?.find( w => w.slug === slug);

    // Handle updating the navbar logo when a watch changes.
    if (currentWatch?.watchIcon) {
      const { navbarLogo } = getThemeState();
      if (navbarLogo === null || navbarLogo !== currentWatch.watchIcon) {
        setThemeState({
          navbarLogo: currentWatch.watchIcon
        });
      }
    }
  }

  componentWillUnmount() {
    clearInterval(this.interval);
    this.props.clearThemeState();
  }

  addClusterToWatch = (clusterId, githubPath) => {
    const { clusterParentSlug } = this.state;
    const upstreamUrl = `ship://ship-cloud/${clusterParentSlug}`;
    this.props.history.push(`/watch/create/init?upstream=${upstreamUrl}&cluster_id=${clusterId}&path=${githubPath}&start=1`);
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
    this.props.listWatchesQuery.refetch()
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

  render() {
    const { match, history, listWatchesQuery } = this.props;
    const {
      displayRemoveClusterModal,
      addNewClusterModal,
      clusterToRemove
    } = this.state;
    if (history.location.pathname == "/watches") {
      if (this.props.listWatchesQuery.loading) {
        return (
          <div className="flex-column flex1 alignItems--center justifyContent--center">
            <Loader size="60" />
          </div>
        );

      // Redirect user to ship-init if they have zero watches
      // (New user)
      } else if (this.props.listWatchesQuery.listWatches.length === 0) {
        return (
         <Redirect to="/watch/create/init" />
        );
      // Render everything normally with the data existing n' stuff
      } else {
      const { slug } = this.props.listWatchesQuery.listWatches[0];
      return (
        <Redirect to={`/watch/${slug}`} />
      );
      }
    }

    const slug = `${match.params.owner}/${match.params.slug}`;
    let watch = listWatchesQuery?.listWatches?.find( w => w.slug === slug );

    // Requested watch is a child watch / downstream cluster watch
    if (!watch && !this.props.listWatchesQuery.loading) {

      // NOTE:
      // This flat array could get really really big should someone have 10 clusters
      // and 25 bajillion children watches. In our current state, I don't feel we need
      // to scale to this as most of our users just have a couple of watches and clusters.
      // This is a non-existant scaling issue, and I think we'll be OK with just a big
      // happy flat array.
      const childWatches = this.props.listWatchesQuery.listWatches
        .map( mainWatch => mainWatch.watches)
        .flat();

      watch = childWatches.find(w => w.slug === slug);
      if (!watch) {
        return (
          <div className="flex-column flex1 alignItems--center justifyContent--center">
            Sorry! We ran into an issue finding the right Watch. Please refresh your browser :(
          </div>
        );
      }
    }

    return (
      <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
        <SidebarLayout
          className="flex u-minHeight--full u-overflow--auto"
          sidebar={(
            <SideBar
              className="flex flex1"
              aggressive={true}
              loading={this.props.listWatchesQuery.loading}
              items={this.props.listWatchesQuery?.listWatches?.map( (item, idx) => (
                <WatchSidebarItem
                  key={idx}
                  className={classNames({ selected: item.slug === watch.slug})}
                  watch={item} />
              ))}
              currentWatch={watch?.watchName}
            />
          )}>
          <div className="flex-column flex3 u-width--full u-height--full">
            <SubNavBar
              className="flex u-marginBottom--30"
              activeTab={match.params.tab || "app"}
              slug={slug}
              watch={watch}
            />
            <Suspense fallback={(
              <div className="flex-column flex1 alignItems--center justifyContent--center">
                <Loader size="60" />
              </div>
            )}>
              {
                this.props.listWatchesQuery.loading ? (
                  <div className="flex-column flex1 alignItems--center justifyContent--center">
                    <Loader size="60" />
                  </div>
                ) :
                (
                  <Switch>
                    {watch && !watch.cluster &&
                      <Route exact path="/watch/:owner/:slug" render={() =>
                        <DetailPageApplication
                          watch={watch}
                          updateCallback={this.refetchGraphQLData}
                          onActiveInitSession={this.props.onActiveInitSession}
                        />
                      } />
                    }
                    {watch && !watch.cluster &&
                      <Route exact path="/watch/:owner/:slug/downstreams" render={() =>
                        <div className="container">
                          <DeploymentClusters
                            appDetailPage={true}
                            parentClusterName={watch.watchName}
                            preparingUpdate={this.state.preparingUpdate}
                            childWatches={watch.watches}
                            handleAddNewCluster={() => this.handleAddNewClusterClick(watch)}
                            onEditApplication={this.onEditApplicationClicked}
                            toggleDeleteDeploymentModal={this.toggleDeleteDeploymentModal}
                          />
                        </div>
                      } />
                    }
                    { /* ROUTE UNUSED */}
                    <Route exact path="/watch/:owner/:slug/integrations" render={() => <DetailPageIntegrations watch={watch} />} />
                    { /* ROUTE UNUSED */}
                    <Route exact path="/watch/:owner/:slug/state" render={() => <StateFileViewer headerText="Edit your application’s state.json file" />} />

                    <Route exact path="/watch/:owner/:slug/version-history" render={() =>
                      <WatchVersionHistory
                        watch={watch}
                      />
                    } />
                    <Route exact path="/watch/:owner/:slug/config" render={() =>
                      <WatchConfig
                        watch={watch}
                      />
                    } />
                    <Route exact path="/watch/:owner/:slug/troubleshoot" render={() =>
                      <WatchTroubleshoot
                        watch={watch}
                      />
                    } />
                    <Route exact path="/watch/:owner/:slug/license" render={() =>
                      <WatchLicense
                        watch={watch}
                      />
                    } />
                    {!this.props.listWatchesQuery.loading && <Route component={NotFound} />}
                  </Switch>
                )
              }
            </Suspense>
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
                onRequestClose={this.closeAddClusterModal}
                existingDeploymentClusters={this.state.existingDeploymentClusters}
              />
            </div>
          </Modal>
        }
        {displayRemoveClusterModal &&
          <Modal
            isOpen={displayRemoveClusterModal}
            onRequestClose={() => this.toggleDeleteDeploymentModal({},"")}
            shouldReturnFocusAfterClose={false}
            contentLabel="Add cluster modal"
            ariaHideApp={false}
            className="RemoveClusterFromWatchModal--wrapper Modal"
          >
            <div className="Modal-body">
              <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Remove {this.state.selectedWatchName} from {clusterToRemove.cluster.title}</h2>
              <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">This application will no longer be deployed to {clusterToRemove.cluster.title}.</p>
              <div className="u-marginTop--10 flex">
                <button onClick={() => this.toggleDeleteDeploymentModal({},"")} className="btn secondary u-marginRight--10">Cancel</button>
                <button onClick={this.onDeleteDeployment} className="btn green primary">Delete deployment</button>
              </div>
            </div>
          </Modal>
        }
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  withTheme,
  graphql(listWatches, {
    name: "listWatchesQuery",
    options: {
      fetchPolicy: "cache-and-network"
    }
  }),
  graphql(createUpdateSession, {
    props: ({ mutate }) => ({
      createUpdateSession: (watchId) => mutate({ variables: { watchId }})
    })
  }),
  graphql(deleteWatch, {
    props: ({ mutate }) => ({
      deleteWatch: (watchId) => mutate({ variables: { watchId }})
    })
  })

)(WatchDetailPage);
