import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { getShipClient, createSessionToken } from "../utils";
import { createNewWatch } from "../../../mutations/WatchMutations";
import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { createNewWatchRaw } from "../../../mutations/WatchMutations";

chai.use(chaiAsPromised);

export default () => {
  it("creates a midstream watch for solo dev", async (done) => {
    await global.provider.addInteraction(createMidstreamWatchInteraction);
    const result = await getShipClient("solo-account-session-1").mutate({
        mutation: createNewWatch,
        variables: {
          owner: "solo-account",
          stateJSON: `{
  "v1": {
    "config": {},
    "releaseName": "factorio",
    "helmValuesDefaults": "",
    "upstream": "https://github.com/helm/charts/tree/ffb84f85a861e765caade879491a75a6dd3091a5/stable/factorio",
    "metadata": {
      "applicationType": "helm",
      "icon": "https://us1.factorio.com/assets/img/factorio-logo.png",
      "name": "factorio",
      "releaseNotes": "",
      "license": {
        "assignee": "",
        "createdAt": "0001-01-01T00:00:00Z",
        "expiresAt": "0001-01-01T00:00:00Z",
        "id": "",
        "type": ""
      },
      "sequence": 0,
      "version": "0.3.1"
    },
    "contentSHA": "126fa6eb8f09171050751c65a386f41aef4fe9ebe00c8b1e66f2c4e60319ec4e"
  }
}`
        },
    });
    global.provider.verify().then(() => done());
  });
}

const createMidstreamWatchInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a mutation to create a midstream watch for solo dev")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("solo-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withOperation("createWatch")
  .withQuery(createNewWatchRaw)
  .withVariables({
    owner: "solo-account",
    stateJSON: `{
  "v1": {
    "config": {},
    "releaseName": "factorio",
    "helmValuesDefaults": "",
    "upstream": "https://github.com/helm/charts/tree/ffb84f85a861e765caade879491a75a6dd3091a5/stable/factorio",
    "metadata": {
      "applicationType": "helm",
      "icon": "https://us1.factorio.com/assets/img/factorio-logo.png",
      "name": "factorio",
      "releaseNotes": "",
      "license": {
        "assignee": "",
        "createdAt": "0001-01-01T00:00:00Z",
        "expiresAt": "0001-01-01T00:00:00Z",
        "id": "",
        "type": ""
      },
      "sequence": 0,
      "version": "0.3.1"
    },
    "contentSHA": "126fa6eb8f09171050751c65a386f41aef4fe9ebe00c8b1e66f2c4e60319ec4e"
  }
}`,
  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        createWatch: {
          id: Matchers.like("generated"),
          slug: Matchers.like("generated"),
          watchName: Matchers.like("generated"),
          createdOn: Matchers.like("generated"),
          lastUpdated: null,
        },
      },
    },
  });
