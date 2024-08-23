import { useQuery } from "@tanstack/react-query";
import classNames from "classnames";
import MaterialReactTable, { MRT_ColumnDef } from "material-react-table";
import { useEffect, useMemo, useReducer, useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";

import { KotsPageTitle } from "@components/Head";
import { useApps } from "@features/App";
import { rbacRoles } from "../../constants/rbac";
import { Utilities } from "../../utilities/utilities";
import CodeSnippet from "../shared/CodeSnippet";

import "@src/scss/components/apps/EmbeddedClusterManagement.scss";
import { isEqual } from "lodash";
import AddANodeModal from "@components/modals/AddANodeModal";
import Icon from "@components/Icon";

const testData = {
  nodes: undefined,
};

type State = {
  displayAddNode: boolean;
  confirmDeleteNode: string;
  deleteNodeError: string;
  showConfirmDrainModal: boolean;
  nodeNameToDrain: string;
  drainingNodeName: string | null;
  drainNodeSuccessful: boolean;
};

const EmbeddedClusterManagement = ({
  fromLicenseFlow = false,
  isEmbeddedClusterWaitingForNodes = false,
}: {
  fromLicenseFlow?: boolean;
  isEmbeddedClusterWaitingForNodes?: boolean;
}) => {
  const [state, setState] = useReducer(
    (prevState: State, newState: Partial<State>) => ({
      ...prevState,
      ...newState,
    }),
    {
      displayAddNode: false,
      confirmDeleteNode: "",
      deleteNodeError: "",
      showConfirmDrainModal: false,
      nodeNameToDrain: "",
      drainingNodeName: null,
      drainNodeSuccessful: false,
    }
  );
  const [selectedNodeTypes, setSelectedNodeTypes] = useState<string[]>([]);

  const { data: appsData, refetch: refetchApps } = useApps();
  // we grab the first app because embeddedcluster users should only ever have one app
  const app = appsData?.apps?.[0];

  const { slug } = useParams();

  const navigate = useNavigate();

  // #region queries
  type NodesResponse = {
    ha: boolean;
    isEmbeddedClusterEnabled: boolean;
    nodes: {
      name: string;
      isConnected: boolean;
      isReady: boolean;
      isPrimaryNode: boolean;
      canDelete: boolean;
      kubeletVersion: string;
      cpu: {
        capacity: number;
        used: number;
      };
      memory: {
        capacity: number;
        used: number;
      };
      pods: {
        capacity: number;
        used: number;
      };
      labels?: string[];
      conditions: {
        memoryPressure: boolean;
        diskPressure: boolean;
        pidPressure: boolean;
        ready: boolean;
      };
    }[];
  };

  const {
    data: nodesData,
    isInitialLoading: nodesLoading,
    error: nodesError,
  } = useQuery<NodesResponse, Error, NodesResponse>({
    queryKey: ["embeddedClusterNodes"],
    queryFn: async () => {
      const res = await fetch(
        `${process.env.API_ENDPOINT}/embedded-cluster/nodes`,
        {
          headers: {
            Accept: "application/json",
          },
          credentials: "include",
          method: "GET",
        }
      );
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
        }
        console.log(
          "failed to get node status list, unexpected status code",
          res.status
        );
        try {
          const error = await res.json();
          throw new Error(
            error?.error?.message || error?.error || error?.message
          );
        } catch (err) {
          throw new Error("Unable to fetch nodes, please try again later.");
        }
      }
      return res.json();
    },
    refetchInterval: (data) => (data ? 1000 : 0),
    retry: false,
  });

  type RolesResponse = {
    roles: string[];
  };

  const {
    data: rolesData,
    isInitialLoading: rolesLoading,
    error: rolesError,
  } = useQuery<RolesResponse, Error, RolesResponse>({
    queryKey: ["embeddedClusterRoles"],
    queryFn: async () => {
      const res = await fetch(
        `${process.env.API_ENDPOINT}/embedded-cluster/roles`,
        {
          headers: {
            Accept: "application/json",
          },
          credentials: "include",
          method: "GET",
        }
      );
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
        }
        console.log(
          "failed to get role list, unexpected status code",
          res.status
        );
        try {
          const error = await res.json();
          throw new Error(
            error?.error?.message || error?.error || error?.message
          );
        } catch (err) {
          throw new Error("Unable to fetch roles, please try again later.");
        }
      }
      return res.json();
    },
    retry: false,
  });

  type AddNodeCommandResponse = {
    command: string;
    expiry: string;
  };

  const {
    data: generateAddNodeCommand,
    isLoading: generateAddNodeCommandLoading,
    error: generateAddNodeCommandError,
  } = useQuery<AddNodeCommandResponse, Error, AddNodeCommandResponse>({
    queryKey: ["generateAddNodeCommand", selectedNodeTypes],
    queryFn: async ({ queryKey }) => {
      const [, nodeTypes] = queryKey;
      const res = await fetch(
        `${process.env.API_ENDPOINT}/embedded-cluster/generate-node-join-command`,
        {
          headers: {
            "Content-Type": "application/json",
            Accept: "application/json",
          },
          credentials: "include",
          method: "POST",
          body: JSON.stringify({
            roles: nodeTypes,
          }),
        }
      );
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
        }
        console.log(
          "failed to get generate node command, unexpected status code",
          res.status
        );
        try {
          const error = await res.json();
          throw new Error(
            error?.error?.message || error?.error || error?.message
          );
        } catch (err) {
          throw new Error(
            "Unable to generate node join command, please try again later."
          );
        }
      }
      return res.json();
    },
    enabled: selectedNodeTypes.length > 0,
  });
  // #endregion

  const onAddNodeClick = () => {
    setState({
      displayAddNode: true,
    });
  };

  // #region node type logic
  const NODE_TYPES = ["controller"];

  useEffect(() => {
    const nodeTypes = rolesData?.roles || NODE_TYPES;
    if (nodeTypes.length === 1) {
      // if there's only one node type, select it by default
      setSelectedNodeTypes(nodeTypes);
    } else if (nodeTypes.length > 1) {
      setSelectedNodeTypes([nodeTypes[0]]);
    }
  }, [rolesData]);

  const determineDisabledState = () => {
    return false;
  };

  const handleSelectNodeType = (nodeType) => {
    setSelectedNodeTypes((prevSelectedNodeTypes) => {
      if (prevSelectedNodeTypes.includes(nodeType)) {
        return prevSelectedNodeTypes.filter((type) => type !== nodeType);
      } else {
        return [...prevSelectedNodeTypes, nodeType];
      }
    });
  };
  // #endregion

  type NodeColumns = {
    name: string;
    roles: string;
    status: string;
    cpu: string;
    memory: string;
  };

  const columns = useMemo<MRT_ColumnDef<NodeColumns>[]>(
    () => [
      {
        accessorKey: "name",
        header: "Name",
        enableHiding: false,
        enableColumnDragging: false,
        size: 150,
        Cell: ({ cell }) => {
          const value = cell.getValue<string>();
          return (
            <Link
              to={slug ? `/${slug}/cluster/${value}` : `/cluster/${value}`}
              className="tw-font-semibold tw-text-blue-300 hover:tw-underline"
            >
              {value}
            </Link>
          );
        },
      },
      {
        accessorKey: "roles",
        header: "Role(s)",
        size: 150,
        Cell: ({ cell }) => {
          const value = cell.getValue<string>();
          if (!value) {
            return "";
          }
          return (
            <div className="tw-w-full tw-flex tw-flex-wrap tw-gap-1">
              {value.split(" ").map((l) => (
                <span
                  key={l}
                  className="tw-font-semibold tw-text-xs tw-px-1 tw-rounded-sm tw-border tw-border-solid tw-bg-white tw-border-gray-100"
                >
                  {l}
                </span>
              ))}
            </div>
          );
        },
      },
      {
        accessorKey: "status",
        header: "Status",
        size: 150,
      },
      {
        accessorKey: "cpu",
        header: "CPU",
        size: 150,
        muiTableBodyCellProps: {
          align: "right",
        },
      },
      {
        accessorKey: "memory",
        header: "Memory",
        size: 150,
        muiTableBodyCellProps: {
          align: "right",
        },
      },
    ],
    []
  );
  const hasNodesChanged = (
    prevNodes: NodesResponse,
    currentNodes: NodesResponse
  ) => {
    return !isEqual(prevNodes, currentNodes);
  };

  const mappedNodes = useMemo(() => {
    return (
      (nodesData?.nodes || testData?.nodes)?.map((n) => ({
        name: n.name,
        roles: n.labels?.join(" ") || "",
        status: n.isReady ? "Ready" : "Not Ready",
        cpu: `${n.cpu.used.toFixed(2)} / ${n.cpu.capacity.toFixed(2)}`,
        memory: `${n.memory.used.toFixed(2)} / ${n.memory.capacity.toFixed(
          2
        )} GB`,
        pause: (
          <>
            <button className="btn secondary">Pause</button>
          </>
        ),
        delete: (
          <>
            <button className="btn red primary">Delete</button>
          </>
        ),
      })) || []
    );
  }, [nodesData?.nodes?.toString(), hasNodesChanged]);
  // #endregion

  const onContinueClick = async () => {
    const res = await fetch(
      `${process.env.API_ENDPOINT}/embedded-cluster/management`,
      {
        headers: {
          Accept: "application/json",
        },
        credentials: "include",
        method: "POST",
      }
    );
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
      }
      console.log(
        "failed to confirm cluster management, unexpected status code",
        res.status
      );
      try {
        const error = await res.json();
        throw new Error(
          error?.error?.message || error?.error || error?.message
        );
      } catch (err) {
        throw new Error(
          "Unable to confirm cluster management, please try again later."
        );
      }
    }

    await refetchApps();

    const data = await res.json();

    if (data.versionStatus === "pending_config") {
      navigate(`/${app?.slug}/config`);
    } else if (data.versionStatus === "pending_preflight") {
      navigate(`/${app?.slug}/preflight`);
    } else {
      navigate(`/app/${app?.slug}`);
    }
  };

  const AddNodeInstructions = () => {
    return (
      <div className="tw-mb-2 tw-text-base">
        {Utilities.isInitialAppInstall(app) && (
          <p>
            Optionally add nodes to the cluster. Click{" "}
            <span className="tw-font-semibold">Continue </span>
            to proceed with a single node.
          </p>
        )}
        <p>
          {rolesData?.roles &&
            rolesData.roles.length > 1 &&
            "Select one or more roles to assign to the new node."}{" "}
          Copy the join command and run it on the machine you'd like to join to
          the cluster.
        </p>
      </div>
    );
  };

  const AddNodeCommands = () => {
    return (
      <>
        {rolesLoading && (
          <p className="tw-text-base tw-w-full tw-text-center tw-py-4 tw-text-gray-500 tw-font-semibold">
            Loading roles...
          </p>
        )}
        {!rolesData && rolesError && (
          <p className="tw-text-base tw-w-full tw-text-center tw-py-4 tw-text-pink-500 tw-font-semibold">
            {rolesError?.message || "Unable to fetch roles"}
          </p>
        )}

        {rolesData?.roles && rolesData.roles.length > 1 && (
          <div className="tw-flex tw-gap-2 tw-items-center tw-mt-2">
            <p className="tw-text-gray-600 tw-font-semibold">Roles: </p>
            {rolesData.roles.map((nodeType) => (
              <div
                key={nodeType}
                className={classNames(
                  "nodeType-selector tw-border-[1px] tw-border-solid tw-border-[#326DE6] tw-rounded tw-px-2 tw-py-2 tw-flex tw-items-center  tw-cursor-pointer",
                  {
                    "tw-text-white tw-bg-[#326DE6]":
                      selectedNodeTypes.includes(nodeType),
                    "is-disabled": determineDisabledState(),
                    "tw-text-[#326DE6] tw-bg-white  tw-hover:tw-bg-[#f8fafe]":
                      !selectedNodeTypes.includes(nodeType),
                  }
                )}
                onClick={() => {
                  handleSelectNodeType(nodeType);
                }}
              >
                <label
                  htmlFor={`${nodeType}NodeType`}
                  className=" u-userSelect--none tw-text-gray-600 u-fontSize--normal u-fontWeight--medium tw-text-center tw-flex tw-items-center"
                >
                  {selectedNodeTypes.includes(nodeType) && (
                    <Icon icon="check" size={12} className="tw-mr-2" />
                  )}
                  <input
                    id={`${nodeType}NodeType`}
                    className="u-cursor--pointer tw-mr-2 hidden-input"
                    type="checkbox"
                    name={`${nodeType}NodeType`}
                    value={nodeType}
                    disabled={determineDisabledState()}
                    checked={selectedNodeTypes.includes(nodeType)}
                  />
                </label>
                {nodeType}
              </div>
            ))}
          </div>
        )}
        <div className="tw-max-w-[700px]">
          {selectedNodeTypes.length > 0 && generateAddNodeCommandLoading && (
            <p className="tw-text-base tw-w-full tw-text-center tw-py-4 tw-text-gray-500 tw-font-semibold">
              Generating command...
            </p>
          )}
          {!generateAddNodeCommand && generateAddNodeCommandError && (
            <p className="tw-text-base tw-w-full tw-text-center tw-py-4 tw-text-pink-500 tw-font-semibold">
              {generateAddNodeCommandError?.message}
            </p>
          )}
          {!generateAddNodeCommandLoading && generateAddNodeCommand?.command && (
            <>
              <CodeSnippet
                key={selectedNodeTypes.toString()}
                language="bash"
                canCopy={true}
                onCopyText={
                  <span className="u-textColor--success">Copied!</span>
                }
              >
                {generateAddNodeCommand?.command}
              </CodeSnippet>
            </>
          )}
        </div>
      </>
    );
  };
  return (
    <div className="EmbeddedClusterManagement--wrapper container u-overflow--auto u-paddingTop--50 tw-font-sans">
      <KotsPageTitle pageName="Cluster Management" />
      <div className="flex1 tw-mb-10 tw-flex tw-flex-col tw-gap-4 card-bg">
        <p className="flex-auto u-fontSize--larger u-fontWeight--bold u-textColor--primary">
          Nodes
        </p>
        <div className="tw-flex tw-gap-6 tw-items-center">
          {" "}
          {(!Utilities.isInitialAppInstall(app) ||
            !isEmbeddedClusterWaitingForNodes) && (
            <>
              <div className="tw-flex tw-gap-6">
                <p>
                  View the nodes in your cluster, generate commands to add nodes
                  to the cluster, and view workloads running on each node.
                </p>
              </div>
              {Utilities.sessionRolesHasOneOf([rbacRoles.CLUSTER_ADMIN]) && (
                <button
                  className="btn primary tw-ml-auto tw-w-fit tw-h-fit"
                  onClick={onAddNodeClick}
                >
                  Add node
                </button>
              )}
            </>
          )}
        </div>
        {(Utilities.isInitialAppInstall(app) ||
          isEmbeddedClusterWaitingForNodes) && (
          <div className="tw-mt-4 tw-flex tw-flex-col">
            <AddNodeInstructions />
            <AddNodeCommands />
          </div>
        )}

        <div className="flex1 u-overflow--auto card-item">
          {nodesLoading && (
            <p className="tw-text-base tw-w-full tw-text-center tw-py-4 tw-text-gray-500 tw-font-semibold">
              Loading nodes...
            </p>
          )}
          {!nodesData && nodesError && (
            <p className="tw-text-base tw-w-full tw-text-center tw-py-4 tw-text-pink-500 tw-font-semibold">
              {nodesError?.message}
            </p>
          )}
          {(nodesData?.nodes || testData?.nodes) && (
            <MaterialReactTable
              columns={columns}
              data={mappedNodes}
              state={{
                columnPinning: { left: ["name"] },
              }}
              enableColumnResizing
              enableColumnActions={false}
              enableColumnOrdering
              enableBottomToolbar={false}
              muiTableHeadProps={{
                sx: {
                  "& hr": {
                    width: "0",
                  },
                },
              }}
              muiTableHeadCellProps={{
                sx: {
                  borderRight: "2px solid #e0e0e0",
                },
              }}
              muiTableBodyProps={{
                sx: {
                  "& tr:nth-of-type(odd)": {
                    backgroundColor: "#f5f5f5",
                  },
                },
              }}
              muiTableBodyCellProps={{
                sx: {
                  borderRight: "2px solid #e0e0e0",
                },
              }}
              muiTablePaperProps={{
                sx: {
                  width: "100%",
                  boxShadow: "none",
                },
              }}
              initialState={{ density: "compact" }}
              enablePagination={false}
              enableColumnFilters={false}
            />
          )}
        </div>
        {fromLicenseFlow && (
          <button
            className="btn primary tw-w-fit tw-ml-auto"
            onClick={() => onContinueClick()}
          >
            Continue
          </button>
        )}
      </div>
      {/* MODALS */}
      <AddANodeModal
        displayAddNode={state.displayAddNode}
        toggleDisplayAddNode={() => setState({ displayAddNode: false })}
        rolesData={rolesData}
      >
        <AddNodeCommands />
      </AddANodeModal>
    </div>
  );
};

export default EmbeddedClusterManagement;
