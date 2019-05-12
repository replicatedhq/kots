import * as React from "react";
import PropTypes from "prop-types";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import ContentHeader from "../shared/ContentHeader";
import WatchCard from "./WatchCard/";
import Loader from "../shared/Loader";
import { listWatches, userFeatures } from "../../queries/WatchQueries";
import { createUpdateSession, deleteWatch, deployWatchVersion } from "../../mutations/WatchMutations";
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
    downloadingIds: new Set(),
    addNewClusterModal: false,
    displayRemoveClusterModal: false,
    clusterToRemove: {},
    existingDeploymentClusters: []
  }

  onEditApplicationClicked = (watch) => {
    const { onActiveInitSession } = this.props;

    this.setState({ watchToEdit: watch });
    this.props.createUpdateSession(watch.id)
      .then(({ data }) => {
        const { createUpdateSession } = data;
        const { id: initSessionId } = createUpdateSession;
        onActiveInitSession(initSessionId);
        this.props.history.push("/ship/update")
      })
      .catch(() => this.setState({ watchToEdit: null }));
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

  async componentDidMount() {
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
    const { listWatchesQuery, history, location } = this.props;
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
  }

  toggleContributorsModal = (watch) => {
    this.setState({
      displayContributorsModal: !this.state.displayContributorsModal,
      editContributorsFor: this.state.displayContributorsModal ? { id: "", name: "" } : watch
    });
  }

  render() {
    const { listWatchesQuery } = this.props;
    const { watches, watchToEdit, downloadingIds, clusterToRemove, addNewClusterModal, displayRemoveClusterModal } = this.state;

    if (!listWatchesQuery || listWatchesQuery.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" color="#44bb66" />
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
            searchCallback={(watches) => { this.setState({ watches }) }}
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
            )) :
              <div className="flex1 flex alignItems--center justifyContent--center">
                <p className="u-fontWeight--medium u-color--dustyGray">No watches found</p>
              </div>
            }
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
  graphql(userFeatures, {
    name: "userFeaturesQuery"
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
  graphql(deployWatchVersion, {
    props: ({ mutate }) => ({
      deployWatchVersion: (watchId, sequence) => mutate({ variables: { watchId, sequence } })
    })
  })
)(WatchedApps);
