import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { createSessionToken } from "../../../utils";
import { createInitSessionRaw } from "../../../../../mutations/WatchMutations";

export const createHelmInitSessionInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a mutation to create a helm init session")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("solo-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withOperation("createInitSession")
  .withQuery(createInitSessionRaw)
  .withVariables({
    upstreamUri: "https://github.com/helm/charts/stable/grafana",
    clusterID: null,
    githubPath: null,
  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        createInitSession: {
          id: Matchers.like("generated"),
          upstreamUri: "https://github.com/helm/charts/stable/grafana",
          createdOn: Matchers.like("generated"),
          finishedOn: null,
          result: null,
        },
      },
    },
  });
