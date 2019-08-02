import * as React from "react";
import Helmet from "react-helmet";
import { withRouter } from "react-router-dom";
import { compose, withApollo } from "react-apollo";

import Select from "react-select";
import Clipboard from "clipboard";
import Modal from "react-modal";

import AddClusterModal from "../shared/modals/AddClusterModal";
import GenerateSupportBundleModal from "../troubleshoot/GenerateSupportBundleModal";

import "../../scss/components/troubleshoot/GenerateSupportBundle.scss";

const NEW_CLUSTER = "Create a new downstream cluster";

class GenerateSupportBundle extends React.Component {

  state = {
    clusters: [],
    selectedCluster: this.props.watch.watches.length ? this.props.watch.watches[0].cluster : "",
    showToast: false,
    copySuccess: false,
    copyMessage: "",
    addNewClusterModal: false,
    displayUploadModal: false
  }

  componentDidMount() {
    const { watch } = this.props;
    if (watch) {
      const watchClusters = watch.watches.map((watch) => watch.cluster);
      const NEW_ADDED_CLUSTER = { title: NEW_CLUSTER };
      this.setState({ clusters: [NEW_ADDED_CLUSTER, ...watchClusters] });
    }
    this.instantiateCopyAction();
  }

  componentDidUpdate(lastProps) {
    const { watch } = this.props;
    if (watch !== lastProps.watch && watch.watches) {
      const watchClusters = watch.watches.map((watch) => watch.cluster);
      const NEW_ADDED_CLUSTER = { title: NEW_CLUSTER };
      this.setState({ clusters: [NEW_ADDED_CLUSTER, ...watchClusters] });
    }
  }

  redirectToCreateCluster = () => {
    localStorage.setItem("clusterRedirect", `automaticDeploy-${this.props.watch.slug}`);
    this.props.history.push("/cluster/create");
  }

  handleClusterChange = (selectedCluster) => {
    if (selectedCluster.title === NEW_CLUSTER) {
      return this.redirectToCreateCluster();
    }
    this.setState({ selectedCluster });
  }

  showCopyToast(message, didCopy) {
    this.setState({
      showToast: didCopy,
      copySuccess: didCopy,
      copyMessage: message
    });
    setTimeout(() => {
      this.setState({
        showToast: false,
        copySuccess: false,
        copyMessage: ""
      });
    }, 3000);
  }

  instantiateCopyAction() {
    let clipboard = new Clipboard(".copy-command");
    clipboard.on("success", () => {
      this.showCopyToast("Command has been copied to your clipboard", true);
    });
    clipboard.on("error", () => {
      this.showCopyToast("Unable to copy, select the text and use 'Command/Ctl + C'", false);
    });
  }

  renderIcons = (shipOpsRef, gitOpsRef) => {
    if (shipOpsRef) {
      return <span className="icon clusterType ship"></span>
    } else if (gitOpsRef) {
      return <span className="icon clusterType git"></span>
    } else {
      return;
    }
  }

  getLabel = ({ shipOpsRef, gitOpsRef, title }) => {
    return (
      <div style={{ alignItems: "center", display: "flex" }}>
        <span style={{ fontSize: 18, marginRight: "0.5em" }}>{this.renderIcons(shipOpsRef, gitOpsRef)}</span>
        <span style={{ fontSize: 14 }}>{title}</span>
      </div>
    );
  }

  openClusterModal = () => {
    this.setState({ addNewClusterModal: true });
  }

  addClusterToWatch = (clusterId, githubPath) => {
    const { watch } = this.props;
    const upstreamUrl = `ship://ship-cloud/${watch.slug}`;
    this.props.history.push(`/watch/create/init?upstream=${upstreamUrl}&cluster_id=${clusterId}&path=${githubPath}`);
  }

  closeAddClusterModal = () => {
    this.setState({ addNewClusterModal: false });
  }

  createDownstreamForCluster = () => {
    const { watch } = this.props;
    localStorage.setItem("clusterRedirect", `/watch/${watch.slug}/downstreams?add=1`);
    this.props.history.push("/cluster/create");
  }

  toggleModal = () => {
    this.setState({
      displayUploadModal: !this.state.displayUploadModal
    })
  }

  render() {
    const { clusters, selectedCluster, showToast, copySuccess, copyMessage, addNewClusterModal, displayUploadModal } = this.state;
    const { watch } = this.props;
    const selectedWatch = watch ?.watches.find((watch) => watch.cluster.id === selectedCluster.id);

    return (
      <div className="GenerateSupportBundle--wrapper container flex-column u-overflow--auto u-paddingTop--30 u-paddingBottom--20 alignItems--center">
        <Helmet>
          <title>{`${watch.watchName} Troubleshoot`}</title>
        </Helmet>
        <div className="GenerateSupportBundle">
          <div>
            <h2 className="u-fontSize--larger u-fontWeight--bold u-color--tuna">Analyze your application for support</h2>
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--medium u-marginTop--5">If you’re having issues with your application, you can run analysis on your cluster to receive insights that can be shared with your vendor for support.</p>
          </div>
          {watch ?.watches.length ?
            <div className="flex1 flex-column u-paddingRight--30">
              <div className="u-marginTop--40">
                <h2 className="u-fontSize--larger u-fontWeight--bold u-color--tuna">Which cluster do you need support with?</h2>
                <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--medium u-marginTop--5">Select the cluster that you are having problems with.</p>
                <div className="u-position--relative">
                  <div className="SelectCluster--wrapper">
                    <div className="SelectCluster-menu">
                      <Select
                        className="replicated-select-container u-marginTop--10"
                        classNamePrefix="replicated-select"
                        placeholder="Select a cluster"
                        options={clusters}
                        getOptionLabel={this.getLabel}
                        getOptionValue={(cluster) => cluster.title}
                        value={selectedCluster}
                        onChange={this.handleClusterChange}
                        isOptionSelected={(option) => { option.value === selectedCluster }}
                      />
                    </div>
                  </div>
                </div>
                <div className="u-marginTop--40">
                  <h2 className="u-fontSize--larger u-fontWeight--bold u-color--tuna">Run this command in your cluster</h2>
                  <div className="GenerateBundleCommand--wrapper u-marginTop--15">
                    <pre className="language-bash docker-command">
                      <code className="u-lineHeight--normal u-fontSize--small u-overflow--auto u-marginTop--15">
                        {selectedWatch ?.bundleCommand}
                      </code>
                    </pre>
                    <textarea value={selectedWatch ?.bundleCommand} className="hidden-input" id="docker-command" readOnly={true}></textarea>
                    <div className="u-marginTop--15 u-marginBottom--normal">
                      {showToast ?
                        <span className={`u-color--tuna u-fontSize--small u-fontWeight--medium ${copySuccess ? "u-color--chateauGreen" : "u-color--chestnut"}`}>{copyMessage}</span>
                        :
                        <span className="flex-auto u-color--astral u-fontSize--small u-fontWeight--medium u-textDecoration--underlineOnHover copy-command" data-clipboard-target="#docker-command">
                          Copy command
                          </span>
                      }
                    </div>
                  </div>
                </div>
                <div className="u-marginTop--15"> 
                <button className="btn secondary" type="button" onClick={this.toggleModal}> Upload a support bundle </button>
                </div>
              </div>
            </div>
            :
            <div className="flex-column flex1 u-marginTop--15">
              <div className="EmptyState--wrapper flex-column flex1">
                <div className="EmptyState flex-column flex1 alignItems--center justifyContent--center">
                  <div className="flex alignItems--center justifyContent--center">
                    <span className="icon ship-complete-icon-gh"></span>
                    <span className="deployment-or-text">OR</span>
                    <span className="icon ship-medium-size"></span>
                  </div>
                  <div className="u-textAlign--center u-marginTop--10">
                    <p className="u-fontSize--largest u-color--tuna u-lineHeight--medium u-fontWeight--bold u-marginBottom--10">Deploy to a cluster</p>
                    <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--medium u-fontWeight--medium"> To troubleshoot {watch.watchName} you should first deploy your application to a cluster.</p>
                  </div>
                  <div className="u-marginTop--20">
                    <button className="btn secondary" onClick={this.openClusterModal}>Add a deployment cluster</button>
                  </div>
                </div>
              </div>
            </div>
        }
        </div>
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
              <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Add {this.props.watch.watchName} to a new downstream</h2>
              <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Select one of your existing downstreams to deploy to.</p>
              <AddClusterModal
                onAddCluster={this.addClusterToWatch}
                onRequestClose={this.closeAddClusterModal}
                createDownstreamForCluster={this.createDownstreamForCluster}
                existingDeploymentClusters={[]}
              />
            </div>
          </Modal>
        }
        <Modal
          isOpen={displayUploadModal}
          onRequestClose={this.toggleModal}
          shouldReturnFocusAfterClose={false}
          ariaHideApp={false}
          contentLabel="GenerateBundle-Modal"
          className="Modal MediumSize"
        >
          <div className="Modal-body">
            <GenerateSupportBundleModal
              watch={this.props.watch}
              bundleCommand={selectedWatch?.bundleCommand}
              submitCallback={(bundleId) => {
                this.props.history.push(`/watch/${this.props.match.params.owner}/${this.props.match.params.slug}/troubleshoot/analyze/${bundleId}`);
              }}
            />
          </div>
        </Modal>
      </div >
    );
  }
}

export default withRouter(compose(
  withApollo
)(GenerateSupportBundle));
