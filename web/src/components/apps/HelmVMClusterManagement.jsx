import classNames from "classnames";
import dayjs from "dayjs";
import React, { useEffect, useMemo, useReducer, useState } from "react";
import Modal from "react-modal";
import { useMutation, useQuery } from "react-query";
import { useNavigate } from "react-router-dom";

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

const HelmVMClusterManagement = ({ fromLicenseFlow = false }) => {
  const [state, setState] = useReducer(
    (state, newState) => ({ ...state, ...newState }),
    {
      displayAddNode: false,
      selectedNodeType: "primary",
      helmvm: null,
      deletNodeError: "",
      confirmDeleteNode: "",
      showConfirmDrainModal: false,
      nodeNameToDrain: "",
      drainingNodeName: null,
      drainNodeSuccessful: false,
    }
  );
  const [selectedNodeTypes, setSelectedNodeTypes] = useState([]);
  const [useStaticToken, setUseStaticToken] = useState(false);

  const navigate = useNavigate();

  const nodes = testData;
  const nodesLoading = false;

  // #region queries

  // const { data: nodes, isLoading: nodesLoading } = useQuery({
  //   queryKey: "helmVmNodes",
  //   queryFn: async () => {
  //     return (
  //       await fetch(`${process.env.API_ENDPOINT}/helmvm/nodes`, {
  //         headers: {
  //           Accept: "application/json",
  //         },
  //         credentials: "include",
  //         method: "GET",
  //       })
  //     ).json();
  //   },
  //   onError: (err) => {
  //     if (err.status === 401) {
  //       Utilities.logoutUser();
  //       return;
  //     }
  //     console.log(
  //       "failed to get node status list, unexpected status code",
  //       err.status
  //     );
  //   },
  //   onSuccess: (data) => {
  //     setState({
  //       // if cluster doesn't support ha, then primary will be disabled. Force into secondary
  //       selectedNodeType: !data.ha ? "secondary" : state.selectedNodeType,
  //     });
  //   },
  //   config: {
  //     refetchInterval: 1000,
  //     retry: false,
  //   },
  // });

  const {
    data: generateSecondaryAddNodeCommand,
    isLoading: generateSecondaryAddNodeCommandLoading,
    error: generateSecondaryAddNodeCommandError,
  } = useQuery({
    queryKey: "generateSecondaryAddNodeCommand",
    queryFn: async () => {
      return (
        await fetch(
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
      ).json();
    },
  });

  const {
    data: generatePrimaryAddNodeCommand,
    isLoading: generatePrimaryAddNodeCommandLoading,
    error: generatePrimaryAddNodeCommandError,
  } = useQuery({
    queryKey: "generatePrimaryAddNodeCommand",
    queryFn: async () => {
      return (
        await fetch(
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
      ).json();
    },
  });

  const {
    mutate: addNodeType,
    isLoading: addNodeTypeLoading,
    error: addNodeTypeError,
  } = useMutation({
    mutationFn: async () => {
      return (
        await fetch(`${process.env.API_ENDPOINT}/helmvm/nodes`, {
          headers: {
            "Content-Type": "application/json",
            Accept: "application/json",
          },
          credentials: "include",
          method: "POST",
        })
      ).json();
    },
    onSuccess: () => {
      // if (fromLicenseFlow && data.isConfigurable) {
      //   navigate(`/${data.slug}/config`, { replace: true });
      //   return;
      // }
    },
  });
  // #endregion

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

  const onAddNodeClick = () => {
    setState({
      displayAddNode: true,
    });
  };

  const ackDeleteNodeError = () => {
    setState({ deleteNodeError: "" });
  };

  const NODE_TYPES = [
    "controlplane",
    "db",
    "app",
    "search",
    "webserver",
    "jobs",
  ];

  const determineDisabledState = (nodeType, selectedNodeTypes) => {
    if (nodeType === "controlplane") {
      const numControlPlanes = testData.nodes.reduce((acc, node) => {
        if (node.labels.includes("controlplane")) {
          acc++;
        }
        return acc;
      });
      return numControlPlanes === 3;
    }
    if (
      (nodeType === "db" || nodeType === "search") &&
      selectedNodeTypes.includes("webserver")
    ) {
      return true;
    }
    return false;
  };

  const handleSelectNodeType = (e) => {
    const nodeType = e.currentTarget.value;
    let types = selectedNodeTypes;

    if (nodeType === "webserver") {
      types = types.filter((type) => type !== "db" && type !== "search");
    }

    if (selectedNodeTypes.includes(nodeType)) {
      setSelectedNodeTypes(types.filter((type) => type !== nodeType));
    } else {
      setSelectedNodeTypes([...types, nodeType]);
    }
  };

  const handleContinue = () => {
    if (data.isConfigurable) {
      navigate(`/${data.slug}/config`, { replace: true });
      return;
    }
  };

  if (nodesLoading) {
    return (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );
  }

  return (
    <div className="HelmVMClusterManagement--wrapper container flex-column flex1 u-overflow--auto u-paddingTop--50 tw-font-sans">
      <KotsPageTitle pageName="Cluster Management" />
      <div className="flex-column flex1 alignItems--center u-paddingBottom--50">
        <div className="centered-container tw-mb-10 tw-pb-6 tw-flex tw-flex-col tw-gap-4">
          <p className="flex-auto u-fontSize--larger u-fontWeight--bold u-textColor--primary">
            Cluster Nodes
          </p>
          <div className="tw-flex tw-gap-6 tw-items-center">
            <p className="tw-text-base tw-flex-1">
              This section lists the nodes that are configured and shows the
              status/health of each.
            </p>
            {Utilities.sessionRolesHasOneOf([rbacRoles.CLUSTER_ADMIN]) && (
              <button
                className="btn primary tw-ml-auto tw-w-fit tw-h-fit"
                onClick={onAddNodeClick}
              >
                Add node type
              </button>
            )}
          </div>
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
          {fromLicenseFlow && (
            <button
              className="btn primary tw-w-fit tw-ml-auto"
              onClick={handleContinue}
            >
              Continue
            </button>
          )}
        </div>
      </div>
      {/* MODALS */}
      <Modal
        isOpen={state.displayAddNode}
        onRequestClose={() => setState({ displayAddNode: false })}
        contentLabel="Add Node"
        className="Modal"
        ariaHideApp={false}
      >
        <div className="Modal-body tw-flex tw-flex-col tw-gap-4">
          <div className="tw-flex">
            <h1 className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--more">
              Add a Node Type
            </h1>
            <Icon
              icon="close"
              size={14}
              className="tw-ml-auto gray-color clickable close-icon"
              onClick={() => setState({ displayAddNode: false })}
            />
          </div>
          <p className="tw-text-base">
            To add a node type to this cluster, select the type of node you are
            adding, and then select an installation method below. This screen
            will automatically show the status when the node successfully joins
            the cluster.
          </p>
          <div className="tw-grid tw-gap-2 tw-grid-cols-4 tw-auto-rows-auto">
            {NODE_TYPES.map((nodeType) => (
              <div
                key={nodeType}
                className={classNames("BoxedCheckbox", {
                  "is-active": selectedNodeTypes.includes(nodeType),
                  "is-disabled": determineDisabledState(
                    nodeType,
                    selectedNodeTypes
                  ),
                })}
              >
                <input
                  id={`${nodeType}NodeType`}
                  className="u-cursor--pointer hidden-input"
                  type="checkbox"
                  name={`${nodeType}NodeType`}
                  value={nodeType}
                  disabled={determineDisabledState(nodeType, selectedNodeTypes)}
                  checked={selectedNodeTypes.includes(nodeType)}
                  onChange={handleSelectNodeType}
                />
                <label
                  htmlFor={`${nodeType}NodeType`}
                  className="tw-block u-cursor--pointer u-userSelect--none u-textColor--primary u-fontSize--normal u-fontWeight--medium tw-text-center"
                >
                  {nodeType}
                </label>
              </div>
            ))}
          </div>
          <div>
            <CodeSnippet
              key={selectedNodeTypes.toString()}
              language="bash"
              canCopy={true}
              onCopyText={<span className="u-textColor--success">Copied!</span>}
            >
              {`curl ${generatePrimaryAddNodeCommand}?token=abc&labels=${selectedNodeTypes.join(
                ","
              )}`}
            </CodeSnippet>
          </div>
          <div className="tw-flex tw-items-center tw-gap-1.5">
            <input
              id="useStaticToken"
              type="checkbox"
              checked={useStaticToken}
              onChange={(e) => setUseStaticToken(e.target.checked)}
            />
            <label
              htmlFor="useStaticToken"
              className="tw-text-base tw-text-gray-700"
            >
              Use a static token (useful for ASGs and scripts)
            </label>
          </div>
          {/* buttons */}
          <div className="tw-w-full tw-flex tw-justify-end tw-gap-2">
            <button
              className="btn secondary large"
              onClick={() => setState({ displayAddNode: false })}
            >
              Close
            </button>
            <button
              className="btn primary large"
              disabled={selectedNodeTypes.length === 0}
              onClick={() => setState({ displayAddNode: false })}
            >
              Add node type
            </button>
          </div>
        </div>
      </Modal>
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
