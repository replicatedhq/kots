import { MaterialReactTable } from "material-react-table";
import React, { useMemo } from "react";
import { useQuery } from "@tanstack/react-query";
import { Link, useParams } from "react-router-dom";

const HelmVMViewNode = () => {
  const { nodeName } = useParams();
  const { data: nodeData } = useQuery({
    queryKey: ["helmVmNode", nodeName],
    queryFn: async ({ queryKey }) => {
      const [, nodeName] = queryKey;
      return (
        await fetch(`${process.env.API_ENDPOINT}/helmvm/node/${nodeName}`, {
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
      retry: false,
    },
  });

  const node = nodeData;

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
        accessorKey: "status",
        header: "Status",
        size: 150,
      },
      {
        accessorKey: "disk",
        header: "Disk",
        size: 150,
      },
      {
        accessorKey: "cpu",
        header: "CPU",
        size: 150,
      },
      {
        accessorKey: "memory",
        header: "Memory",
        size: 150,
      },
      {
        accessorKey: "canDelete",
        header: "Delete Pod",
        size: 150,
      },
    ],
    []
  );

  const mappedPods = useMemo(() => {
    return node?.podList?.map((p) => ({
      name: p.metadata.name,
      status: p.status.phase,
      disk: null,
      cpu: null,
      memory: null,
      canDelete: (
        <>
          <button className="btn red primary">Delete</button>
        </>
      ),
    }));
  }, [node?.podList?.toString()]);
  // #endregion

  return (
    <div className="container u-paddingTop--50 tw-mb-10 tw-pb-6 tw-flex tw-flex-col tw-gap-6 tw-font-sans">
      {/* Breadcrumbs */}
      <p className="tw-text-sm tw-text-gray-400">
        <Link
          to={`/cluster/manage`}
          className="!tw-text-blue-300 tw-font-semibold hover:tw-underline"
        >
          Cluster Nodes
        </Link>{" "}
        / {node?.name}
      </p>
      {/* Node Info */}
      <div
        className="tw-flex tw-flex-col tw-gap-2 tw-bg-white tw-border tw-border-solid tw-border-gray-100 tw-rounded 
      tw-shadow-md tw-p-3"
      >
        <p className="tw-font-semibold tw-text-2xl tw-text-gray-800">
          Node Info
        </p>
        <div className="tw-flex tw-gap-2">
          <p className="tw-text-base tw-text-gray-800 tw-font-semibold">Name</p>
          <p className="tw-text-base tw-text-gray-400">{node?.name}</p>
        </div>
      </div>
      {/* Pods table */}
      <div
        className="tw-bg-white tw-border tw-border-solid tw-border-gray-100 tw-rounded 
      tw-shadow-md tw-p-3"
      >
        <p className="tw-font-semibold tw-text-2xl tw-text-gray-800">Pods</p>
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
      {/* Troubleshooting */}
      <div
        className="tw-bg-white tw-border tw-border-solid tw-border-gray-100 tw-rounded 
      tw-shadow-md tw-p-3"
      >
        <p className="tw-font-semibold tw-text-2xl tw-text-gray-800">
          Troubleshooting
        </p>
      </div>
      {/* Danger Zone */}
      <div
        className="tw-bg-white tw-border tw-border-solid tw-border-gray-100 tw-rounded 
      tw-shadow-md tw-p-3 tw-flex tw-flex-col tw-gap-3"
      >
        <p className="tw-font-semibold tw-text-2xl tw-text-gray-800">
          Danger Zone
        </p>
        <button className="btn red primary tw-w-fit">
          Prepare node for delete
        </button>
      </div>
    </div>
  );
};

export default HelmVMViewNode;
