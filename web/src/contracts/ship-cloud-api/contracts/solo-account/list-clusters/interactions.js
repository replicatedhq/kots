import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { listClustersRaw } from "../../../../../queries/ClusterQueries";
import { createSessionToken } from "../../../utils";

export const listClustersInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a query to list clusters for solo account")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("solo-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withQuery(listClustersRaw)
  .withOperation("listClusters")
  .withVariables({

  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        listClusters: [
          {
            "id": "solo-account-cluster-1",
            "title": "Solo Cluster",
            "slug": "solo-cluster",
            "totalApplicationCount": 1,
            "createdOn": Matchers.like("2019-04-10 12:34:56.789"),
            "lastUpdated": Matchers.like("2019-04-11 01:23:45.567"),
            "gitOpsRef": null,
            "shipOpsRef": {
              "token": "solo-account-cluster-token",
            },
          },
          {
            "id": "solo-account-cluster-2",
            "title": "Solo GitHub Cluster",
            "slug": "solo-cluster-2",
            "totalApplicationCount": 0,
            "createdOn": Matchers.like("2019-04-10 12:34:56.789"),
            "lastUpdated": Matchers.like("2019-04-11 01:23:45.567"),
            "gitOpsRef": {
              "owner": "lonely-github-dev",
              "repo": "gitops-deploy",
              "branch": "master"
            },
            "shipOpsRef": null,
          }
        ]
      }
    }
  });

export const listClustersAfterCreatingShipInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a query to list clusters for solo account after creating a ship cluster")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("solo-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withQuery(listClustersRaw)
  .withOperation("listClusters")
  .withVariables({

  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        listClusters: [
          {
            "id": "solo-account-cluster-1",
            "title": "Solo Cluster",
            "slug": "solo-cluster",
            "createdOn": Matchers.like("2019-04-10 12:34:56.789"),
            "lastUpdated": Matchers.like("2019-04-11 01:23:45.567"),
            "gitOpsRef": null,
            "shipOpsRef": {
              "token": "solo-account-cluster-token",
            },
          },
          {
            "id": "solo-account-cluster-2",
            "title": "Solo GitHub Cluster",
            "slug": "solo-cluster-2",
            "createdOn": Matchers.like("2019-04-10 12:34:56.789"),
            "lastUpdated": Matchers.like("2019-04-11 01:23:45.567"),
            "gitOpsRef": {
              "owner": "lonely-github-dev",
              "repo": "gitops-deploy",
              "branch": "master"
            },
            "shipOpsRef": null,
          },
          {
            "id": Matchers.like("generated"),
            "title": "FooBarBaz Cluster",
            "slug": "foobarbaz-cluster",
            "createdOn": Matchers.like("2019-04-10 12:34:56.789"),
            "lastUpdated": null,
            "gitOpsRef": null,
            "shipOpsRef": {
              "token": Matchers.like("generated"),
            },
          }
        ]
      }
    }
  });

  export const listClustersAfterCreatingGitOpsInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a query to list clusters for solo account after creating a gitops cluster")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("solo-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withQuery(listClustersRaw)
  .withOperation("listClusters")
  .withVariables({

  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        listClusters: [
          {
            "id": "solo-account-cluster-1",
            "title": "Solo Cluster",
            "slug": "solo-cluster",
            "createdOn": Matchers.like("2019-04-10 12:34:56.789"),
            "lastUpdated": Matchers.like("2019-04-11 01:23:45.567"),
            "gitOpsRef": null,
            "shipOpsRef": {
              "token": "solo-account-cluster-token",
            },
          },
          {
            "id": "solo-account-cluster-2",
            "title": "Solo GitHub Cluster",
            "slug": "solo-cluster-2",
            "createdOn": Matchers.like("2019-04-10 12:34:56.789"),
            "lastUpdated": Matchers.like("2019-04-11 01:23:45.567"),
            "gitOpsRef": {
              "owner": "lonely-github-dev",
              "repo": "gitops-deploy",
              "branch": "master"
            },
            "shipOpsRef": null,
          },
          {
            "id": Matchers.like("generated"),
            "title": "FooBarGit Cluster",
            "slug": "foobargit-cluster",
            "createdOn": Matchers.like("2019-04-10 12:34:56.789"),
            "lastUpdated": null,
            "gitOpsRef": {
              "owner": "me",
              "repo": "myself",
              "branch": "i",
            },
            "shipOpsRef": null,
          }
        ]
      }
    }
  });
