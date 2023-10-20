import { MenuItem } from "@mui/material";
import { useQuery } from "@tanstack/react-query";
import MaterialReactTable, { MRT_ColumnDef } from "material-react-table";
import { useMemo, useReducer } from "react";
import { Link, useParams } from "react-router-dom";

import { KotsPageTitle } from "@components/Head";
import { useApps } from "@features/App";
import { rbacRoles } from "../../constants/rbac";
import { Utilities } from "../../utilities/utilities";
import AddNodeModal from "./AddNodeModal";

import "@src/scss/components/apps/HelmVMClusterManagement.scss";

const testData = {
  nodes: undefined,
};
// const testData = {
//   nodes: [
//     {
//       name: "laverya-helmvm",
//       isConnected: true,
//       isReady: true,
//       isPrimaryNode: true,
//       canDelete: false,
//       kubeletVersion: "v1.28.2+k0s",
//       kubeProxyVersion: "v1.28.2+k0s",
//       operatingSystem: "linux",
//       kernelVersion: "5.10.0-26-cloud-amd64",
//       cpu: { capacity: 4, used: 1.9364847660000002 },
//       memory: { capacity: 15.633056640625, used: 3.088226318359375 },
//       pods: { capacity: 110, used: 27 },
//       labels: ["controller"],
//       conditions: {
//         memoryPressure: false,
//         diskPressure: false,
//         pidPressure: false,
//         ready: true,
//       },
//       podList: [],
//     },
//   ],
//   ha: true,
//   isHelmVMEnabled: true,
// };

type State = {
  displayAddNodeModal: boolean;
  confirmDeleteNode: string;
  deleteNodeError: string;
  showConfirmDrainModal: boolean;
  nodeNameToDrain: string;
  drainingNodeName: string | null;
  drainNodeSuccessful: boolean;
};

const HelmVMClusterManagement = ({
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
      displayAddNodeModal: false,
      confirmDeleteNode: "",
      deleteNodeError: "",
      showConfirmDrainModal: false,
      nodeNameToDrain: "",
      drainingNodeName: null,
      drainNodeSuccessful: false,
    }
  );

  const { data: appsData } = useApps();
  // we grab the first app because helmvm users should only ever have one app
  const app = appsData?.apps?.[0];

  const { slug } = useParams();

  // #region queries
  type NodesResponse = {
    ha: boolean;
    isHelmVMEnabled: boolean;
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
    queryKey: ["helmVmNodes"],
    queryFn: async () => {
      const res = await fetch(`${process.env.API_ENDPOINT}/helmvm/nodes`, {
        headers: {
          Accept: "application/json",
        },
        credentials: "include",
        method: "GET",
      });
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
  // #endregion

  const onAddNodeClick = () => {
    setState({
      displayAddNodeModal: true,
    });
  };

  // #region table logic
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

  const handleCloseModal = () => {
    setState({
      displayAddNodeModal: false,
    });
  };

  return (
    <div className="HelmVMClusterManagement--wrapper container u-overflow--auto u-paddingTop--50 tw-font-sans">
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
                columnPinning: { left: ["mrt-row-actions", "name"] },
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
              enableRowActions
              renderRowActionMenuItems={({ closeMenu, row }) => [
                <MenuItem
                  key="edit"
                  onClick={() => {
                    console.info("Edit");
                    closeMenu();
                  }}
                >
                  Edit
                </MenuItem>,
                <MenuItem
                  key="delete"
                  onClick={() => {
                    console.info("Delete");
                    closeMenu();
                  }}
                >
                  Delete
                </MenuItem>,
              ]}
              displayColumnDefOptions={{
                "mrt-row-actions": {
                  size: 36,
                },
              }}
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
      <AddNodeModal
        showModal={state.displayAddNodeModal}
        handleCloseModal={handleCloseModal}
      />
    </div>
  );
};

export default HelmVMClusterManagement;
