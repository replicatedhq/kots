import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { createSessionToken } from "../../../utils";
import { createShipOpsClusterRaw } from "../../../../../mutations/ClusterMutations";

export const createShipClusterInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a mutation to create a ship cluster for solo dev")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("solo-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withOperation("createShipOpsCluster")
  .withQuery(createShipOpsClusterRaw)
  .withVariables({
    title: "FooBarBaz Cluster",
  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        createShipOpsCluster: {
          id: Matchers.like("generated"),
          slug: "foobarbaz-cluster",
          shipOpsRef: {
            token: Matchers.like("generated"),
          },
        },
      },
    },
  });
