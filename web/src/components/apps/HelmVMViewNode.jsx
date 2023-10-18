import { MaterialReactTable } from "material-react-table";
import React, { useMemo } from "react";
import { useQuery } from "react-query";
import { Link, useParams } from "react-router-dom";

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
      podList: [
        {
          metadata: {
            name: "example-es-85fc9df74-g9jbn",
            generateName: "example-es-85fc9df74-",
            namespace: "helmvm",
            uid: "1caba3fb-bd52-430a-9cff-0eb0939317fa",
            resourceVersion: "40284",
            creationTimestamp: "2023-10-17T16:22:37Z",
            labels: {
              app: "example",
              component: "es",
              "kots.io/app-slug": "laverya-minimal-kots",
              "kots.io/backup": "velero",
              "pod-template-hash": "85fc9df74",
            },
            annotations: {
              "cni.projectcalico.org/containerID":
                "c3fa12aad2ed6f726ecda31f7f94d1224c9f50a805a9efc67aaf4959e464434c",
              "cni.projectcalico.org/podIP": "10.244.45.141/32",
              "cni.projectcalico.org/podIPs": "10.244.45.141/32",
              "kots.io/app-slug": "laverya-minimal-kots",
            },
            ownerReferences: [
              {
                apiVersion: "apps/v1",
                kind: "ReplicaSet",
                name: "example-es-85fc9df74",
                uid: "b5008bca-1ad0-4107-8603-397fc3be74f8",
                controller: true,
                blockOwnerDeletion: true,
              },
            ],
          },
          spec: {
            volumes: [
              {
                name: "kube-api-access-fhfc4",
                projected: {
                  sources: [
                    {
                      serviceAccountToken: {
                        expirationSeconds: 3607,
                        path: "token",
                      },
                    },
                    {
                      configMap: {
                        name: "kube-root-ca.crt",
                        items: [{ key: "ca.crt", path: "ca.crt" }],
                      },
                    },
                    {
                      downwardAPI: {
                        items: [
                          {
                            path: "namespace",
                            fieldRef: {
                              apiVersion: "v1",
                              fieldPath: "metadata.namespace",
                            },
                          },
                        ],
                      },
                    },
                  ],
                  defaultMode: 420,
                },
              },
            ],
            containers: [
              {
                name: "es",
                image:
                  "docker.elastic.co/elasticsearch/elasticsearch-oss:6.8.21",
                envFrom: [{ configMapRef: { name: "example-config" } }],
                resources: {
                  limits: { cpu: "500m", memory: "256Mi" },
                  requests: { cpu: "50m", memory: "16Mi" },
                },
                volumeMounts: [
                  {
                    name: "kube-api-access-fhfc4",
                    readOnly: true,
                    mountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
                  },
                ],
                terminationMessagePath: "/dev/termination-log",
                terminationMessagePolicy: "File",
                imagePullPolicy: "IfNotPresent",
              },
            ],
            restartPolicy: "Always",
            terminationGracePeriodSeconds: 30,
            dnsPolicy: "ClusterFirst",
            serviceAccountName: "default",
            serviceAccount: "default",
            nodeName: "laverya-helmvm",
            securityContext: {},
            imagePullSecrets: [{ name: "laverya-minimal-kots-registry" }],
            schedulerName: "default-scheduler",
            tolerations: [
              {
                key: "node.kubernetes.io/not-ready",
                operator: "Exists",
                effect: "NoExecute",
                tolerationSeconds: 300,
              },
              {
                key: "node.kubernetes.io/unreachable",
                operator: "Exists",
                effect: "NoExecute",
                tolerationSeconds: 300,
              },
            ],
            priority: 0,
            enableServiceLinks: true,
            preemptionPolicy: "PreemptLowerPriority",
          },
          status: {
            phase: "Running",
            conditions: [
              {
                type: "Initialized",
                status: "True",
                lastProbeTime: null,
                lastTransitionTime: "2023-10-17T16:22:37Z",
              },
              {
                type: "Ready",
                status: "False",
                lastProbeTime: null,
                lastTransitionTime: "2023-10-17T19:55:16Z",
                reason: "ContainersNotReady",
                message: "containers with unready status: [es]",
              },
              {
                type: "ContainersReady",
                status: "False",
                lastProbeTime: null,
                lastTransitionTime: "2023-10-17T19:55:16Z",
                reason: "ContainersNotReady",
                message: "containers with unready status: [es]",
              },
              {
                type: "PodScheduled",
                status: "True",
                lastProbeTime: null,
                lastTransitionTime: "2023-10-17T16:22:37Z",
              },
            ],
            hostIP: "10.128.0.44",
            podIP: "10.244.45.141",
            podIPs: [{ ip: "10.244.45.141" }],
            startTime: "2023-10-17T16:22:37Z",
            containerStatuses: [
              {
                name: "es",
                state: {
                  waiting: {
                    reason: "CrashLoopBackOff",
                    message:
                      "back-off 5m0s restarting failed container=es pod=example-es-85fc9df74-g9jbn_helmvm(1caba3fb-bd52-430a-9cff-0eb0939317fa)",
                  },
                },
                lastState: {
                  terminated: {
                    exitCode: 137,
                    reason: "OOMKilled",
                    startedAt: "2023-10-17T19:55:11Z",
                    finishedAt: "2023-10-17T19:55:13Z",
                    containerID:
                      "containerd://9cce5c792b7ad61d040f7b8aca042d13a714100c75ebc40e71eb5444bbb65e83",
                  },
                },
                ready: false,
                restartCount: 46,
                image:
                  "docker.elastic.co/elasticsearch/elasticsearch-oss:6.8.21",
                imageID:
                  "docker.elastic.co/elasticsearch/elasticsearch-oss@sha256:86e7750c4d896d41bd638b6e510e0610b98fd9fa48f8caeeed8ccd8424b1dc9f",
                containerID:
                  "containerd://9cce5c792b7ad61d040f7b8aca042d13a714100c75ebc40e71eb5444bbb65e83",
                started: false,
              },
            ],
            qosClass: "Burstable",
          },
        },
      ],
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

  const node = nodeData || testData.nodes[0];

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
    return node.podList.map((n) => ({
      name: n.metadata.name,
      status: n.status.phase,
      disk: null,
      cpu: null,
      memory: null,
      canDelete: (
        <>
          <button className="btn red primary">Delete</button>
        </>
      ),
    }));
  }, [node.podList]);

  return (
    <div className="container u-paddingTop--50 tw-mb-10 tw-pb-6 tw-flex tw-flex-col tw-gap-6 tw-font-sans">
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
