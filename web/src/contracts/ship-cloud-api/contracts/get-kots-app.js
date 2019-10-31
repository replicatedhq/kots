
/* global
  it
*/

import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";

import { getKotsApp, getKotsAppRaw } from "../../../queries/AppsQueries";
import { getShipClient, createSessionToken } from "../utils";

chai.use(chaiAsPromised);
const expect = chai.expect;

export default () => {
  it("gets a kots app", async done => {

    await global.provider.addInteraction(getKotsAppInteraction);
    const result = await getShipClient("get-kots-app-user-session").mutate({
      mutation: getKotsApp,
      variables: {
        slug: "kots-app-slug"
      },
    });
    expect(result.data.getKotsApp.id).to.equal("get-kots-app-id");
    expect(result.data.getKotsApp.name).to.equal("kots-app-name");
    expect(result.data.getKotsApp.slug).to.equal("kots-app-slug");
    expect(result.data.getKotsApp.currentSequence).to.equal(0);
    expect(result.data.getKotsApp.hasPreflight).to.equal(false);
    expect(result.data.getKotsApp.isAirgap).to.equal(false);
    expect(result.data.getKotsApp.currentVersion).to.equal(null);

    global.provider.verify().then(() => done());

  });

  const getKotsAppInteraction = new Pact.GraphQLInteraction()
    .uponReceiving("A query to get a kots app")
    .withRequest({
      path: "/graphql",
      method: "POST",
      headers: {
        "Authorization": createSessionToken("get-kots-app-user-session"),
        "Content-Type": "application/json",
      }
    })
    .withOperation("getKotsApp")
    .withQuery(getKotsAppRaw)
    .withVariables({
      slug: "kots-app-slug"
    })
    .willRespondWith({
      status: 200,
      headers: { "Content-Type": "application/json" },
      body: {
        data: {
          getKotsApp: {
            id: "get-kots-app-id",
            name: "kots-app-name",
            createdAt: Matchers.like("date"),
            updatedAt: Matchers.like("date"),
            slug: "kots-app-slug",
            currentSequence: 0,
            hasPreflight: false,
            isAirgap: false,
            currentVersion: null,
            lastUpdateCheckAt: Matchers.like("date"),
            // This is undefined because it's coming from the Params.getParams() which aren't set right now
            bundleCommand: "\n      curl https://krew.sh/support-bundle | bash\n      kubectl support-bundle undefined/api/v1/troubleshoot/kots-app-slug\n    ",
            downstreams: []
          },
        },
      },
    });
};
