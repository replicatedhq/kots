package hostnetwork

import v1 "k8s.io/api/core/v1"

type HostPortMap struct {
	MinioMinio int32
}

var (
	podnetPorts = HostPortMap{
		MinioMinio: 0,
	}
	hostnetPorts = HostPortMap{
		MinioMinio: 9000,
	}
)

// Return a port map with either all zeroes (do not set HostPort fields)
// or one with specific ports for each service we ship
func MaybeHostPortMap(useHostNetwork bool) HostPortMap {
	if useHostNetwork {
		return hostnetPorts
	}
	return podnetPorts
}

// Adds a NoSchedule toleration so that we can run kotsadm stack
// when the NoSchedule taint exists due to CNI being unready:
//
//   Type             Status  LastHeartbeatTime                 LastTransitionTime                Reason                       Message
//   ----             ------  -----------------                 ------------------                ------                       -------
//   Ready            False   Sat, 01 Feb 2020 23:56:38 +0000   Sat, 01 Feb 2020 17:51:05 +0000   KubeletNotReady              runtime network not ready: NetworkReady=false reason:NetworkPluginNotReady message:docker: network plugin is not ready: cni config uninitialized
//
// Above is from `kubectl describe node` after running a kURL
// installer with just K8s + docker (https://kurl.sh/10df2f5)
//
// apiVersion: "kurl.sh/v1beta1"
// kind: "Installer"
// metadata:
//   name: ""
// spec:
//   kubernetes:
//     version: "1.16.4"
//   docker:
//     version: "latest"
//
// toleration spec is borrowed from weave-net bootstrap YAML:
//
//           tolerations:
//            - effect: NoSchedule
//              operator: Exist
//
func Tolerations(useHostNetwork bool) []v1.Toleration {
	if !useHostNetwork {
		return nil
	}

	return []v1.Toleration{
		{
			Effect:   v1.TaintEffectNoSchedule,
			Key:      "node.kubernetes.io/not-ready",
			Operator: v1.TolerationOpExists,
		},
	}
}
