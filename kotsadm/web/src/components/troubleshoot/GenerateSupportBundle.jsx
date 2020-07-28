import * as React from "react";
import Helmet from "react-helmet";
import { withRouter, Link } from "react-router-dom";
import { compose, withApollo, graphql } from "react-apollo";
import Modal from "react-modal";

import Loader from "../shared/Loader";
import CodeSnippet from "@src/components/shared/CodeSnippet";
import UploadSupportBundleModal from "../troubleshoot/UploadSupportBundleModal";
import ConfigureRedactorsModal from "./ConfigureRedactorsModal";
import { Utilities } from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";

import "../../scss/components/troubleshoot/GenerateSupportBundle.scss";

const NEW_CLUSTER = "Create a new downstream cluster";

class GenerateSupportBundle extends React.Component {
  constructor(props) {
    super(props);

    const clustersArray = props.watch.downstreams;
    this.state = {
      clusters: [],
      selectedCluster: clustersArray.length ? clustersArray[0].cluster : "",
      displayUploadModal: false,
      totalBundles: null,
      showRunCommand: false,
      isGeneratingBundle: false,
      displayRedactorModal: false,
      loadingSupportBundles: false,
      supportBundles: [],
      listSupportBundlesJob: new Repeater(),
    };
  }

  componentDidMount() {
    const { watch } = this.props;
    const clusters = watch.downstreams;
    if (watch) {
      const watchClusters = clusters.map(c => c.cluster);
      const NEW_ADDED_CLUSTER = { title: NEW_CLUSTER };
      this.setState({ clusters: [NEW_ADDED_CLUSTER, ...watchClusters] });
    }
    this.listSupportBundles();
  }

  componentWillUnmount() {
    this.state.listSupportBundlesJob.stop();
  }

  componentDidUpdate(lastProps) {
    const { watch, history } = this.props;
    const { totalBundles, loadingSupportBundles, supportBundles } = this.state;
    const clusters = watch.downstream;
    if (watch !== lastProps.watch && clusters) {
      const watchClusters = clusters.map(c => c.cluster);
      const NEW_ADDED_CLUSTER = { title: NEW_CLUSTER };
      this.setState({ clusters: [NEW_ADDED_CLUSTER, ...watchClusters] });
    }

    if (!loadingSupportBundles) {
      if (totalBundles === null) {
        this.setState({
          totalBundles: supportBundles?.length
        });
        this.state.listSupportBundlesJob.start(this.listSupportBundles, 2000);
        return;
      }

      if (supportBundles?.length > totalBundles) {
        this.state.listSupportBundlesJob.stop();
        const bundle = supportBundles[0]; // safe. there's at least 1 element in this array.
        history.push(`/app/${watch.slug}/troubleshoot/analyze/${bundle.id}`);
      }
    }
  }

  listSupportBundles = () => {
    return new Promise((resolve, reject) => {
      this.setState({ loadingSupportBundles: true });

      fetch(`${window.env.API_ENDPOINT}/troubleshoot/app/${this.props.watch?.slug}/supportbundles`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      })
        .then(async (res) => {
          const response = await res.json();
          this.setState({
            supportBundles: response.supportBundles,
            loadingSupportBundles: false,
          });
          resolve();
        })
        .catch((err) => {
          console.log(err);
          this.setState({ loadingSupportBundles: false });
          reject(err);
        });
    });
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

  toggleShow = (ev, section) => {
    ev.preventDefault();
    this.setState({
      [section]: !this.state[section],
    });
  }

  getLabel = ({ shipOpsRef, gitOpsRef, title }) => {
    return (
      <div style={{ alignItems: "center", display: "flex" }}>
        <span style={{ fontSize: 18, marginRight: "0.5em" }}>{this.renderIcons(shipOpsRef, gitOpsRef)}</span>
        <span style={{ fontSize: 14 }}>{title}</span>
      </div>
    );
  }

  collectBundle = (clusterId) => {
    const { watch } = this.props;

    this.setState({
      isGeneratingBundle: true,
    });

    const currentBundles = this.state.supportBundles?.map(bundle => {
      bundle.id
    });

    fetch(`${window.env.API_ENDPOINT}/troubleshoot/supportbundle/app/${watch?.id}/cluster/${clusterId}/collect`, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "POST",
    })
      .then(() => {
        this.redirectOnNewBundle(currentBundles);
      })
      .catch((err) => {
        console.log(err);
        this.setState({ isGeneratingBundle: false });
      });
  }

  redirectOnNewBundle(currentBundles) {
    if (this.state.supportBundles?.length === currentBundles.length) {
      setTimeout(() => {
        this.redirectOnNewBundle(currentBundles);
      }, 1000);
      return;
    }
  }

  toggleModal = () => {
    this.setState({
      displayUploadModal: !this.state.displayUploadModal
    })
  }

  toggleRedactorModal = () => {
    this.setState({
      displayRedactorModal: !this.state.displayRedactorModal
    })
  }

  render() {
    const { selectedCluster, displayUploadModal, showRunCommand, isGeneratingBundle } = this.state;
    const { watch } = this.props;
    const watchClusters = watch.downstreams;
    const selectedWatch = watchClusters.find(c => c.cluster.id === selectedCluster.id);
    const appTitle = watch.watchName || watch.name;

    let command = selectedWatch?.bundleCommand || watch.bundleCommand;
    if (command) {
      command = command.replace("API_ADDRESS", window.location.origin);
    }
    return (
      <div className="GenerateSupportBundle--wrapper container flex-column u-overflow--auto u-paddingTop--30 u-paddingBottom--20 alignItems--center">
        <Helmet>
          <title>{`${appTitle} Troubleshoot`}</title>
        </Helmet>
        <div className="GenerateSupportBundle">
          {!watchClusters.length && !this.state.supportBundles?.length ?
            <Link to={`/watch/${watch.slug}/troubleshoot`} className="replicated-link u-marginRight--5"> &lt; Support Bundle List </Link> : null
           }
          <div className="u-marginTop--15">
            <h2 className="u-fontSize--larger u-fontWeight--bold u-color--tuna">Analyze {appTitle} for support</h2>
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--medium u-marginTop--5">
              To diagnose any problems with the application, click the button below to get started. This will
              collect logs, resources and other data from the running application and analyze them against a set of known
              problems in {appTitle}. Logs, cluster info and other data will not leave your cluster.
            </p>
          </div>
          <div className="flex1 flex-column u-paddingRight--30">
            <div>
              {isGeneratingBundle ?
                <div className="flex1 flex-column justifyContent--center alignItems--center">
                  <Loader size="60" />
                </div>
              :
                <div className="flex alignItems--center u-marginTop--20">
                  <button className="btn primary blue" type="button" onClick={this.collectBundle.bind(this, watchClusters[0].cluster.id)}>Analyze {appTitle}</button>
                  <span className="replicated-link flex alignItems--center u-fontSize--small u-marginLeft--20" onClick={this.toggleRedactorModal}><span className="icon clickable redactor-spec-icon u-marginRight--5" /> Configure redaction</span>
                </div>
              }
              {showRunCommand ?
                <div>
                  <div className="u-marginTop--40">
                    <h2 className="u-fontSize--larger u-fontWeight--bold u-color--tuna">Run this command in your cluster</h2>
                    <CodeSnippet
                      language="bash"
                      canCopy={true}
                      onCopyText={<span className="u-color--chateauGreen">Command has been copied to your clipboard</span>}
                    >
                      {command?.split("\n")}
                    </CodeSnippet>
                  </div>
                  <div className="u-marginTop--15">
                    <button className="btn secondary" type="button" onClick={this.toggleModal}> Upload a support bundle </button>
                  </div>
                </div>
              :
                <div>
                  <div className="u-marginTop--40">
                    If you'd prefer, <a href="#" className="replicated-link" onClick={(e) => this.toggleShow(e, "showRunCommand")}>click here</a> to get a command to manually generate a support bundle.
                  </div>
                </div>
              }
            </div>
          </div>
        </div>
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
              onBundleUploaded={(bundleId) => {
                const url = `/app/${this.props.match.params.slug}/troubleshoot/analyze/${bundleId}`;
                this.props.history.push(url);
              }}
            />
          </div>
        </Modal>
        {this.state.displayRedactorModal && 
          <ConfigureRedactorsModal onClose={this.toggleRedactorModal} />
        }
      </div >
    );
  }
}

export default withRouter(GenerateSupportBundle);
