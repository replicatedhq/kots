import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { listWatchesRaw } from "../../../../../queries/WatchQueries";
import { createSessionToken } from "../../../utils";

export const listWatchesInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a query to list watches for ship-clusters account")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("ship-cluster-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withQuery(listWatchesRaw)
  .withOperation("listWatches")
  .withVariables({

  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        listWatches: [
          {
            "id": "better-db-midstream",
            "stateJSON": Matchers.like("\n  \"v1\": {\n    \"config\": null,\n    \"helmValues\": \"# Default values for better-db.\\n# This is a YAML-formatted file.\\n# Declare variables to be passed into your templates.\\n\\nreplicaCount: 1\\n\\nimage:\\n  repository: nginx\\n  tag: 1.15.1\\n  pullPolicy: IfNotPresent\\n\\nservice:\\n  type: ClusterIP\\n  port: 80\\n\\nsecurityContext:\\n  allowPrivilegeEscalation: true\\n\\nresources:\\n  # limits:\\n  #  cpu: 100m\\n  #  memory: 128Mi\\n  # requests:\\n  #  cpu: 100m\\n #  memory: 128Mi\\n\\nnodeSelector: {}\\n\\ntolerations: []\\n\\naffinity: {}\\n\",\n    \"releaseName\": \"better-db\",\n    \"helmValuesDefaults\": \"# Default values for better-db.\\n# This is a YAML-formatted file.\\n# Declare variables to be passed into your templates.\\n\\nreplicaCount: 1\\n\\nimage:\\nrepository: nginx\\n  tag: 1.15.1\\n  pullPolicy: IfNotPresent\\n\\nservice:\\n  type: ClusterIP\\n  port:80\\n\\nsecurityContext:\\n  allowPrivilegeEscalation: true\\n\\nresources:\\n  # limits:\\n  #  cpu: 100m\\n  #  memory: 128Mi\\n  # requests:\\n  #  cpu: 100m\\n  #  memory: 128Mi\\n\\nnodeSelector: {}\\n\\ntolerations: []\\n\\naffinity: {}\\n\",\n    \"upstream\": \"github.com/better-db/chart\",\n    \"metadata\": {\n   \"applicationType\": \"helm\",\n      \"sequence\": 0,\n      \"name\": \"better-db\",\n  \"releaseNotes\": \"bump\",\n      \"version\": \"0.1.3\",\n      \"license\": {\n        \"id\": \"\",\n    \"assignee\": \"\",\n        \"createdAt\": \"0001-01-01T00:00:00Z\",\n        \"expiresAt\": \"0001-01-01T00:00:00Z\",\n        \"type\": \"\"\n      }\n    },\n    \"contentSHA\": \"f6ce910a6e0d560c8687b774cf5e4f8848de312819b9173834fabe297a34a6c3\",\n    \"lifecycle\": {\n      \"stepsCompleted\": {\n\"intro\": true,\n        \"kustomize\": true,\n        \"kustomize-intro\": true,\n        \"render\": true,\n      \"values\": true\n      }\n    }\n  }\n}\n"),
            "watchName": "Better DB Midstream",
            "slug": "ship-cluster-account/better-db-midstream",
            "createdOn": Matchers.like("2019-04-18 12:34:56.789"),
            "lastUpdated": Matchers.like("2019-04-19 01:23:45.567"),
            "watchIcon": "",
            "contributors": [
              {
                "id": "ship-cluster-account",
                "createdAt": Matchers.like("2019-04-18 12:34:56.789"),
                "githubId": 2222,
                "login": "ship-cluster-dev",
                "avatar_url": "https://avatars3.githubusercontent.com/u/234567?v=4",
              }
            ],
            "currentVersion": {
              "title": "0.1.3",
              "status": "deployed",
              "createdOn": Matchers.like("2019-04-18 12:34:56.789"),
              "sequence": 0,
              "pullrequestNumber": null,
            },
            "pendingVersions": [

            ],
            "pastVersions": [

            ],
            "notifications": [

            ],
            "watches": [
              {
                "id": "better-db-prod",
                "stateJSON": "{\n  \"v1\": {\n    \"config\": null,\n    \"releaseName\": \"ship\",\n    \"upstream\": \"http://ship-cloud-api.default.svc.cluster.local:3000/v1/watch/better-db-midstream/upstream.yaml?token=better-db-midstream-downstream-1\",\n    \"contentSHA\": \"2a5418afdd4eb29e72fe63b4ce756fbdb56f1553b4b5848b921d28ef4ab8421a\",\n    \"lifecycle\": {\n      \"stepsCompleted\": {\n        \"kustomize\": true,\n        \"kustomize-intro\": true,\n        \"render\": true\n      }\n    }\n  }\n}\n",
                "watchName": "Better DB Ship 1",
                "slug": "ship-cluster-account/better-db-prod",
                "createdOn": Matchers.like("2019-04-19 12:34:56.789"),
                "lastUpdated": Matchers.like("2019-04-20 01:23:45.567"),
                "watchIcon": "",
                "contributors": [
                  {
                    "id": "ship-cluster-account",
                    "createdAt": Matchers.like("2019-04-18 12:34:56.789"),
                    "githubId": 2222,
                    "login": "ship-cluster-dev",
                    "avatar_url": "https://avatars3.githubusercontent.com/u/234567?v=4",
                  }
                ],
                "currentVersion": {
                  "title": "0.1.3",
                  "status": "deployed",
                  "createdOn": Matchers.like("2019-04-19 12:34:56.789"),
                  "sequence": 0,
                  "pullrequestNumber": null,
                },
                "pendingVersions": [
                  {
                    "title": "0.1.4",
                    "status": "pending",
                    "createdOn": Matchers.like("2019-04-20 12:34:56.789"),
                    "sequence": 1,
                    "pullrequestNumber": null,
                  }
                ],
                "pastVersions": [

                ],
                "notifications": [

                ],
                "cluster": {
                  "id": "ship-cluster-1",
                  "title": "Ship Cluster 1",
                  "slug": "ship-cluster-1",
                  "createdOn": Matchers.like("2019-04-18 12:34:56.78"),
                  "lastUpdated": Matchers.like("2019-04-19 01:23:45.56"),
                  "gitOpsRef": null,
                  "shipOpsRef": {
                    "token": "ship-cluster-1-token"
                  }
                }
              },
            ]
          },
        ]
      }
    }
  });

