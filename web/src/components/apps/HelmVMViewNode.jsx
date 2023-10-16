import React, { useMemo } from "react";
import { useQuery } from "react-query";
import { Link } from "react-router-dom";

const HelmVMViewNode = () => {
  const { data: nodes } = useQuery({
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
      retry: false,
    },
  });

  const node = nodes.nodes[0];

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
        accessorKey: "isConnected",
        header: "Connection",
        size: 150,
      },
      {
        accessorKey: "kubeletVersion",
        header: "Kubelet Version",
        size: 170,
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
        accessorKey: "pods",
        header: "Pods",
        size: 150,
      },
      {
        accessorKey: "canDelete",
        header: "Delete Node",
        size: 150,
      },
    ],
    []
  );

  return (
    <div className="container u-paddingTop--50 tw-flex tw-flex-col tw-gap-6 tw-font-sans">
      {/* Breadcrumbs */}
      <p className="tw-text-sm tw-text-gray-400">
        <Link
          to="/cluster/manage"
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
        <table>
          <thead>
            <tr>
              {columns.map((col) => {
                return (
                  <th key={col.accessorKey}>
                    <p className="tw-font-semibold tw-text-gray-800 tw-px-2 tw-py-1.5">
                      {col.header}
                    </p>
                  </th>
                );
              })}
            </tr>
          </thead>
          <tbody>Some pods here</tbody>
        </table>
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
