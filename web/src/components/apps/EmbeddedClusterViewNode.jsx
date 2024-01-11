import { MaterialReactTable } from "material-react-table";
import { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";
import Loader from "@components/shared/Loader";

const testData = undefined;

const EmbeddedClusterViewNode = () => {
  const { slug, nodeName } = useParams();
  const { data: nodeData, isLoading: nodeLoading } = useQuery({
    queryKey: ["embeddedClusterNode", nodeName],
    queryFn: async ({ queryKey }) => {
      const [, nodeName] = queryKey;
      return (
        await fetch(
          `${process.env.API_ENDPOINT}/embedded-cluster/node/${nodeName}`,
          {
            headers: {
              Accept: "application/json",
            },
            credentials: "include",
            method: "GET",
          }
        )
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
  });

  const node = nodeData || testData;

  // #region table data
  const columns = useMemo(
    () => [
      {
        accessorKey: "name",
        header: "Name",
        enableHiding: false,
        enableColumnDragging: false,
        size: 150,
      },
      {
        accessorKey: "namespace",
        header: "Namespace",
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
      //   accessorKey: "delete",
      //   header: "Delete",
      //   size: 80,
      // },
    ],
    []
  );

  const mappedPods = useMemo(() => {
    return node?.podList?.map((p) => ({
      name: p.name,
      namespace: p.namespace,
      status: p.status,
      cpu: p.cpu,
      memory: p.memory,
      delete: (
        <>
          <button className="btn red primary">Delete</button>
        </>
      ),
    }));
  }, [node?.podList?.toString()]);
  // #endregion

  return (
    <div className="container u-paddingTop--50 tw-min-h-full tw-box-border tw-mb-10 tw-pb-6 tw-flex tw-flex-col tw-gap-6 tw-font-sans">
      {/* Breadcrumbs */}
      <p className="tw-text-sm tw-text-gray-400">
        <Link
          to={slug ? `/${slug}/cluster/manage` : `/cluster/manage`}
          className="!tw-text-blue-300 tw-font-semibold hover:tw-underline"
        >
          Cluster Nodes
        </Link>{" "}
        / {nodeName}
      </p>

      {nodeLoading && (
        <div className="tw-w-full tw-h-full tw-flex tw-justify-center tw-items-center">
          <Loader size="70" />
        </div>
      )}
      {!nodeLoading && node && (
        <>
          {/* Node Info */}
          <div className="tw-flex tw-flex-col tw-gap-2 tw-p-3 card-bg">
            <p className="tw-font-semibold tw-text-xl tw-text-gray-800">
              {node?.name}
            </p>
            <div className="tw-flex tw-flex-col tw-text-sm tw-gap-2 card-item">
              <div className="tw-flex tw-gap-2">
                <p className="tw-text-gray-800 tw-font-semibold">
                  kubelet version
                </p>
                <p className="tw-text-gray-400">{node?.kubeletVersion}</p>
              </div>
              <div className="tw-flex tw-gap-2">
                <p className="tw-text-gray-800 tw-font-semibold">
                  kube-proxy version
                </p>
                <p className="tw-text-gray-400">{node?.kubeProxyVersion}</p>
              </div>
              <div className="tw-flex tw-gap-2">
                <p className="tw-text-gray-800 tw-font-semibold">
                  kernel version
                </p>
                <p className="tw-text-gray-400">{node?.kernelVersion}</p>
              </div>
            </div>
          </div>
          {/* Pods table */}
          <div className="card-bg tw-p-3 tw-flex tw-flex-col tw-gap-2">
            <p className="tw-font-semibold tw-text-xl tw-text-gray-800">Pods</p>
            <div className="card-item">
              <MaterialReactTable
                columns={columns}
                data={mappedPods}
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
            </div>
          </div>
          {/* Troubleshooting */}
          {/* <div className="card-bg tw-p-3">
            <p className="tw-font-semibold tw-text-xl tw-text-gray-800">
              Troubleshooting
            </p>
          </div> */}
          {/* Danger Zone */}
          {/* <div className="card-bg tw-p-3 tw-flex tw-flex-col tw-gap-3">
            <p className="tw-font-semibold tw-text-xl tw-text-gray-800">
              Danger Zone
            </p>
            <button className="btn red primary tw-w-fit">
              Prepare node for delete
            </button>
          </div> */}
        </>
      )}
    </div>
  );
};

export default EmbeddedClusterViewNode;
