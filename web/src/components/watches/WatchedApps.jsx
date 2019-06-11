import * as React from "react";
import PropTypes from "prop-types";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import ContentHeader from "../shared/ContentHeader";
import WatchCard from "./WatchCard/";
import PendingWatchCard from "./PendingWatchCard.jsx";
import Loader from "../shared/Loader";
import { listWatches, listPendingInit, userFeatures } from "../../queries/WatchQueries";
import { createEditSession, deleteWatch, deployWatchVersion } from "../../mutations/WatchMutations";
import ShipLoading from "../ShipLoading";
import Modal from "react-modal";
import AddClusterModal from "../shared/modals/AddClusterModal";
import { Utilities } from "../../utilities/utilities";

import "../../scss/components/watches/WatchedApps.scss";
import "../../scss/components/watches/WatchCard.scss";
import find from "lodash/find";

export class WatchedApps extends React.Component {
  static propTypes = {
    history: PropTypes.object.isRequired,
    onActiveInitSession: PropTypes.func.isRequired,
  };

  state = {
    displayContributorsModal: false,
    editContributorsFor: {
      id: "",
      name: ""
    },
    clusterIdToAddTo: "",
    watchToEdit: null,
    watches: [],
    pendingWatches: [],
    downloadingIds: new Set(),
    addNewClusterModal: false,
    displayRemoveClusterModal: false,
    clusterToRemove: {},
    existingDeploymentClusters: []
  }

  onEditApplicationClicked = (watch) => {
    this.setState({ watchToEdit: watch });

    this.props.client.mutate({
      mutation: createEditSession,
      variables: {
        watchId: watch.id,
      },
    })
    .then(({ data }) => {
      this.props.onActiveInitSession(data.createEditSession.id);
      this.props.history.push("/ship/edit")
    })
    .catch(() => this.setState({ watchToEdit: null }));
  }

  onPendingInstallClick = async (watch) => {
    const { id } = watch;
    this.props.history.push(`/watch/create/init?pendingInitId=${id}&start=1`);
  }

  onCardActionClick = (url) => {
    this.props.history.push(url);
  }

  onNotificationsClick = (slug) => {
    this.props.history.push(`/notifications/${slug}`);
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
    })
  }

  closeAddClusterModal = () => {
    this.setState({
      addNewClusterModal: false,
      clusterParentSlug: "",
      selectedWatchName: "",
      existingDeploymentClusters: []
    })
  }

  setWatchIdToDownload = (id) => {
    this.setState({ watchIdToDownload: id });
  }

  toggleDeleteDeploymentModal = (watch, parentName) => {
    this.setState({
      clusterToRemove: watch,
      selectedWatchName: parentName,
      displayRemoveClusterModal: !this.state.displayRemoveClusterModal
    })
  }

  makeCurrentRelease = async (watchId, sequence) => {
    await this.props.deployWatchVersion(watchId, sequence).then(() => {
      this.props.listWatchesQuery.refetch();
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
      this.props.listWatchesQuery.refetch();
    })
  }

  handleDownload = async() => {
    const { watchIdToDownload } = this.state;
    this.setState({
      downloadingIds: new Set([...this.state.downloadingIds].concat([watchIdToDownload]))
    });
    await Utilities.handleDownload(watchIdToDownload)
    this.setState({
      downloadingIds: new Set([...this.state.downloadingIds].filter(x => x !== watchIdToDownload))
    });
  }

  componentDidMount() {
    // If redirect from github
    const { location } = this.props;
    const _search = location && location.search;
    const searchParams = new URLSearchParams(_search);
    const installationId = searchParams.get("installation_id");
    if (installationId) {
      let appRedirect = document.cookie.match("(^|;)\\s*appRedirect\\s*=\\s*([^;]+)");
      if (appRedirect) {
        appRedirect = appRedirect.pop();
        this.props.history.push(`${appRedirect}?configure`);
      }
    }

  }

  componentDidUpdate(lastProps) {
    const { listWatchesQuery, pendingWatchesQuery, history, location } = this.props;


    // HACK:
    // This view is no longer being used!
    // When the watches are fetched, this condition replaces the current view
    // with the WatchDetailPage.jsx view
    if (listWatchesQuery.loading) {
      return;
    }

    if (listWatchesQuery?.listWatches?.length) {
      const [firstWatch] = listWatchesQuery.listWatches;
      history.replace(`/watch/${firstWatch.slug}`);
    } else {
      history.replace("/watch/create/init");
    }

    // Looks like this stuff isn't really being used anymore...
    const _search = location && location.search;
    const searchParams = new URLSearchParams(_search);
    const addClusterId = searchParams.get("add_cluster_id");
    if (listWatchesQuery !== lastProps.listWatchesQuery && listWatchesQuery.listWatches) {
      this.setState({ watches: listWatchesQuery.listWatches })
      if (!listWatchesQuery.loading && !listWatchesQuery.listWatches.length) {
        history.push("/watch/create/init");
      }
      if (addClusterId) {
        const watch = find(listWatchesQuery.listWatches, ["id", addClusterId]);
        this.handleAddNewClusterClick(watch);
      }
    }
    if (pendingWatchesQuery !== lastProps.pendingWatchesQuery && pendingWatchesQuery.listPendingInitSessions) {
      this.setState({ pendingWatches: pendingWatchesQuery.listPendingInitSessions });
    }
  }

  toggleContributorsModal = (watch) => {
    this.setState({
      displayContributorsModal: !this.state.displayContributorsModal,
      editContributorsFor: this.state.displayContributorsModal ? { id: "", name: "" } : watch
    });
  }

  render() {
    const { listWatchesQuery, pendingWatchesQuery } = this.props;
    const { watches, pendingWatches, watchToEdit, downloadingIds, clusterToRemove, addNewClusterModal, displayRemoveClusterModal } = this.state;
    const showLoader = (!listWatchesQuery || listWatchesQuery.loading || !pendingWatchesQuery || pendingWatchesQuery.loading);

    if (showLoader) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    if (watchToEdit) {
      return <ShipLoading headerText={`Fetching ${Utilities.toTitleCase(watchToEdit.watchName) || ""}`} subText="We're moving as fast as we can but it may take a moment." />
    }

    return (
      <div className="container flex-column flex1 u-overflow--auto">
        <div className="flex-column flex1">
          <ContentHeader
            title="Installed 3rd-party applications"
            buttonText="Install a new application"
            onClick={() => this.props.history.push("/watch/create/init")}
            searchCallback={(watches, pendingWatches) => { this.setState({ watches, pendingWatches }) }}
            showUnfork
          />
          <div className="flex1 u-paddingBottom--20 installed-watches-wrapper">
            {watches.length ? watches.map((watch) => (
              <div key={watch.id} className="installed-watch-wrapper flex flex-auto u-paddingBottom--20">
                <WatchCard
                  item={watch}
                  clustersEnabled={true}
                  handleDownload={() => this.handleDownload()}
                  setWatchIdToDownload={this.setWatchIdToDownload}
                  handleAddNewClusterClick={() => this.handleAddNewClusterClick(watch)}
                  downloadingIds={downloadingIds}
                  onEditContributorsClick={this.toggleContributorsModal}
                  onCardActionClick={this.onCardActionClick}
                  onEditApplication={this.onEditApplicationClicked}
                  toggleDeleteDeploymentModal={this.toggleDeleteDeploymentModal}
                  installLatestVersion={this.makeCurrentRelease}
                  submitCallback={() => {
                    this.props.listWatchesQuery.refetch();
                  }}
                />
              </div>
            )) : null}
            {pendingWatches.length ? pendingWatches.map((pendingWatch) => (
              <div key={pendingWatch.id} className="installed-watch-wrapper pending flex flex-auto u-paddingBottom--20">
                <PendingWatchCard
                  pendingContext="install"
                  item={pendingWatch}
                  onEditApplication={this.onPendingInstallClick}
                  submitCallback={() => {
                    this.props.pendingWatchesQuery.refetch();
                  }}
                />
              </div>
            ))
            : null}
            {!watches.length && !pendingWatches.length ?
              <div className="flex1 flex alignItems--center justifyContent--center">
                <p className="u-fontWeight--medium u-color--dustyGray">No watches found</p>
              </div>
            : null}
          </div>
        </div>
        <Modal
          isOpen={addNewClusterModal}
          onRequestClose={this.closeAddClusterModal}
          shouldReturnFocusAfterClose={false}
          contentLabel="Add cluster modal"
          ariaHideApp={false}
          className="AddNewClusterModal--wrapper Modal"
        >
          <div className="Modal-body">
            <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Add {this.state.selectedWatchName} to a deployment cluster</h2>
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Select one of your existing clusters to deploy to.</p>
            <AddClusterModal
              onAddCluster={this.addClusterToWatch}
              onRequestClose={this.closeAddClusterModal}
              existingDeploymentClusters={this.state.existingDeploymentClusters}
            />
          </div>
        </Modal>
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
              <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Remove {this.state.selectedWatchName} from your {clusterToRemove.cluster.title} cluster</h2>
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
  withRouter,
  withApollo,
  graphql(listWatches, {
    name: "listWatchesQuery",
    options: {
      fetchPolicy: "network-only"
    }
  }),
  graphql(listPendingInit, {
    name: "pendingWatchesQuery",
    options: {
      fetchPolicy: "no-cache"
    }
  }),
  graphql(userFeatures, {
    name: "userFeaturesQuery"
  }),
  graphql(deleteWatch, {
    props: ({ mutate }) => ({
      deleteWatch: (watchId) => mutate({ variables: { watchId }})
    })
  }),
  graphql(deployWatchVersion, {
    props: ({ mutate }) => ({
      deployWatchVersion: (watchId, sequence) => mutate({ variables: { watchId, sequence } })
    })
  })
)(WatchedApps);
