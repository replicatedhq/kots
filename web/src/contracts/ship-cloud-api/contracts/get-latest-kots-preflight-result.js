/* global
  it
*/
import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { getShipClient, createSessionToken } from "../utils";
import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { getLatestKotsPreflight, getLatestKotsPreflightRaw } from "../../../queries/AppsQueries";

chai.use(chaiAsPromised);
const expect = chai.expect;

export default () => {
  it("Gets the latest preflight result for a kots app", async done => {
    await global.provider.addInteraction(getLatestKotsPreflightResultInteraction);
    const result = await getShipClient("get-latest-kots-preflight-result-user-session").query({
      query: getLatestKotsPreflight
    });

    // const { getLatestKotsPreflightResult: gqlData } = result.data;

    // expect(gqlData.appSlug).to.equal("get-latest-kots-preflight-result-app-slug");
    // expect(gqlData.clusterSlug).to.equal("get-latest-kots-preflight-result-cluster-slug");
    // expect(typeof gqlData.result).to.equal("string");
    // expect(typeof gqlData.createdAt).to.equal("string");

    global.provider.verify().then(() => done());
  });
};

const getLatestKotsPreflightResultInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a query to get the latest kots preflight result")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("get-latest-kots-preflight-result-user-session"),
      "Content-Type": "application/json",
    }
  })
  .withOperation("getLatestKotsPreflight")
  .withQuery(getLatestKotsPreflightRaw)
  .withVariables({})
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        getLatestKotsPreflightResult: {
          appSlug: "get-latest-kots-preflight-result-app-slug",
          clusterSlug: "get-latest-kots-preflight-result-cluster-slug",
          result: Matchers.like("JSONString"),
          createdAt: Matchers.like("date")
        },
      },
    },
  });
