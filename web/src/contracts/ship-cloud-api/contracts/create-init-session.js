import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { getShipClient, createSessionToken } from "../utils";
import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { createInitSessionRaw } from "../../../mutations/WatchMutations";

import { createInitSession } from "../../../mutations/WatchMutations";

chai.use(chaiAsPromised);

export default () => {
  it("creates an init session for solo dev", async (done) => {
    await global.provider.addInteraction(createHelmInitSessionInteraction);
    const result = await getShipClient("solo-account-session-1").mutate({
      mutation: createInitSession,
      variables: {
        pendingInitId: "",
        upstreamUri: "https://github.com/helm/charts/stable/grafana",
        clusterID: null,
        githubPath: null,
      },
    });
    global.provider.verify().then(() => done());
    });
}

const createHelmInitSessionInteraction = new Pact.GraphQLInteraction()
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
    pendingInitId: "",
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
        },
      },
    },
  });
