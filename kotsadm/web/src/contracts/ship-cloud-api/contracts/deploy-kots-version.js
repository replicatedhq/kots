/* global
  it
*/
import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { getShipClient, createSessionToken } from "../utils";
import * as Pact from "@pact-foundation/pact";
import { deployKotsVersion, deployKotsVersionRaw } from "../../../mutations/AppsMutations";

chai.use(chaiAsPromised);
const expect = chai.expect;

export default () => {
  it("deploys a kots version", async (done) => {
    await global.provider.addInteraction(deployKotsVersionInteraction);
    const result = await getShipClient("deploy-kots-version-user-session").mutate({
      mutation: deployKotsVersion,
      variables: {
        upstreamSlug: "deploy-kots-version-app-slug",
        sequence: 1,
        clusterSlug: "deploy-kots-version-cluster-slug"
      },
    });
    // NOTE: This isn't a super great graphQL mutation if it just returns true...

    expect(result.data.deployKotsVersion).to.equal(true);

    global.provider.verify().then(() => done());
  });
}

const deployKotsVersionInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a mutation to deploy a kots version")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("deploy-kots-version-user-session"),
      "Content-Type": "application/json",
    }
  })
  .withOperation("deployKotsVersion")
  .withQuery(deployKotsVersionRaw)
  .withVariables({
    upstreamSlug: "deploy-kots-version-app-slug",
    sequence: 1,
    clusterSlug: "deploy-kots-version-cluster-slug"
  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        deployKotsVersion: true
      },
    },
  });
