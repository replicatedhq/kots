import classNames from "classnames";
import dayjs from "dayjs";
import React, { useEffect, useReducer } from "react";
import Modal from "react-modal";
import { useQuery } from "react-query";

import { KotsPageTitle } from "@components/Head";
import { rbacRoles } from "../../constants/rbac";
import { Repeater } from "../../utilities/repeater";
import { Utilities } from "../../utilities/utilities";
import Icon from "../Icon";
import ErrorModal from "../modals/ErrorModal";
import CodeSnippet from "../shared/CodeSnippet";
import Loader from "../shared/Loader";
import HelmVMNodeRow from "./HelmVMNodeRow";

import "@src/scss/components/apps/HelmVMClusterManagement.scss";

const testData = {
  isHelmVMEnabled: true,
  ha: false,
  nodes: [
    {
      name: "test-helmvm-node",
      isConnected: true,
      isReady: true,
      isPrimaryNode: true,
      canDelete: false,
      kubeletVersion: "v1.28.2",
      cpu: {
        capacity: 8,
        available: 7.466876775,
      },
      memory: {
        capacity: 31.33294677734375,
        available: 24.23790740966797,
      },
      pods: {
        capacity: 110,
        available: 77,
      },
      labels: [
        "beta.kubernetes.io/arch:amd64",
        "beta.kubernetes.io/os:linux",
        "node-role.kubernetes.io/master:",
        "node.kubernetes.io/exclude-from-external-load-balancers:",
        "kubernetes.io/arch:amd64",
        "kubernetes.io/hostname:laverya-kurl",
        "kubernetes.io/os:linux",
        "node-role.kubernetes.io/control-plane:",
      ],
      conditions: {
        memoryPressure: false,
        diskPressure: false,
        pidPressure: false,
        ready: true,
      },
    },
    {
      name: "test-helmvm-worker",
      isConnected: true,
      isReady: true,
      isPrimaryNode: false,
      canDelete: false,
      kubeletVersion: "v1.28.2",
      cpu: {
        capacity: 4,
        available: 3.761070507,
      },
      memory: {
        capacity: 15.50936508178711,
        available: 11.742542266845703,
      },
      pods: {
        capacity: 110,
        available: 94,
      },
      labels: [
        "beta.kubernetes.io/arch:amd64",
        "beta.kubernetes.io/os:linux",
        "kubernetes.io/arch:amd64",
        "kubernetes.io/os:linux",
        "kurl.sh/cluster:true",
      ],
      conditions: {
        memoryPressure: false,
        diskPressure: false,
        pidPressure: false,
        ready: true,
      },
    },
  ],
};

const HelmVMClusterManagement = () => {
  const [state, setState] = useReducer(
    (state, newState) => ({ ...state, ...newState }),
    {
      generating: false,
      command: "",
      expiry: null,
      displayAddNode: false,
      selectedNodeType: "primary",
      generateCommandErrMsg: "",
      helmvm: null,
      deletNodeError: "",
      confirmDeleteNode: "",
      showConfirmDrainModal: false,
      nodeNameToDrain: "",
      drainingNodeName: null,
      drainNodeSuccessful: false,
    }
  );

  const { data: nodes, isLoading: nodesLoading } = useQuery({
    queryKey: "helmVmNodes",
    queryFn: async () => {
      return (
        await fetch(`${process.env.API_ENDPOINT}/helmvm/nodes`, {
          headers: {
            Accept: "application/json",
          },
          credentials: "include",
          method: "GET",
        })
      ).json();
    },
    onError: (err) => {
      if (err.status === 401) {
        Utilities.logoutUser();
        return;
      }
      console.log(
        "failed to get node status list, unexpected status code",
        err.status
      );
    },
    onSuccess: (data) => {
      setState({
        // if cluster doesn't support ha, then primary will be disabled. Force into secondary
        selectedNodeType: !data.ha ? "secondary" : state.selectedNodeType,
      });
    },
    config: {
      refetchInterval: 1000,
      retry: false,
    },
  });

  const deleteNode = (name) => {
    setState({
      confirmDeleteNode: name,
    });
  };

  const cancelDeleteNode = () => {
    setState({
      confirmDeleteNode: "",
    });
  };

  const reallyDeleteNode = () => {
    const name = state.confirmDeleteNode;
    cancelDeleteNode();

    fetch(`${process.env.API_ENDPOINT}/helmvm/nodes/${name}`, {
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      credentials: "include",
      method: "DELETE",
    })
      .then(async (res) => {
        if (!res.ok) {
          if (res.status === 401) {
            Utilities.logoutUser();
            return;
          }
          setState({
            deleteNodeError: `Delete failed with status ${res.status}`,
          });
        }
      })
      .catch((err) => {
        console.log(err);
      });
  };

  const generateWorkerAddNodeCommand = async () => {
    setState({
      generating: true,
      command: "",
      expiry: null,
      generateCommandErrMsg: "",
    });

    fetch(
      `${process.env.API_ENDPOINT}/helmvm/generate-node-join-command-secondary`,
      {
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        credentials: "include",
        method: "POST",
      }
    )
      .then(async (res) => {
        if (!res.ok) {
          setState({
            generating: false,
            generateCommandErrMsg: `Failed to generate command with status ${res.status}`,
          });
        } else {
          const data = await res.json();
          setState({
            generating: false,
            command: data.command,
            expiry: data.expiry,
          });
        }
      })
      .catch((err) => {
        console.log(err);
        setState({
          generating: false,
          generateCommandErrMsg: err ? err.message : "Something went wrong",
        });
      });
  };

  const onDrainNodeClick = (name) => {
    setState({
      showConfirmDrainModal: true,
      nodeNameToDrain: name,
    });
  };

  const drainNode = async (name) => {
    setState({ showConfirmDrainModal: false, drainingNodeName: name });
    fetch(`${process.env.API_ENDPOINT}/helmvm/nodes/${name}/drain`, {
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      credentials: "include",
      method: "POST",
    })
      .then(async (res) => {
        setState({ drainNodeSuccessful: true });
        setTimeout(() => {
          setState({
            drainingNodeName: null,
            drainNodeSuccessful: false,
          });
        }, 3000);
      })
      .catch((err) => {
        console.log(err);
        setState({
          drainingNodeName: null,
          drainNodeSuccessful: false,
        });
      });
  };

  const generatePrimaryAddNodeCommand = async () => {
    setState({
      generating: true,
      command: "",
      expiry: null,
      generateCommandErrMsg: "",
    });

    fetch(
      `${process.env.API_ENDPOINT}/helmvm/generate-node-join-command-primary`,
      {
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        credentials: "include",
        method: "POST",
      }
    )
      .then(async (res) => {
        if (!res.ok) {
          setState({
            generating: false,
            generateCommandErrMsg: `Failed to generate command with status ${res.status}`,
          });
        } else {
          const data = await res.json();
          setState({
            generating: false,
            command: data.command,
            expiry: data.expiry,
          });
        }
      })
      .catch((err) => {
        console.log(err);
        setState({
          generating: false,
          generateCommandErrMsg: err ? err.message : "Something went wrong",
        });
      });
  };

  const onAddNodeClick = () => {
    setState(
      {
        displayAddNode: true,
      },
      async () => {
        await generateWorkerAddNodeCommand();
      }
    );
  };

  const onSelectNodeType = (event) => {
    const value = event.currentTarget.value;
    setState(
      {
        selectedNodeType: value,
      },
      async () => {
        if (state.selectedNodeType === "secondary") {
          await generateWorkerAddNodeCommand();
        } else {
          await generatePrimaryAddNodeCommand();
        }
      }
    );
  };

  const ackDeleteNodeError = () => {
    setState({ deleteNodeError: "" });
  };

  const { displayAddNode, generateCommandErrMsg } = state;

  if (nodesLoading) {
    return (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );
  }

  return (
    <div className="HelmVMClusterManagement--wrapper container flex-column flex1 u-overflow--auto u-paddingTop--50">
      <KotsPageTitle pageName="Cluster Management" />
      <div className="flex-column flex1 alignItems--center u-paddingBottom--50">
        <div className="flex1 flex-column centered-container">
          <div className="u-paddingBottom--30">
            <p className="flex-auto u-fontSize--larger u-fontWeight--bold u-textColor--primary u-paddingBottom--10">
              Cluster Nodes
            </p>
            <p className="u-paddingBottom--10">
              This section lists the nodes that are configured and shows the
              status/health of each. To add additional nodes to this cluster,
              click the "Add node" button at the bottom of this page.
            </p>
            <div className="flex1 u-overflow--auto">
              {(nodes?.nodes || testData?.nodes) &&
                (nodes?.nodes || testData?.nodes).map((node, i) => (
                  <HelmVMNodeRow
                    key={i}
                    node={node}
                    drainingNodeName={state.drainingNodeName}
                    drainNodeSuccessful={state.drainNodeSuccessful}
                    drainNode={nodes?.isHelmVMEnabled ? onDrainNodeClick : null}
                    deleteNode={nodes?.isHelmVMEnabled ? deleteNode : null}
                  />
                ))}
            </div>
          </div>
          {(nodes?.isHelmVMEnabled || testData.isHelmVMEnabled) &&
          Utilities.sessionRolesHasOneOf([rbacRoles.CLUSTER_ADMIN]) ? (
            !displayAddNode ? (
              <div className="flex justifyContent--center alignItems--center">
                <button className="btn primary" onClick={onAddNodeClick}>
                  Add a node
                </button>
              </div>
            ) : (
              <div className="flex-column">
                <div>
                  <p className="u-width--full u-fontSize--larger u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-borderBottom--gray u-paddingBottom--10">
                    Add a node
                  </p>
                </div>
                <div className="flex justifyContent--center alignItems--center u-marginTop--15">
                  <div
                    className={classNames(
                      "BoxedCheckbox flex-auto flex u-marginRight--20",
                      {
                        "is-active": state.selectedNodeType === "primary",
                        "is-disabled": nodes ? !nodes?.ha : !testData?.ha,
                      }
                    )}
                  >
                    <input
                      id="primaryNode"
                      className="u-cursor--pointer hidden-input"
                      type="radio"
                      name="nodeType"
                      value="primary"
                      disabled={nodes ? !nodes?.ha : !testData?.ha}
                      checked={state.selectedNodeType === "primary"}
                      onChange={onSelectNodeType}
                    />
                    <label
                      htmlFor="primaryNode"
                      className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none"
                    >
                      <div className="flex-auto">
                        <Icon
                          icon="commit"
                          size={32}
                          className="clickable u-marginRight--10"
                        />
                      </div>
                      <div className="flex1">
                        <p className="u-textColor--primary u-fontSize--normal u-fontWeight--medium">
                          Primary Node
                        </p>
                        <p className="u-textColor--bodyCopy u-lineHeight--normal u-fontSize--small u-fontWeight--medium u-marginTop--5">
                          Provides high availability
                        </p>
                      </div>
                    </label>
                  </div>
                  <div
                    className={classNames(
                      "BoxedCheckbox flex-auto flex u-marginRight--20",
                      {
                        "is-active": state.selectedNodeType === "secondary",
                      }
                    )}
                  >
                    <input
                      id="secondaryNode"
                      className="u-cursor--pointer hidden-input"
                      type="radio"
                      name="nodeType"
                      value="secondary"
                      checked={state.selectedNodeType === "secondary"}
                      onChange={onSelectNodeType}
                    />
                    <label
                      htmlFor="secondaryNode"
                      className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none"
                    >
                      <div className="flex-auto">
                        <Icon
                          icon="commit"
                          size={32}
                          className="clickable u-marginRight--10"
                        />
                      </div>
                      <div className="flex1">
                        <p className="u-textColor--primary u-fontSize--normal u-fontWeight--medium">
                          Secondary Node
                        </p>
                        <p className="u-textColor--bodyCopy u-lineHeight--normal u-fontSize--small u-fontWeight--medium u-marginTop--5">
                          Optimal for running application workloads
                        </p>
                      </div>
                    </label>
                  </div>
                </div>
                {state.generating && (
                  <div className="flex u-width--full justifyContent--center">
                    <Loader size={60} />
                  </div>
                )}
                {!state.generating && state.command?.length > 0 ? (
                  <>
                    <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--medium u-lineHeight--normal u-marginBottom--5 u-marginTop--15">
                      Run this command on the node you wish to join the cluster
                    </p>
                    <CodeSnippet
                      language="bash"
                      canCopy={true}
                      onCopyText={
                        <span className="u-textColor--success">
                          Command has been copied to your clipboard
                        </span>
                      }
                    >
                      {[state.command.join(" \\\n  ")]}
                    </CodeSnippet>
                    {state.expiry && (
                      <span className="timestamp u-marginTop--15 u-width--full u-textAlign--right u-fontSize--small u-fontWeight--bold u-textColor--primary">
                        {`Expires on ${dayjs(state.expiry).format(
                          "MMM Do YYYY, h:mm:ss a z"
                        )} UTC${(-1 * new Date().getTimezoneOffset()) / 60}`}
                      </span>
                    )}
                  </>
                ) : (
                  <>
                    {generateCommandErrMsg && (
                      <div className="alignSelf--center u-marginTop--15">
                        <span className="u-textColor--error">
                          {generateCommandErrMsg}
                        </span>
                      </div>
                    )}
                  </>
                )}
              </div>
            )
          ) : null}
        </div>
      </div>
      {state.deleteNodeError && (
        <ErrorModal
          errorModal={true}
          toggleErrorModal={ackDeleteNodeError}
          err={"Failed to delete node"}
          errMsg={state.deleteNodeError}
        />
      )}
      <Modal
        isOpen={!!state.confirmDeleteNode}
        onRequestClose={cancelDeleteNode}
        shouldReturnFocusAfterClose={false}
        contentLabel="Confirm Delete Node"
        ariaHideApp={false}
        className="Modal"
      >
        <div className="Modal-body">
          <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">
            Deleting this node may cause data loss. Are you sure you want to
            proceed?
          </p>
          <div className="u-marginTop--10 flex">
            <button
              onClick={reallyDeleteNode}
              type="button"
              className="btn red primary"
            >
              Delete {state.confirmDeleteNode}
            </button>
            <button
              onClick={cancelDeleteNode}
              type="button"
              className="btn secondary u-marginLeft--20"
            >
              Cancel
            </button>
          </div>
        </div>
      </Modal>
      {state.showConfirmDrainModal && (
        <Modal
          isOpen={true}
          onRequestClose={() =>
            setState({
              showConfirmDrainModal: false,
              nodeNameToDrain: "",
            })
          }
          shouldReturnFocusAfterClose={false}
          contentLabel="Confirm Drain Node"
          ariaHideApp={false}
          className="Modal MediumSize"
        >
          <div className="Modal-body">
            <p className="u-fontSize--larger u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
              Are you sure you want to drain {state.nodeNameToDrain}?
            </p>
            <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">
              Draining this node may cause data loss. If you want to delete{" "}
              {state.nodeNameToDrain} you must disconnect it after it has been
              drained.
            </p>
            <div className="u-marginTop--10 flex">
              <button
                onClick={() => drainNode(state.nodeNameToDrain)}
                type="button"
                className="btn red primary"
              >
                Drain {state.nodeNameToDrain}
              </button>
              <button
                onClick={() =>
                  setState({
                    showConfirmDrainModal: false,
                    nodeNameToDrain: "",
                  })
                }
                type="button"
                className="btn secondary u-marginLeft--20"
              >
                Cancel
              </button>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
};

export default HelmVMClusterManagement;
