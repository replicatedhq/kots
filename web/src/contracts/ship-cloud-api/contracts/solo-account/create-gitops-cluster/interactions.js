import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { createSessionToken } from "../../../utils";
import { createGitOpsClusterRaw } from "../../../../../mutations/ClusterMutations";

export const createGitOpsClusterInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a mutation to create a gitops cluster for solo dev")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("solo-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withOperation("createGitOpsCluster")
  .withQuery(createGitOpsClusterRaw)
  .withVariables({
    title: "FooBarGit Cluster",
    installationId: 987654,
    gitOpsRef: {
      owner: "me",
      repo: "myself",
      branch: "i",
    },
  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        createGitOpsCluster: {
          id: Matchers.like("generated"),
          slug: "foobargit-cluster",
        },
      },
    },
  });
