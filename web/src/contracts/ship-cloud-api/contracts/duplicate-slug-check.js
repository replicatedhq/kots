import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { getShipClient, createSessionToken } from "../utils";
import { createNewWatch } from "../../../mutations/WatchMutations";
import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { createNewWatchRaw } from "../../../mutations/WatchMutations";

chai.use(chaiAsPromised);

export default () => {
  it("creates a watch that will default to a duplicate slug", async (done) => {
    await global.provider.addInteraction(createDuplicateSlugInteraction);
    const result = await getShipClient("duplicate-slug-account-session-1").mutate({
        mutation: createNewWatch,
        variables: {
          owner: "duplicate-slug-account",
          stateJSON: `{
  "v1": {
    "config": {},
    "releaseName": "factorio",
    "helmValuesDefaults": "",
    "upstream": "",
    "metadata": {
      "applicationType": "helm",
      "icon": "https://us1.factorio.com/assets/img/factorio-logo.png",
      "name": "duplicate-slug-one",
      "releaseNotes": "",
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

const createDuplicateSlugInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a mutation to create a watch that will default to a duplicate slug")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("duplicate-slug-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withOperation("createWatch")
  .withQuery(createNewWatchRaw)
  .withVariables({
    owner: "duplicate-slug-account",
    stateJSON: `{
  "v1": {
    "config": {},
    "releaseName": "factorio",
    "helmValuesDefaults": "",
    "upstream": "",
    "metadata": {
      "applicationType": "helm",
      "icon": "https://us1.factorio.com/assets/img/factorio-logo.png",
      "name": "duplicate-slug-one",
      "releaseNotes": "",
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
          slug: "duplicate-slug-account/duplicate-slug-one-1",
          watchName: Matchers.like("generated"),
          createdOn: Matchers.like("generated"),
          lastUpdated: null,
        },
      },
    },
  });
