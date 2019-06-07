import React, {Suspense, lazy} from "react";
import { withRouter, Switch, Route } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import omit from "lodash/omit";
import Modal from "react-modal";

import withTheme from "@src/components/context/withTheme";
<<<<<<< HEAD
import { getWatch, listWatches } from "@src/queries/WatchQueries";
=======
import { getWatch } from "@src/queries/WatchQueries";
>>>>>>> Make logo at top change when you view a watch
import { createUpdateSession, deleteWatch } from "../../mutations/WatchMutations";
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

class WatchDetailPage extends React.Component {
  constructor() {
    super();
    this.state = {
      watch: null,
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

  componentDidUpdate(lastProps) {
    const { getThemeState, setThemeState } = this.props;
    const { getWatch } = this.props.data;
    const { watch } = this.state;
    if (getWatch !== lastProps.data.getWatch && getWatch) {
      this.setState({ watch: omit(getWatch, ["__typename"]) });
      if (getWatch.cluster) {
        this.props.history.replace(`/watch/${getWatch.slug}/state`);
      }
    }
    if (watch?.watchIcon) {
      const { navbarLogo } = getThemeState();
<<<<<<< HEAD
=======
      console.log(navbarLogo);
>>>>>>> Make logo at top change when you view a watch
      if (navbarLogo === null || navbarLogo !== watch.watchIcon) {
        setThemeState({
          navbarLogo: watch.watchIcon
        });
      }
    }
  }

  componentDidMount() {
    const { getWatch } = this.props.data;
    if (getWatch) {
      this.setState({ watch: omit(getWatch, ["__typename"]) });
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

  onDeleteDeployment = async () => {
    const { clusterToRemove } = this.state;
    await this.props.deleteWatch(clusterToRemove.id).then(() => {
      this.setState({
        clusterToRemove: {},
        selectedWatchName: "",
        displayRemoveClusterModal: false
      });
      this.props.data.refetch();
    })
  }

  render() {
    const { match } = this.props;
    const {
      watch,
      displayRemoveClusterModal,
      addNewClusterModal,
      clusterToRemove
    } = this.state;

    const slug = `${match.params.owner}/${match.params.slug}`;
    if (!watch || this.props.data.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }
    return (
      <div className="WatchDetailPage--wrapper flex-column flex1">
        <SidebarLayout
          className="flex u-minHeight--full"
          condition={this.props.listWatchesQuery?.listWatches?.length > 1}
          sidebar={(
            <SideBar
              className="flex flex1"
              watches={this.props.listWatchesQuery?.listWatches}
              currentWatch={watch.watchName}
            />
          )}
        >
          <div className="flex-column flex3 u-width--full">
            <SubNavBar
              className="flex u-marginBottom--30"
              activeTab={match.params.tab || "app"}
              slug={slug}
              watch={watch}
            />
            <Suspense fallback={<div className="flex-column flex1 alignItems--center justifyContent--center"><Loader size="60" /></div>}>
              <Switch>
                {!watch.cluster &&
                  <Route exact path="/watch/:owner/:slug" render={() =>
                    <DetailPageApplication
                      watch={watch}
                      updateCallback={this.props.data.refetch}
                    />
                  } />
                }
                {!watch.cluster &&
                  <Route exact path="/watch/:owner/:slug/deployment-clusters" render={() =>
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
<<<<<<< HEAD
  graphql(listWatches, {
    name: "listWatchesQuery",
    options: {
      fetchPolicy: "network-only"
    }
  }),
=======
>>>>>>> Make logo at top change when you view a watch
  graphql(
    getWatch, {
      options: ({ match }) => ({
        variables: { slug: `${match.params.owner}/${match.params.slug}` },
        fetchPolicy: "network-only"
      })
    }
  ),
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
