import * as React from "react";
import Helmet from "react-helmet";
import { withRouter, Link } from "react-router-dom";
import { compose, withApollo, graphql } from "react-apollo";
import AceEditor from "react-ace";
import Modal from "react-modal";
import yaml from "js-yaml";

import { Utilities } from "../../utilities/utilities";
import Loader from "../shared/Loader";
import CodeSnippet from "@src/components/shared/CodeSnippet";
import UploadSupportBundleModal from "../troubleshoot/UploadSupportBundleModal";
import { listSupportBundles } from "../../queries/TroubleshootQueries";
import { collectSupportBundle } from "../../mutations/TroubleshootMutations";

import "../../scss/components/troubleshoot/GenerateSupportBundle.scss";

import "brace/mode/text";
import "brace/mode/yaml";
import "brace/theme/chrome";

const NEW_CLUSTER = "Create a new downstream cluster";
const CUSTOM_SPEC_TEMPLATE = `
apiVersion: troubleshoot.replicated.com/v1beta1
kind: Redactor
metadata:
  name: my-application-name
spec:
  redacts:
`;

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
      showRedactors: false,
      activeRedactorTab: "linkSpec",
      redactorUri: "",
      customRedactorSpec: CUSTOM_SPEC_TEMPLATE,
      specUriSaved: false,
      errorSavingSpecUri: false,
      specSaved: false,
      errorSavingSpec: false,
      savingSpecUriError: "this is an error",
      savingSpecError: "",
      savingRedactor: false
    };
  }

  componentDidMount() {
    const { watch } = this.props;
    const clusters = watch.downstreams;
    if (watch) {
      const watchClusters = clusters.map(c => c.cluster);
      const NEW_ADDED_CLUSTER = { title: NEW_CLUSTER };
      this.setState({ clusters: [NEW_ADDED_CLUSTER, ...watchClusters] });
      this.getRedactor();
    }
  }

  componentDidUpdate(lastProps) {
    const { watch, listSupportBundles, history } = this.props;
    const { totalBundles } = this.state;
    const clusters = watch.downstream;
    if (watch !== lastProps.watch && clusters) {
      const watchClusters = clusters.map(c => c.cluster);
      const NEW_ADDED_CLUSTER = { title: NEW_CLUSTER };
      this.setState({ clusters: [NEW_ADDED_CLUSTER, ...watchClusters] });
    }

    const isLoading = listSupportBundles.loading;
    if (!isLoading) {
      if (totalBundles === null) {
        this.setState({
          totalBundles: listSupportBundles?.listSupportBundles.length
        });
        listSupportBundles.startPolling(2000);
        return;
      }

      if (listSupportBundles?.listSupportBundles.length > totalBundles) {
        listSupportBundles.stopPolling();
        const bundle = listSupportBundles.listSupportBundles[listSupportBundles.listSupportBundles.length - 1];
        history.push(`/app/${watch.slug}/troubleshoot/analyze/${bundle.id}`);
      }
    }
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

    const currentBundles = this.props.listSupportBundles?.listSupportBundles?.map(bundle => {
      bundle.id
    });

    this.props.collectSupportBundle(watch.id, clusterId);

    setTimeout(() => {
      this.redirectOnNewBundle(currentBundles);
    }, 1000);
  }

  redirectOnNewBundle(currentBundles) {
    if (this.props.listSupportBundles?.listSupportBundles?.length === currentBundles.length) {
      setTimeout(() => {
        this.redirectOnNewBundle(currentBundles);
      }, 1000);
      return;
    }
  }

  toggleRedactorAction = (active) => {
    this.setState({
      activeRedactorTab: active,
      specSaved: false,
      errorSavingSpec: false,
      savingSpecError: "",
      savingSpecUriError: ""
    });
  }

  toggleModal = () => {
    this.setState({
      displayUploadModal: !this.state.displayUploadModal
    })
  }

  handleFormChange = (field, val) => {
    let nextState = {};
    nextState[field] = val;
    this.setState(nextState);
  }

  onRedactorChange = (value) => {
    this.setState({
      customRedactorSpec: value,
    });
  }

  getRedactor = () => {
    fetch(`${window.env.API_ENDPOINT}/redact/get`, {
      headers: {
        "Authorization": `${Utilities.getToken()}`,
        "Content-Type": "application/json",
      },
      method: "GET",
    })
      .then(async (res) => {
        const response = await res.json();
        try {
          const r = yaml.safeLoad(response.updatedSpec);
          if (typeof r === "object") {
            this.setState({ customRedactorSpec: response.updatedSpec, showRedactors: response.updatedSpec !== "", activeRedactorTab: "writeSpec" });
          } else {
            this.setState({ redactorUri: response.updatedSpec, showRedactors: response.updatedSpec !== "", activeRedactorTab: "linkSpec" });
          }
        } catch (e) {
          console.log(e);
        }
      })
      .catch((err) => {
        console.log(err);
      });
  }

  saveRedactor = () => {
    const { activeRedactorTab, redactorUri, customRedactorSpec } = this.state;
    const isRedactorLink = activeRedactorTab === "linkSpec";
    this.setState({ errorSavingSpec: false, savingSpecError: "", savingSpecUriError: "" });

    let payload;
    if (isRedactorLink) {
      if (!redactorUri.length || redactorUri === "") {
        return this.setState({ errorSavingSpecUri: true, savingSpecUriError: "No uri was provided" })
      }
      payload = {
        redactSpecUrl: redactorUri
      };
    } else {
      payload = {
        redactSpec: customRedactorSpec
      };
    }

    this.setState({ savingRedactor: true });
    fetch(`${window.env.API_ENDPOINT}/redact/set`, {
      headers: {
        "Authorization": `${Utilities.getToken()}`,
        "Content-Type": "application/json",
        "Accept": "application/json",
      },
      method: "PUT",
      body: JSON.stringify(payload)
    })
      .then(() => {
        this.setState({ savingRedactor: false, specSaved: true });
        setTimeout(() => {
          this.setState({ specSaved: false });
        }, 3000);
      })
      .catch((err) => {
        if (isRedactorLink) {
          this.setState({ savingRedactor: false, errorSavingSpecUri: true, savingSpecUriError: err });
        } else {
          this.setState({ savingRedactor: false, errorSavingSpec: true, savingSpecError: err });
        }
      });
  }

  renderRedactorTab = () => {
    const { activeRedactorTab, redactorUri, customRedactorSpec, savingRedactor } = this.state;
    switch (activeRedactorTab) {
      case "linkSpec":
        return (
          <div className="flex1">
            <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--5">Where is your spec located</p>
            <p className="u-lineHeight--normal u-fontSize--small u-color--dustyGray u-fontWeight--medium u-marginBottom--10">Provide the URI where your redactor spec is located.</p>
            <input type="text" className="Input" placeholder="github.com/org/myrepo/redactor.yaml" value={redactorUri} autoComplete="" onChange={(e) => { this.handleFormChange("redactorUri", e.target.value) }} />
            <div className="u-marginTop--10 flex alignItems--center">
              <button className="btn secondary blue" onClick={this.saveRedactor} disabled={savingRedactor}>{savingRedactor ? "Saving" : "Save"}</button>
              {this.state.specSaved &&
                <span className="u-marginLeft--10 flex alignItems--center">
                  <span className="icon checkmark-icon u-marginRight--5" />
                  <span className="u-color--chateauGreen u-fontSize--small u-fontWeight--medium u-lineHeight--normal">Saved</span>
                </span>
              }
              {this.state.errorSavingSpecUri &&
                <span className="u-marginLeft--10 flex alignItems--center">
                  <span className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal">{this.state.savingSpecUriError}</span>
                </span>
              }
            </div>
          </div>
        );
      case "writeSpec":
        return (
          <div>
            <div className="flex1 u-border--gray">
              <AceEditor
                ref={(input) => this.refAceEditor = input}
                mode="yaml"
                theme="chrome"
                className="flex1 flex"
                readOnly={true}
                value={customRedactorSpec}
                height="380px"
                width="100%"
                markers={this.state.activeMarkers}
                editorProps={{
                  $blockScrolling: Infinity,
                  useSoftTabs: true,
                  tabSize: 2,
                }}
                onChange={(value) => this.onRedactorChange(value)}
                setOptions={{
                  scrollPastEnd: false,
                  showGutter: true,
                }}
              />
            </div>
            <div className="u-marginTop--10 flex alignItems--center">
              <button className="btn secondary blue" onClick={this.saveRedactor} disabled={savingRedactor}>{savingRedactor ? "Saving spec" : "Save spec"}</button>
              {this.state.specSaved &&
                <span className="u-marginLeft--10 flex alignItems--center">
                  <span className="icon checkmark-icon u-marginRight--5" />
                  <span className="u-color--chateauGreen u-fontSize--small u-fontWeight--medium u-lineHeight--normal">Spec saved</span>
                </span>
              }
              {this.state.errorSavingSpec &&
                <span className="u-marginLeft--10 flex alignItems--center">
                  <span className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal">{this.state.savingSpecError}</span>
                </span>
              }
            </div>
          </div>
        );
      default:
        return null;
    }
  }

  render() {
    const { selectedCluster, displayUploadModal, showRunCommand, isGeneratingBundle, showRedactors } = this.state;
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
          {!watchClusters.length && !this.props.listSupportBundles?.listSupportBundles?.length ?
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
                <div>
                  <button className="btn primary blue u-marginTop--20" type="button" onClick={this.collectBundle.bind(this, watchClusters[0].cluster.id)}>Analyze {appTitle} </button>
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
              {showRedactors ?
                <div>
                  <div className="u-marginTop--40">
                    <div className="flex action-tab-bar">
                      <span className={`${this.state.activeRedactorTab === "linkSpec" ? "is-active" : ""} tab-item`} onClick={() => this.toggleRedactorAction("linkSpec")}>Link to a spec</span>
                      <span className={`${this.state.activeRedactorTab === "writeSpec" ? "is-active" : ""} tab-item`} onClick={() => this.toggleRedactorAction("writeSpec")}>Write your own spec</span>
                    </div>
                    <div className="flex-column flex1 action-content">
                      {this.renderRedactorTab()}
                    </div>
                  </div>
                </div>
              :
                <div>
                  <div className="u-marginTop--40">
                    If you would like to use custom redactors, <a href="#" className="replicated-link" onClick={(e) => this.toggleShow(e, "showRedactors")}>click here</a> to link to a redactor file or you can write your own.
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
  graphql(collectSupportBundle, {
    props: ({ mutate }) => ({
      collectSupportBundle: (appId, clusterId) => mutate({ variables: { appId, clusterId } })
    })
  }),
)(GenerateSupportBundle));
