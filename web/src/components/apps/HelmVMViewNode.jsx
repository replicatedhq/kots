import React, { useMemo } from "react";
import { useQuery } from "react-query";
import { Link } from "react-router-dom";

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

const HelmVMViewNode = () => {
  // const { data: nodes } = useQuery({
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
  //     retry: false,
  //   },
  // });

  const node = testData.nodes[0];

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
        <table className="tw-w-full">
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
