import { useQuery } from "@tanstack/react-query";
import classNames from "classnames";
import MaterialReactTable, { MRT_ColumnDef } from "material-react-table";
import { ChangeEvent, useMemo, useReducer, useState } from "react";
import Modal from "react-modal";
import { Link, useParams } from "react-router-dom";

import { KotsPageTitle } from "@components/Head";
import { useApps } from "@features/App";
import { rbacRoles } from "../../constants/rbac";
import { Utilities } from "../../utilities/utilities";
import Icon from "../Icon";
import CodeSnippet from "../shared/CodeSnippet";

import "@src/scss/components/apps/EmbeddedClusterManagement.scss";

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
}: {
  fromLicenseFlow?: boolean;
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

  const { data: appsData } = useApps();
  // we grab the first app because embeddedcluster users should only ever have one app
  const app = appsData?.apps?.[0];

  const { slug } = useParams();

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

  const determineDisabledState = () => {
    return false;
  };

  const handleSelectNodeType = (e: ChangeEvent<HTMLInputElement>) => {
    let nodeType = e.currentTarget.value;
    let types = selectedNodeTypes;

    if (selectedNodeTypes.includes(nodeType)) {
      setSelectedNodeTypes(types.filter((type) => type !== nodeType));
    } else {
      setSelectedNodeTypes([...types, nodeType]);
    }
  };
  // #endregion

  type NodeColumns = {
    name: string | JSX.Element;
    roles: JSX.Element;
    status: string;
    cpu: string;
    memory: string;
    pause: JSX.Element;
    delete: JSX.Element;
  };

  const columns = useMemo<MRT_ColumnDef<NodeColumns>[]>(
    () => [
      {
        accessorKey: "name",
        header: "Name",
        enableHiding: false,
        enableColumnDragging: false,
        size: 150,
      },
      {
        accessorKey: "roles",
        header: "Role(s)",
        size: 150,
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
      // {
      //   accessorKey: "pause",
      //   header: "Pause",
      //   size: 100,
      // },
      // {
      //   accessorKey: "delete",
      //   header: "Delete",
      //   size: 80,
      // },
    ],
    []
  );

  const mappedNodes = useMemo(() => {
    return (
      (nodesData?.nodes || testData?.nodes)?.map((n) => ({
        name: (
          <Link
            to={slug ? `/${slug}/cluster/${n.name}` : `/cluster/${n.name}`}
            className="tw-font-semibold tw-text-blue-300 hover:tw-underline"
          >
            {n.name}
          </Link>
        ),
        roles: (
          <div className="tw-w-full tw-flex tw-flex-wrap tw-gap-1">
            {n?.labels?.map((l) => (
              <span
                key={l}
                className="tw-font-semibold tw-text-xs tw-px-1 tw-rounded-sm tw-border tw-border-solid tw-bg-white tw-border-gray-100"
              >
                {l}
              </span>
            ))}
          </div>
        ),
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
  }, [nodesData?.nodes?.toString()]);
  // #endregion

  return (
    <div className="EmbeddedClusterManagement--wrapper container u-overflow--auto u-paddingTop--50 tw-font-sans">
      <KotsPageTitle pageName="Cluster Management" />
      <div className="flex1 tw-mb-10 tw-flex tw-flex-col tw-gap-4 card-bg">
        <p className="flex-auto u-fontSize--larger u-fontWeight--bold u-textColor--primary">
          Cluster Nodes
        </p>
        <div className="tw-flex tw-gap-6 tw-items-center">
          <p className="tw-text-base tw-flex-1 tw-text-gray-600">
            This page lists the nodes that are configured and shows the
            status/health of each.
          </p>
          {Utilities.sessionRolesHasOneOf([rbacRoles.CLUSTER_ADMIN]) && (
            <button
              className="btn primary tw-ml-auto tw-w-fit tw-h-fit"
              onClick={onAddNodeClick}
            >
              Add node
            </button>
          )}
        </div>
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
          <Link
            className="btn primary tw-w-fit tw-ml-auto"
            to={
              app?.isConfigurable ? `/${app?.slug}/config` : `/app/${app?.slug}`
            }
          >
            Continue
          </Link>
        )}
      </div>
      {/* MODALS */}
      <Modal
        isOpen={state.displayAddNode}
        onRequestClose={() => setState({ displayAddNode: false })}
        contentLabel="Add Node"
        className="Modal"
        ariaHideApp={false}
      >
        <div className="Modal-body tw-flex tw-flex-col tw-gap-4 tw-font-sans">
          <div className="tw-flex">
            <h1 className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--more">
              Add a Node
            </h1>
            <Icon
              icon="close"
              size={14}
              className="tw-ml-auto gray-color clickable close-icon"
              onClick={() => setState({ displayAddNode: false })}
            />
          </div>
          <p className="tw-text-base tw-text-gray-600">
            To add a node to this cluster, select the type of node you'd like to
            add. Once you've selected a node type, we will generate a node join
            command for you to use in the CLI. When the node successfully joins
            the cluster, you will see it appear in the list of nodes on this
            page.
          </p>
          <div className="tw-grid tw-gap-2 tw-grid-cols-4 tw-auto-rows-auto">
            {rolesLoading && (
              <p className="tw-text-base tw-w-full tw-text-center tw-py-4 tw-text-gray-500 tw-font-semibold">
                Loading roles...
              </p>
            )}
            {!rolesData && rolesError && (
              <p className="tw-text-base tw-w-full tw-text-center tw-py-4 tw-text-pink-500 tw-font-semibold">
                {rolesError?.message}
              </p>
            )}
            {(rolesData?.roles || NODE_TYPES).map((nodeType) => (
              <div
                key={nodeType}
                className={classNames("BoxedCheckbox", {
                  "is-active": selectedNodeTypes.includes(nodeType),
                  "is-disabled": determineDisabledState(),
                })}
              >
                <input
                  id={`${nodeType}NodeType`}
                  className="u-cursor--pointer hidden-input"
                  type="checkbox"
                  name={`${nodeType}NodeType`}
                  value={nodeType}
                  disabled={determineDisabledState()}
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
                <p className="tw-text-sm tw-text-gray-500 tw-font-semibold tw-mt-2">
                  Command expires: {generateAddNodeCommand?.expiry}
                </p>
              </>
            )}
          </div>
          {/* buttons */}
          <div className="tw-w-full tw-flex tw-justify-end tw-gap-2">
            <button
              className="btn secondary large"
              onClick={() => setState({ displayAddNode: false })}
            >
              Close
            </button>
          </div>
        </div>
      </Modal>
    </div>
  );
};

export default EmbeddedClusterManagement;
