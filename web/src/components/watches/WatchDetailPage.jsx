import React, { Component, Fragment } from "react";
import classNames from "classnames";
import { withRouter, Switch, Route } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import Modal from "react-modal";

import withTheme from "@src/components/context/withTheme";
import { getWatch } from "@src/queries/WatchQueries";
import { createUpdateSession, deleteWatch, checkForUpdates } from "../../mutations/WatchMutations";
import WatchSidebarItem from "@src/components/watches/WatchSidebarItem";
import NotFound from "../static/NotFound";
import DetailPageApplication from "./DetailPageApplication";
import DetailPageIntegrations from "./DetailPageIntegrations";
import StateFileViewer from "../state/StateFileViewer";
import DeploymentClusters from "../watches/DeploymentClusters";
import AddClusterModal from "../shared/modals/AddClusterModal";
import WatchVersionHistory from "./WatchVersionHistory";
import WatchConfig from "./WatchConfig";
import WatchTroubleshoot from "./WatchTroubleshoot";
import WatchLicense from "./WatchLicense";
import SubNavBar from "@src/components/shared/SubNavBar";
import SidebarLayout from "../layout/SidebarLayout/SidebarLayout";
import SideBar from "../shared/SideBar";
import Loader from "../shared/Loader";

import "../../scss/components/watches/WatchDetailPage.scss";

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

  static defaultProps = {
    listWatches: [],
    getWatch: {
      loading: true
    }
  }

  componentDidUpdate(/* lastProps */) {
    const { getThemeState, setThemeState, match, listWatches } = this.props;

    const slug = `${match.params.owner}/${match.params.slug}`;
    const currentWatch = listWatches.find( w => w.slug === slug);

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

  onCheckForUpdates = () => {
    const { client, getWatchQuery } = this.props;
    const { getWatch: watch } = getWatchQuery;
    client.mutate({
      mutation: checkForUpdates,
      variables: {
        watchId: watch.id,
      }
    });
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

  render() {
    const { match, history, getWatchQuery, listWatches, refetchListWatches } = this.props;
    const {
      displayRemoveClusterModal,
      addNewClusterModal,
      clusterToRemove
    } = this.state;
    const centeredLoader = (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );

    const { getWatch: watch, loading } = getWatchQuery;

    if (history.location.pathname == "/watches") {
      if (listWatches[0]) {
        history.replace(`/watch/${listWatches[0].slug}`);
      }
      return centeredLoader;
    }

    return (
      <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
        <SidebarLayout
          className="flex u-minHeight--full u-overflow--hidden"
          condition={listWatches.length > 1}
          sidebar={(
            <SideBar
              className="flex flex1"
              items={listWatches.map( (item, idx) => (
                <WatchSidebarItem
                  key={idx}
                  className={classNames({ selected: item.slug === watch?.slug})}
                  watch={item} />
              ))}
              currentWatch={watch?.watchName}
            />
          )}>
          <div className="flex-column flex3 u-width--full u-height--full">
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
                    {watch && !watch.cluster &&
                      <Route exact path="/watch/:owner/:slug" render={() =>
                        <DetailPageApplication
                          watch={watch}
                          refetchListWatches={refetchListWatches}
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
  graphql(getWatch, {
    name: "getWatchQuery",
    options: props => {
      const { owner, slug } = props.match.params;
      return {
        variables: {
          slug: `${owner}/${slug}`
        }
      }
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
  }),
)(WatchDetailPage);
