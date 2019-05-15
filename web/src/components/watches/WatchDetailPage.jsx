import * as React from "react";
import { withRouter, Switch, Route, Link } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import { getWatch } from "../../queries/WatchQueries";
import { omit } from "lodash";
import Loader from "../shared/Loader";
import NotFound from "../static/NotFound";
import "../../scss/components/watches/WatchDetailPage.scss";
import DetailPageApplication from "./DetailPageApplication";
import DetailPageIntegrations from "./DetailPageIntegrations";
import StateFileViewer from "../state/StateFileViewer";
import DeploymentClusters from "../watches/DeploymentClusters";
import { createUpdateSession, deleteWatch } from "../../mutations/WatchMutations";
import Modal from "react-modal";
import AddClusterModal from "../shared/modals/AddClusterModal";

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
    const { getWatch } = this.props.data;
    if (getWatch !== lastProps.data.getWatch && getWatch) {
      this.setState({ watch: omit(getWatch, ["__typename"]) });
      if (getWatch.cluster) {
        this.props.history.replace(`/watch/${getWatch.slug}/state`);
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
    const { watch, displayRemoveClusterModal, addNewClusterModal, clusterToRemove } = this.state;
    const { match } = this.props;
    const slug = `${match.params.owner}/${match.params.slug}`;

    if (!watch || this.props.data.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" color="#44bb66" />
        </div>
      )
    }

    return (
      <div className="WatchDetailPage--wrapper flex-column flex1">
        <div className="watch-detail-header flex-column alignItems--center justifyContent--center">
          <span className="watch-icon" style={{ backgroundImage: `url(${watch.watchIcon})` }}></span>
          <p className="u-color--tundora u-fontWeight--bold u-marginTop--10">{watch.watchName}</p>
        </div>
        <div className="details-subnav flex flex u-marginBottom--30">
          <ul>
            {!watch.cluster && <li className={`${!match.params.tab ? "is-active" : ""}`}><Link to={`/watch/${slug}`}>Application</Link></li>}
            {!watch.cluster && <li className={`${match.params.tab === "deployment-clusters" ? "is-active" : ""}`}><Link to={`/watch/${slug}/deployment-clusters`}>Deployment clusters</Link></li>}
            <li className={`${match.params.tab === "integrations" ? "is-active" : ""}`}><Link to={`/watch/${slug}/integrations`}>Integrations</Link></li>
            <li className={`${match.params.tab === "state" ? "is-active" : ""}`}><Link to={`/watch/${slug}/state`}>State JSON</Link></li>
          </ul>
        </div>
        <Switch>
          {!watch.cluster &&
            <Route exact path="/watch/:owner/:slug" render={() =>
              <DetailPageApplication
                watch={watch}
                updateCallback={() => {
                  this.props.data.refetch();
                }}
              />
            }/>
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
            }/>
          }
          <Route exact path="/watch/:owner/:slug/integrations" render={() => <DetailPageIntegrations watch={watch} /> } />
          <Route exact path="/watch/:owner/:slug/state" render={() =>  <StateFileViewer headerText="Edit your application’s state.json file" /> } />
          <Route component={NotFound} />
        </Switch>
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
              <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Add {this.state.selectedWatchName} to a deployment cluster</h2>
              <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Select one of your existing clusters to deploy to.</p>
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
  withApollo,
  withRouter,
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
