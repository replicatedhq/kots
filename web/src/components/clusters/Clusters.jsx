import * as React from "react";
import PropTypes from "prop-types";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import ContentHeader from "../shared/ContentHeader";
import ClusterCard from "./ClusterCard";
import Modal from "react-modal";
import ConfigureGitHubCluster from "../shared/ConfigureGitHubCluster";
import Loader from "../shared/Loader";
import Clipboard from "clipboard";
import { listClusters } from "../../queries/ClusterQueries";
import { updateCluster, deleteCluster } from "../../mutations/ClusterMutations";

import "../../scss/components/watches/WatchedApps.scss";
import "../../scss/components/watches/WatchCard.scss";
import "../../scss/components/clusters/CreateCluster.scss";

export class Clusters extends React.Component {
  static propTypes = {
    history: PropTypes.object.isRequired,
  };

  state = {
    clusters: [],
    clusterToManage: {},
    displayManageModal: false,
    clusterName: "",
    gitOpsRef: {},
    displayInstallModal: false,
    updatingCluster: false,
    displayDeleteClusterlModal: false,
    deletingCluster: false,
    deleteErr: ""
  }

  gatherGitHubData = (key, value) => {
    this.setState({
      ...this.state,
      gitOpsRef: {
        ...this.state.gitOpsRef,
        [`${key}`]: value
      }
    })
  }

  manageClusterClick = (cluster) => {
    this.setState({
      clusterToManage: cluster,
      clusterName: cluster.title,
      gitOpsRef:cluster.gitOpsRef ? {
        owner: cluster.gitOpsRef.owner,
        repo: cluster.gitOpsRef.repo,
        branch: cluster.gitOpsRef.branch,
      } : null,
      displayManageModal: true
    })
  }

  toggleInstallShipModal = (cluster) => {
    this.setState({
      clusterId: cluster.id,
      clusterName: cluster.title,
      clusterToken: cluster.shipOpsRef.token,
      displayInstallModal: true
    })
  }

  toggleDeleteClusterModal = (cluster) => {
    this.setState({
      displayDeleteClusterlModal: true,
      clusterToManage: cluster
    })
  }

  closeCopyToast = () => {
    this.setState({
      shipInstallSnippetCopySuccess: false,
      copyMessage: ""
    });
  }

  showCopyToast = (message) => {
    this.setState({
      shipInstallSnippetCopySuccess: true,
      copyMessage: message
    });
    setTimeout(() => {
      this.closeCopyToast();
    }, 3000);
  }

  instantiateCopyAction = () => {
    let clipboard = new Clipboard(`.copy-ship-install-snippet`);
    this.setState({
      shipInstallSnippetCopySuccess: false
    });
    clipboard.on("success", () => {
      this.showCopyToast("Copied");
    });
    clipboard.on("error", () => {
      this.showCopyToast("Unable to copy, select the text and use 'Command/Ctl + C'");
    });
  }

  onUpdateCluster = async () => {
    const { clusterToManage, clusterName, gitOpsRef } = this.state;
    this.setState({ updatingCluster: true });
    await this.props.updateCluster(clusterToManage.id, clusterName, gitOpsRef)
    .then(() => {
      this.props.listClustersQuery.refetch();
      this.setState({
        updatingCluster: false,
        displayManageModal: false,
        clusterToManage: {},
        clusterName: "",
        gitOpsRef: {}
      })
    })
    .catch()
  }

  onDeleteCluster = async () => {
    const { clusterToManage } = this.state;
    this.setState({ deletingCluster: true });
    await this.props.deleteCluster(clusterToManage.id).then(() => {
      this.props.listClustersQuery.refetch();
      this.setState({
        displayDeleteClusterlModal: false,
        clusterToManage: {},
        deletingCluster: false,
        deleteErr: ""
      });
    }).catch((err) => {
      err.graphQLErrors.map(({ message }) => {
        this.setState({
          deleteErr: message,
          deletingCluster: false
        })
      })
    });
  }

  componentDidUpdate(lastProps, lastState) {
    if (this.state.displayInstallModal !== lastState.displayInstallModal && this.state.displayInstallModal) {
      this.instantiateCopyAction();
    }
  }

  render() {
    const { clusterToManage } = this.state;
    const { listClustersQuery } = this.props;
    if (this.props.listClustersQuery.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    return (
      <div className="ClusterDashboard--wrapper container flex-column flex1 u-overflow--auto">
        <div className="flex-column flex1">
          <ContentHeader
            title="Deployed clusters"
            buttonText="Add a new cluster"
            onClick={() => this.props.history.push("/cluster/create")}
          />
          <div className="flex-column flex-1-auto u-paddingBottom--20 u-overflow--auto">
            {listClustersQuery ?
              !listClustersQuery.listClusters.length ?
                <div className="flex-column flex1 justifyContent--center alignItems--center">
                  <div className="EmptyState flex-column flex1 alignItems--center justifyContent--center">
                    <div className="u-textAlign--center u-marginTop--10">
                      <p className="u-fontSize--largest u-color--tuna u-lineHeight--medium u-fontWeight--bold u-marginBottom--10">No clusters found</p>
                      <p className="u-fontSize--more u-color--dustyGray u-lineHeight--medium u-fontWeight--medium">You haven't connected any deployment pipelines yet. Ship can deploy directly to a cluster or by pushing to a GitHub repo and using a GitOps workflow.</p>
                    </div>
                    <div className="u-marginTop--20">
                      <button className="btn primary" onClick={() => this.props.history.push("/cluster/create")}>Connect your Kubernetes cluster</button>
                    </div>
                  </div>
                </div>
              :
              <div className="u-flexTabletReflow flex-auto installed-clusters-wrapper flexWrap--wrap">
                {listClustersQuery.listClusters.map((cluster, i) => (
                  <div key={cluster.id} className="installed-cluster-wrapper flex flex1 u-paddingBottom--20">
                    <ClusterCard
                      index={i}
                      item={cluster}
                      handleManageClick={() => this.manageClusterClick(cluster)}
                      toggleInstallShipModal={() => this.toggleInstallShipModal(cluster)}
                      toggleDeleteClusterModal={() => this.toggleDeleteClusterModal(cluster)}
                    />
                  </div>
                ))}
              </div>
            : null}
          </div>
        </div>
        {this.state.displayManageModal &&
          <Modal
            isOpen={this.state.displayManageModal}
            onRequestClose={() => this.setState({ clusterToManage: {}, clusterName: "", displayManageModal: false })}
            shouldReturnFocusAfterClose={false}
            contentLabel="Manage cluster modal"
            ariaHideApp={false}
            className={`ManageClusterModal--wrapper Modal ${clusterToManage.gitOpsRef ? "MediumSize" : "DefaultSize"}`}
          >
            <div className="Modal-body">
              <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Update {clusterToManage.title} cluster</h2>
              <div className="flex flex1">
                <div className="flex-column flex1">
                  <p className="u-fontWeight--bold u-fontSize--normal u-color--tundora u-marginBottom--10 u-marginTop--10">Cluster name</p>
                  <input type="text" className="Input" placeholder="/my-path" value={this.state.clusterName} onChange={(e) => { this.setState({ clusterName: e.target.value }); }}/>
                  {clusterToManage.gitOpsRef ?
                    <div className="u-marginTop--10">
                      <ConfigureGitHubCluster
                        clusterTitle={this.state.clusterName}
                        hideRootPath={true}
                        integrationToManage={clusterToManage.gitOpsRef}
                        gatherGitHubData={this.gatherGitHubData}
                      />
                    </div>
                    : null}
                  <div className={`u-marginTop--${clusterToManage.gitOpsRef ? "20" : "10"} u-paddingTop--5 flex`}>
                    <button onClick={() => this.setState({ clusterToManage: {}, displayManageModal: false })} className="btn secondary u-marginRight--10">Cancel</button>
                    <button disabled={!this.state.clusterName.length || this.state.updatingCluster} onClick={this.onUpdateCluster} className="btn green primary">Update cluster</button>
                  </div>
                </div>
              </div>
            </div>
          </Modal>
        }
        {this.state.displayInstallModal &&
          <Modal
            isOpen={this.state.displayInstallModal}
            onRequestClose={() => this.setState({ clusterToManage: {}, clusterName: "", displayInstallModal: false })}
            shouldReturnFocusAfterClose={false}
            contentLabel="Install Ship cluster modal"
            ariaHideApp={false}
            className="InstallClusterModal--wrapper Modal DefaultSize"
          >
            <div className="Modal-body">
              <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Install {this.state.clusterName}</h2>
                <div className="flex-column flex1">
                  <p className="u-fontWeight--bold u-fontSize--normal u-color--tundora u-marginTop--10">Run this command to install your cluster</p>
                  <code className="u-lineHeight--normal u-fontSize--small u-overflow--auto">
                    kubectl apply -f {`${window.env.INSTALL_ENDPOINT}/${this.state.clusterId}/${this.state.clusterToken}`}
                  </code>
                  <div className="HiddenText-wrapper u-position--absolute">
                    <textarea value={`kubectl apply -f ${window.env.INSTALL_ENDPOINT}/${this.state.clusterId}/${this.state.clusterToken}`} id="copy-ship-install-snippet" readOnly={true}></textarea>
                  </div>
                  {Clipboard.isSupported() ?
                    <span className={`copy-ship-install-snippet u-fontSize--small u-fontWeight--medium u-marginBottom--10`} data-clipboard-target={`#copy-ship-install-snippet`}>
                      {this.state.shipInstallSnippetCopySuccess ?
                        <span className="u-color--chateauGreen u-userSelect--none u-cursor--default">{this.state.copyMessage}</span>
                        :
                        <span className="replicated-link">Copy command</span>
                      }
                    </span>
                  : null}
                  <div className="u-marginTop--10">
                    <button onClick={() => this.setState({ clusterToManage: {}, clusterName: "", displayInstallModal: false })} className="btn green primary">Ok, got it!</button>
                  </div>
                </div>
            </div>
          </Modal>
        }
        {this.state.displayDeleteClusterlModal &&
          <Modal
            isOpen={this.state.displayDeleteClusterlModal}
            onRequestClose={() => this.setState({ clusterToManage: {}, displayDeleteClusterlModal: false, deleteErr: "" })}
            shouldReturnFocusAfterClose={false}
            contentLabel="Delete cluster modal"
            ariaHideApp={false}
            className="DeleteClusterModal--wrapper Modal DefaultSize"
          >
            <div className="Modal-body">
              <h2 className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">Are you sure you want to delete {this.state.clusterToManage.title}</h2>
                <div className="flex-column flex1">
                  <p className="u-fontWeight--medium u-fontSize--normal u-color--dustyGray u-lineHeight--more u-marginTop--10">You cannot undo this action. Clusters that have applications deployed to them cannot be deleted.</p>
                  {this.state.deleteErr && this.state.deleteErr.length &&
                    <p className="u-fontWeight--medium u-fontSize--normal u-color--red u-lineHeight--more u-marginTop--10">{this.state.deleteErr}</p>
                  }
                  <div className="u-marginTop--20">
                    <button onClick={() => this.onDeleteCluster() } className="btn red primary">Delete cluster</button>
                  </div>
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
  graphql(listClusters, {
    name: "listClustersQuery",
    options: {
      fetchPolicy: "network-only"
    }
  }),
  graphql(updateCluster, {
    props: ({ mutate }) => ({
      updateCluster: (clusterId, clusterName, gitOpsRef) => mutate({ variables: { clusterId, clusterName, gitOpsRef }})
    })
  }),
  graphql(deleteCluster, {
    props: ({ mutate }) => ({
      deleteCluster: (clusterId) => mutate({ variables: { clusterId }})
    })
  }),
)(Clusters);
