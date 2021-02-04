import React, { Component, Fragment } from "react";
import classNames from "classnames";
import moment from "moment";
import { withRouter } from "react-router-dom";
import { Helmet } from "react-helmet";
import CodeSnippet from "../shared/CodeSnippet";
import NodeRow from "./NodeRow";
import Loader from "../shared/Loader";
import { rbacRoles } from "../../constants/rbac";
import { Utilities } from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";
import ErrorModal from "../modals/ErrorModal";
import Modal from "react-modal";

import "@src/scss/components/apps/ClusterNodes.scss";

export class ClusterNodes extends Component {
  state = {
    generating: false,
    command: "",
    expiry: null,
    displayAddNode: false,
    selectedNodeType: "worker", // Change when master node script is enabled
    generateCommandErrMsg: "",
    kurl: null,
    getNodeStatusJob: new Repeater(),
    deletNodeError: "",
    confirmDeleteNode: "",
    drainingNode: false,
    drainNodeSuccessful: false
  }

  componentDidMount() {
    this.getNodeStatus();
    this.state.getNodeStatusJob.start(this.getNodeStatus, 1000);
  }

  componentWillUnmount() {
    this.state.getNodeStatusJob.stop();
  }

  getNodeStatus = async () => {
    try {
      const res = await fetch(`${window.env.API_ENDPOINT}/kurl/nodes`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Accept": "application/json",
        },
        method: "GET",
      });
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }
        console.log("failed to get node status list, unexpected status code", res.status);
        return;
      }
      const response = await res.json();
      this.setState({
        kurl: response,
      });
      return response;
    } catch(err) {
      console.log(err);
      throw err;
    }
  }

  deleteNode = (name) => {
    this.setState({
      confirmDeleteNode: name,
    });
  }

  cancelDeleteNode = () => {
    this.setState({
      confirmDeleteNode: "",
    });
  }

  reallyDeleteNode = () => {
    const name = this.state.confirmDeleteNode;
    this.cancelDeleteNode();

    fetch(`${window.env.API_ENDPOINT}/kurl/nodes/${name}`, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
        "Accept": "application/json",
      },
      method: "DELETE",
    })
      .then(async (res) => {
        if (!res.ok) {
          if (res.status === 401) {
            Utilities.logoutUser();
            return;
          }
          if (res.status === 422) {
            this.setState({
              deleteNodeError: "The ekco add-on is required to delete nodes but was not found in your cluster. https://kurl.sh/docs/add-ons/ekco",
            });
            return;
          }
          this.setState({
            deleteNodeError: `Delete failed with status ${res.status}`,
          });
        }
      })
      .catch((err) => {
        console.log(err);
      })
  }

  generateWorkerAddNodeCommand = async () => {
    this.setState({ generating: true, command: "", expiry: null, generateCommandErrMsg: "" });

    fetch(`${window.env.API_ENDPOINT}/kurl/generate-node-join-command-worker`, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
        "Accept": "application/json",
      },
      method: "POST",
    })
      .then(async (res) => {
        const data = await res.json();
        this.setState({ generating: false, command: data.command, expiry: data.expiry });
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          generating: false,
          generateCommandErrMsg: err ? err.message : "Something went wrong",
        });
      });
  }

  drainNode = async (name) => {
    this.setState({ drainingNode: true });
    fetch(`${window.env.API_ENDPOINT}/kurl/nodes/${name}/drain`, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
        "Accept": "application/json",
      },
      method: "POST",
    })
      .then(async (res) => {
        this.setState({
          drainingNode: false,
          drainNodeSuccessful: true
        });
        setTimeout(() => {
          this.setState({ drainNodeSuccessful: false });
        }, 3000);
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          drainingNode: false,
          drainNodeSuccessful: false
        });
      })
  }

  generateMasterAddNodeCommand = async () => {
    this.setState({ generating: true, command: "", expiry: null, generateCommandErrMsg: "" });

    fetch(`${window.env.API_ENDPOINT}/kurl/generate-node-join-command-master`, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
        "Accept": "application/json",
      },
      method: "POST",
    })
      .then(async (res) => {
        const data = await res.json();
        this.setState({ generating: false, command: data.command, expiry: data.expiry });
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          generating: false,
          generateCommandErrMsg: err ? err.message : "Something went wrong",
        });
      });
  }

  onAddNodeClick = () => {
    this.setState({
      displayAddNode: true
    }, async () => {
      await this.generateWorkerAddNodeCommand();
    });
  }

  onSelectNodeType = event => {
    const value = event.currentTarget.value;
    this.setState({
      selectedNodeType: value
    }, async () => {
      if (this.state.selectedNodeType === "worker") {
        await this.generateWorkerAddNodeCommand();
      } else {
        await this.generateMasterAddNodeCommand();
      }
    });
  }

  ackDeleteNodeError = () => {
    this.setState({ deleteNodeError: "" });
  }

  render() {
    const { kurl } = this.state;
    const { displayAddNode, generateCommandErrMsg } = this.state;

    if (!kurl) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }
    return (
      <div className="ClusterNodes--wrapper container flex-column flex1 u-overflow--auto u-paddingTop--50">
        <Helmet>
          <title>{`${this.props.appName ? `${this.props.appName} Admin Console` : "Admin Console"}`}</title>
        </Helmet>
        <div className="flex-column flex1 alignItems--center">
          <div className="flex1 flex-column centered-container">
            <div className="u-paddingBottom--30">
              <p className="flex-auto u-fontSize--larger u-fontWeight--bold u-color--tuna u-paddingBottom--10">Your nodes</p>
              <div className="flex1 u-overflow--auto">
                {kurl?.nodes && kurl?.nodes.map((node, i) => (
                  <NodeRow
                    key={i}
                    node={node}
                    drainingNode={this.state.drainingNode}
                    drainNodeSuccessful={this.state.drainNodeSuccessful}
                    drainNode={kurl?.isKurlEnabled ? this.drainNode : null}
                    deleteNode={kurl?.isKurlEnabled ? this.deleteNode : null} />
                ))}
              </div>
            </div>
            {kurl?.isKurlEnabled && Utilities.sessionRolesHasOneOf([rbacRoles.CLUSTER_ADMIN]) ?
              !displayAddNode
                ? (
                  <div className="flex justifyContent--center alignItems--center">
                    <button className="btn primary" onClick={this.onAddNodeClick}>Add a node</button>
                  </div>
                )
                : (
                  <div className="flex-column">
                    <div>
                      <p className="u-width--full u-fontSize--larger u-color--tuna u-fontWeight--bold u-lineHeight--normal u-borderBottom--gray u-paddingBottom--10">
                        Add a Node
                      </p>
                    </div>
                    <div className="flex justifyContent--center alignItems--center u-marginTop--15">
                      <div className={classNames("BoxedCheckbox flex-auto flex u-marginRight--20", {
                        "is-active": this.state.selectedNodeType === "master",
                        "is-disabled": !kurl?.ha
                      })}>
                        <input
                          id="masterNode"
                          className="u-cursor--pointer hidden-input"
                          type="radio"
                          name="nodeType"
                          value="master"
                          disabled={!kurl?.ha}
                          checked={this.state.selectedNodeType === "master"}
                          onChange={this.onSelectNodeType}
                        />
                        <label
                          htmlFor="masterNode"
                          className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none">
                          <div className="flex-auto">
                            <span className="icon clickable commitOptionIcon u-marginRight--10" />
                          </div>
                          <div className="flex1">
                            <p className="u-color--tuna u-fontSize--normal u-fontWeight--medium">Master Node</p>
                            <p className="u-color--dustyGray u-lineHeight--normal u-fontSize--small u-fontWeight--medium u-marginTop--5">Provides high availability</p>
                          </div>
                        </label>
                      </div>
                      <div className={classNames("BoxedCheckbox flex-auto flex u-marginRight--20", {
                        "is-active": this.state.selectedNodeType === "worker"
                      })}>
                        <input
                          id="workerNode"
                          className="u-cursor--pointer hidden-input"
                          type="radio"
                          name="nodeType"
                          value="worker"
                          checked={this.state.selectedNodeType === "worker"}
                          onChange={this.onSelectNodeType}
                        />
                        <label htmlFor="workerNode" className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none">
                          <div className="flex-auto">
                            <span className="icon clickable commitOptionIcon u-marginRight--10" />
                          </div>
                          <div className="flex1">
                            <p className="u-color--tuna u-fontSize--normal u-fontWeight--medium">Worker Node</p>
                            <p className="u-color--dustyGray u-lineHeight--normal u-fontSize--small u-fontWeight--medium u-marginTop--5">Optimal for running application workloads</p>
                          </div>
                        </label>
                      </div>
                    </div>
                    {this.state.generating && (
                      <div className="flex u-width--full justifyContent--center">
                        <Loader size={60} />
                      </div>
                    )}
                    {!this.state.generating && this.state.command?.length > 0
                      ? (
                        <Fragment>
                          <p className="u-fontSize--normal u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginBottom--5 u-marginTop--15">
                            Run this command on the node you wish to join the cluster
                          </p>
                          <CodeSnippet
                            language="bash"
                            canCopy={true}
                            onCopyText={<span className="u-color--chateauGreen">Command has been copied to your clipboard</span>}
                          >
                            {[this.state.command.join(" \\\n  ")]}
                          </CodeSnippet>
                          {this.state.expiry && (
                            <span className="timestamp u-marginTop--15 u-width--full u-textAlign--right u-fontSize--small u-fontWeight--bold u-color--tuna">
                              {`Expires on ${moment(this.state.expiry).format("MMM Do YYYY, h:mm:ss a z")} UTC${ -1 * (new Date().getTimezoneOffset()) / 60}`}
                            </span>
                          )}
                        </Fragment>
                      )
                      : (
                        <Fragment>
                          {generateCommandErrMsg &&
                            <div className="alignSelf--center u-marginTop--15">
                              <span className="u-color--chestnut">{generateCommandErrMsg}</span>
                            </div>
                          }
                        </Fragment>
                      )
                    }
                  </div>
                )
            : null}
          </div>
        </div>
        {this.state.deleteNodeError &&
          <ErrorModal
            errorModal={true}
            toggleErrorModal={this.ackDeleteNodeError}
            err={"Failed to delete node"}
            errMsg={this.state.deleteNodeError}
          />
        }
        <Modal
          isOpen={!!this.state.confirmDeleteNode}
          onRequestClose={this.cancelDeleteNode}
          shouldReturnFocusAfterClose={false}
          contentLabel="Confirm Delete Node"
          ariaHideApp={false}
          className="Modal"
        >
          <div className="Modal-body">
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">
              Deleting this node may cause data loss. Are you sure you want to proceed?
            </p>
            <div className="u-marginTop--10 flex">
              <button
                onClick={this.reallyDeleteNode}
                type="button"
                className="btn blue primary"
              >
                Delete {this.state.confirmDeleteNode}
              </button>
              <button
                onClick={this.cancelDeleteNode}
                type="button"
                className="btn secondary u-marginLeft--20"
              >
                Cancel
              </button>
            </div>
          </div>
        </Modal>
      </div>
    );
  }
}

export default withRouter(ClusterNodes);
