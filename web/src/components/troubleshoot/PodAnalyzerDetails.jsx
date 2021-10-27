import * as React from "react";
import { withRouter } from "react-router-dom";
import AceEditor from "react-ace";
import Select from "react-select";

const POD_DEFINITION = `[
  {
    "metadata": {
      "name": "kotsadm-ff9f58888-gxwbc",
      "generateName": "kotsadm-ff9f58888-",
      "namespace": "default",
      "uid": "ff59f30f-5336-43d3-9748-2e8706bb8e2b",
      "resourceVersion": "37828",
      "creationTimestamp": "2021-10-04T21:23:15Z",
      "labels": {
        "app": "kotsadm",
        "app.kubernetes.io/managed-by": "skaffold",
        "app.kubernetes.io/name": "kotsadm",
        "kots.io/backup": "velero",
        "kots.io/kotsadm": "true",
        "pod-template-hash": "ff9f58888",
        "skaffold.dev/run-id": "bd888987-b230-4766-9686-f3a567d09721"
      },
      "annotations": {
        "backup.velero.io/backup-volumes": "backup",
        "pre.hook.backup.velero.io/command": "[\"/bin/bash\", \"-c\", \"PGPASSWORD=password pg_dump -U kotsadm -h kotsadm-postgres \u003e /backup/kotsadm-postgres.sql\"]",
        "pre.hook.backup.velero.io/timeout": "3m"
      },
      "ownerReferences": [
        {
          "apiVersion": "apps/v1",
          "kind": "ReplicaSet",
          "name": "kotsadm-ff9f58888",
          "uid": "49e99dd8-6462-40b5-97f2-95300aca1e6f",
          "controller": true,
          "blockOwnerDeletion": true
        }
      ],
      "managedFields": [
        {
          "manager": "k3s",
          "operation": "Update",
          "apiVersion": "v1",
          "time": "2021-10-04T21:23:20Z",
          "fieldsType": "FieldsV1",
          "fieldsV1": {
            "f:metadata": {
              "f:annotations": {
                ".": {},
                "f:backup.velero.io/backup-volumes": {},
                "f:pre.hook.backup.velero.io/command": {},
                "f:pre.hook.backup.velero.io/timeout": {}
              },
              "f:generateName": {},
              "f:labels": {
                ".": {},
                "f:app": {},
                "f:app.kubernetes.io/managed-by": {},
                "f:app.kubernetes.io/name": {},
                "f:kots.io/backup": {},
                "f:kots.io/kotsadm": {},
                "f:pod-template-hash": {},
                "f:skaffold.dev/run-id": {}
              },
              "f:ownerReferences": {
                ".": {},
                "k:{\"uid\":\"49e99dd8-6462-40b5-97f2-95300aca1e6f\"}": {
                  ".": {},
                  "f:apiVersion": {},
                  "f:blockOwnerDeletion": {},
                  "f:controller": {},
                  "f:kind": {},
                  "f:name": {},
                  "f:uid": {}
                }
              }
            },
            "f:spec": {
              "f:containers": {
                "k:{\"name\":\"kotsadm\"}": {
                  ".": {},
                  "f:env": {
                    ".": {},
                    "k:{\"name\":\"AIRGAP_UPLOAD_PARALLELISM\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"API_ADVERTISE_ENDPOINT\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"API_ENCRYPTION_KEY\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"API_ENDPOINT\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"AUTO_CREATE_CLUSTER\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"AUTO_CREATE_CLUSTER_NAME\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"AUTO_CREATE_CLUSTER_TOKEN\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"DEBUG\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"DEX_PGPASSWORD\"}": {
                      ".": {},
                      "f:name": {},
                      "f:valueFrom": {
                        ".": {},
                        "f:secretKeyRef": {
                          ".": {},
                          "f:key": {},
                          "f:name": {}
                        }
                      }
                    },
                    "k:{\"name\":\"DISABLE_SPA_SERVING\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"ENABLE_WEB_PROXY\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"KOTSADM_ENV\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"KOTSADM_LOG_LEVEL\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"KOTSADM_TARGET_NAMESPACE\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"KOTS_INSTALL_ID\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"KURL_PROXY_TLS_CERT_PATH\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"POD_NAMESPACE\"}": {
                      ".": {},
                      "f:name": {},
                      "f:valueFrom": {
                        ".": {},
                        "f:fieldRef": {
                          ".": {},
                          "f:apiVersion": {},
                          "f:fieldPath": {}
                        }
                      }
                    },
                    "k:{\"name\":\"POD_OWNER_KIND\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"POSTGRES_URI\"}": {
                      ".": {},
                      "f:name": {},
                      "f:valueFrom": {
                        ".": {},
                        "f:secretKeyRef": {
                          ".": {},
                          "f:key": {},
                          "f:name": {}
                        }
                      }
                    },
                    "k:{\"name\":\"REPLICATED_API_ENDPOINT\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"S3_ACCESS_KEY_ID\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"S3_BUCKET_ENDPOINT\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"S3_BUCKET_NAME\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"S3_ENDPOINT\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"S3_SECRET_ACCESS_KEY\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"SESSION_KEY\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    },
                    "k:{\"name\":\"SHARED_PASSWORD_BCRYPT\"}": {
                      ".": {},
                      "f:name": {},
                      "f:value": {}
                    }
                  },
                  "f:image": {},
                  "f:imagePullPolicy": {},
                  "f:name": {},
                  "f:ports": {
                    ".": {},
                    "k:{\"containerPort\":3000,\"protocol\":\"TCP\"}": {
                      ".": {},
                      "f:containerPort": {},
                      "f:name": {},
                      "f:protocol": {}
                    },
                    "k:{\"containerPort\":9229,\"protocol\":\"TCP\"}": {
                      ".": {},
                      "f:containerPort": {},
                      "f:name": {},
                      "f:protocol": {}
                    }
                  },
                  "f:resources": {
                    ".": {},
                    "f:limits": {
                      ".": {},
                      "f:cpu": {},
                      "f:memory": {}
                    },
                    "f:requests": {
                      ".": {},
                      "f:cpu": {},
                      "f:memory": {}
                    }
                  },
                  "f:terminationMessagePath": {},
                  "f:terminationMessagePolicy": {}
                }
              }
            }
          }
        }
      ]
    }
  }
]
`;
const POD_LOGS = `2021/10/04 21:42:06 kotsadm version v1.52.0
2021/10/04 21:42:06 Starting monitor loop
Starting Admin Console API on port 3000...
{"level":"error","ts":"2021-10-04T21:42:22Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:42:27Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:42:32Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:42:37Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:42:42Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:42:47Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:42:52Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:42:57Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:02Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:07Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:12Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:17Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:22Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:28Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:33Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:38Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:43Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:48Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:53Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:43:58Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:03Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:08Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:13Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:18Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:23Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:28Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:33Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:38Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:43Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:48Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:53Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:44:58Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:03Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:08Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:13Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:18Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:23Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:28Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:33Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:38Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:43Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:48Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:53Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:45:58Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:03Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:08Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:13Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:18Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:23Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:28Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:33Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:38Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:43Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:49Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:54Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
{"level":"error","ts":"2021-10-04T21:46:59Z","msg":"failed to get service to check status, namespace = discourse: services \"discourse-svc\" not found"}
`;
const POD_EVENTS = `[
  {
    "metadata": {
      "name": "kotsadm.16aaf0083f33c33f",
      "namespace": "default",
      "uid": "d7c0f9db-d20e-47f0-b45e-9d81aad50f68",
      "resourceVersion": "36964",
      "creationTimestamp": "2021-10-04T21:08:53Z",
      "managedFields": [
        {
          "manager": "k3s",
          "operation": "Update",
          "apiVersion": "v1",
          "time": "2021-10-04T21:08:53Z",
          "fieldsType": "FieldsV1",
          "fieldsV1": {
            "f:count": {},
            "f:firstTimestamp": {},
            "f:involvedObject": {
              "f:apiVersion": {},
              "f:kind": {},
              "f:name": {},
              "f:namespace": {},
              "f:resourceVersion": {},
              "f:uid": {}
            },
            "f:lastTimestamp": {},
            "f:message": {},
            "f:reason": {},
            "f:source": {
              "f:component": {}
            },
            "f:type": {}
          }
        }
      ]
    },
    "involvedObject": {
      "kind": "Deployment",
      "namespace": "default",
      "name": "kotsadm",
      "uid": "a2311d9b-67e4-4cb3-a93e-8f5c73914a38",
      "apiVersion": "apps/v1",
      "resourceVersion": "36961"
    },
    "reason": "ScalingReplicaSet",
    "message": "Scaled up replica set kotsadm-7f8c74b8db to 1",
    "source": {
      "component": "deployment-controller"
    },
    "firstTimestamp": "2021-10-04T21:08:53Z",
    "lastTimestamp": "2021-10-04T21:08:53Z",
    "count": 1,
    "type": "Normal",
    "eventTime": null,
    "reportingComponent": "",
    "reportingInstance": ""
  },
  {
    "metadata": {
      "name": "kotsadm-web.16aaf0083fa525a6",
      "namespace": "default",
      "uid": "eb2b8209-f9e8-4dc5-beaa-4d7c75f74631",
      "resourceVersion": "36967",
      "creationTimestamp": "2021-10-04T21:08:53Z",
      "managedFields": [
        {
          "manager": "k3s",
          "operation": "Update",
          "apiVersion": "v1",
          "time": "2021-10-04T21:08:53Z",
          "fieldsType": "FieldsV1",
          "fieldsV1": {
            "f:count": {},
            "f:firstTimestamp": {},
            "f:involvedObject": {
              "f:apiVersion": {},
              "f:kind": {},
              "f:name": {},
              "f:namespace": {},
              "f:resourceVersion": {},
              "f:uid": {}
            },
            "f:lastTimestamp": {},
            "f:message": {},
            "f:reason": {},
            "f:source": {
              "f:component": {}
            },
            "f:type": {}
          }
        }
      ]
    },
    "involvedObject": {
      "kind": "Deployment",
      "namespace": "default",
      "name": "kotsadm-web",
      "uid": "975a5412-eb21-4c69-92c8-119febc67849",
      "apiVersion": "apps/v1",
      "resourceVersion": "36963"
    },
    "reason": "ScalingReplicaSet",
    "message": "Scaled up replica set kotsadm-web-5cb5565c7d to 1",
    "source": {
      "component": "deployment-controller"
    },
    "firstTimestamp": "2021-10-04T21:08:53Z",
    "lastTimestamp": "2021-10-04T21:08:53Z",
    "count": 1,
    "type": "Normal",
    "eventTime": null,
    "reportingComponent": "",
    "reportingInstance": ""
  }
]
`;

export class PodAnalyzerDetails extends React.Component {

  state = {
    activeTab: "podDefinition",
    containersDropdownOptions: [
      { value: "container-1", name: "container-97361-hfg2s" },
      { value: "container-2", name: "container-12984-sdf23" }
    ],
    selectedAction: { value: "container-1", name: "container-97361-hfg2s" }
  }
  
  togglePodDetailView = (active) => {
    this.setState({ activeTab: active });
  }

  componentDidUpdate(lastProps, lastState) {
    if (this.state.activeTab !== lastState.activeTab && this.aceEditor) {
      this.aceEditor.editor.resize(true);
    }
  }

  onActionChange = (selectedOption) => {
    this.setState({ selectedAction: selectedOption });
  }

  renderPodDetailView = () => {
    switch (this.state.activeTab) {
    case "podDefinition":
      return (
        <div className="flex1 u-border--gray">
          <AceEditor
            ref={el => (this.aceEditor = el)}
            mode="json"
            theme="chrome"
            className="flex1 flex"
            readOnly={true}
            value={POD_DEFINITION}
            height="500px"
            width="100%"
            editorProps={{
              $blockScrolling: Infinity,
              useSoftTabs: true,
              tabSize: 2,
            }}
            setOptions={{
              scrollPastEnd: false,
              showGutter: true,
            }}
          />
        </div>
      )
    case "podLogs":
      return (
        <div>
          <p className="u-fontSize--normal u-fontWeight--medium u-textColor--header u-lineHeight--normal u-marginBottom--10">Which container logs would you like to view?</p>
          <div className="u-marginBottom--10 flex-auto">
            <Select
              className="replicated-select-container"
              classNamePrefix="replicated-select"
              options={this.state.containersDropdownOptions}
              getOptionLabel={(option) => option.name}
              getOptionValue={(option) => option.value}
              value={this.state.selectedAction}
              onChange={this.onActionChange}
              isOptionSelected={(option) => { option.value === this.state.selectedAction.value }}
            />
          </div>
          <div className="flex1 u-border--gray">
            <AceEditor
              ref={el => (this.aceEditor = el)}
              mode="text"
              theme="chrome"
              className="flex1 flex"
              readOnly={true}
              value={POD_LOGS}
              height="500px"
              width="100%"
              editorProps={{
                $blockScrolling: Infinity,
                useSoftTabs: true,
                tabSize: 2,
              }}
              setOptions={{
                scrollPastEnd: false,
                showGutter: true,
              }}
            />
          </div>
        </div>
      )
    case "podEvents":
      return (
        <div className="flex1 u-border--gray">
          <AceEditor
            ref={el => (this.aceEditor = el)}
            mode="json"
            theme="chrome"
            className="flex1 flex"
            readOnly={true}
            value={POD_EVENTS}
            height="500px"
            width="100%"
            editorProps={{
              $blockScrolling: Infinity,
              useSoftTabs: true,
              tabSize: 2,
            }}
            setOptions={{
              scrollPastEnd: false,
              showGutter: true,
            }}
          />
        </div>
      )
    default:
      return <div>nothing selected</div>
    }
  }

  render() {
    const { pod } = this.props;
    return (
        <div className="flex1 flex-column">
          <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--more">Details for {pod.primary}</p>
          <div className="SupportBundleTabs--wrapper flex-column flex1">
            <div className="flex tab-items">
              <span className={`${this.state.activeTab === "podDefinition" ? "is-active" : ""} tab-item blue`} onClick={() => this.togglePodDetailView("podDefinition")}>Pod definition</span>
              <span className={`${this.state.activeTab === "podLogs" ? "is-active" : ""} tab-item blue`} onClick={() => this.togglePodDetailView("podLogs")}>Pod logs</span>
              <span className={`${this.state.activeTab === "podEvents" ? "is-active" : ""} tab-item blue`} onClick={() => this.togglePodDetailView("podEvents")}>Pod events</span>
            </div>
            <div className="flex flex1 action-content">
              <div className="flex1 flex-column file-contents-wrapper u-position--relative">
                {this.renderPodDetailView()}
              </div>
            </div>
          </div>
        </div>

    );
  }
}

export default withRouter(PodAnalyzerDetails);
