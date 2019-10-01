import * as React from "react";
import Helmet from "react-helmet";
import { withRouter, Link } from "react-router-dom";
import { compose, withApollo, graphql } from "react-apollo";

import Select from "react-select";
import Modal from "react-modal";

import CodeSnippet from "@src/components/shared/CodeSnippet";
import AddClusterModal from "../shared/modals/AddClusterModal";
import UploadSupportBundleModal from "../troubleshoot/UploadSupportBundleModal";
import { listSupportBundles } from "../../queries/TroubleshootQueries";
import { isKotsApplication } from "../../utilities/utilities";

import "../../scss/components/troubleshoot/GenerateSupportBundle.scss";

const NEW_CLUSTER = "Create a new downstream cluster";

class GenerateSupportBundle extends React.Component {
  constructor(props) {
    super(props);

    const clustersArray = props.watch.watches || props.watch.downstreams;
    this.state = {
      clusters: [],
      selectedCluster: clustersArray.length ? clustersArray[0].cluster : "",
      addNewClusterModal: false,
      displayUploadModal: false
    };
  }

  componentDidMount() {
    const { watch } = this.props;
    const clusters = watch.watches || watch.downstreams;
    if (watch) {
      const watchClusters = clusters.map(c => c.cluster);
      const NEW_ADDED_CLUSTER = { title: NEW_CLUSTER };
      this.setState({ clusters: [NEW_ADDED_CLUSTER, ...watchClusters] });
    }
  }

  componentDidUpdate(lastProps) {
    const { watch } = this.props;
    const clusters = watch.watches || watch.downstream;
    if (watch !== lastProps.watch && clusters) {
      const watchClusters = clusters.map(c => c.cluster);
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
    localStorage.setItem("clusterRedirect", `/watch/${watch.slug}/troubleshoot/generate`);
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
    const { clusters, selectedCluster, addNewClusterModal, displayUploadModal } = this.state;
    const { watch } = this.props;
    const watchClusters = watch.watches || watch.downstreams;
    const selectedWatch = watchClusters.find(c => c.cluster.id === selectedCluster.id);
    const appTitle = watch.watchName || watch.name;
    return (
      <div className="GenerateSupportBundle--wrapper container flex-column u-overflow--auto u-paddingTop--30 u-paddingBottom--20 alignItems--center">
        <Helmet>
          <title>{`${appTitle} Troubleshoot`}</title>
        </Helmet>
        <div className="GenerateSupportBundle">
          {!watchClusters.length && !this.props.listSupportBundles?.listSupportBundles?.length ?
            <Link to={`/watch/${watch.slug}/troubleshoot`} className="replicated-link u-marginRight--5"> &lt; Support Bundle List </Link> : null
           }
          <div className="u-marginTop--15">
            <h2 className="u-fontSize--larger u-fontWeight--bold u-color--tuna">Analyze {appTitle} for support</h2>
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--medium u-marginTop--5">If you’re having issues with {appTitle}, you can analyze the current state to receive insights that are useful to remediate or to share with the application vendor for support.</p>
          </div>
          {watchClusters.length ?
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
                  <CodeSnippet
                    language="bash"
                    canCopy={true}
                    onCopyText={<span className="u-color--chateauGreen">Command has been copied to your clipboard</span>}
                  >
                    {selectedWatch?.bundleCommand?.split("\n") || watch.bundleCommand?.split("\n")}
                  </CodeSnippet>
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
                    <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--medium u-fontWeight--medium"> To troubleshoot {appTitle} you should first deploy your application to a cluster.</p>
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
              <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Add {appTitle} to a new downstream</h2>
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
            <UploadSupportBundleModal
              watch={this.props.watch}
              bundleCommand={selectedWatch ?.bundleCommand}
              submitCallback={(bundleId) => {
                let url;
                if (isKotsApplication(watch)) {
                  url = `/app/${this.props.match.params.slug}/troubleshoot/analyze/${bundleId}`;
                } else {
                  url = `/watch/${this.props.match.params.owner}/${this.props.match.params.slug}/troubleshoot/analyze/${bundleId}`;
                }
                this.props.history.push(url);
              }}
            />
          </div>
        </Modal>
      </div >
    );
  }
}

export default withRouter(compose(
  withApollo,
  graphql(listSupportBundles, {
    name: "listSupportBundles",
    options: props => {
      return {
        variables: {
          watchSlug: props.watch.slug
        },
        fetchPolicy: "no-cache",
      }
    }
  }),
)(GenerateSupportBundle));