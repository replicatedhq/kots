
/* global
  it
*/

import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";

import { getKotsDownstreamHistory, getKotsDownstreamHistoryRaw } from "../../../queries/AppsQueries";
import { getShipClient, createSessionToken } from "../utils";

chai.use(chaiAsPromised);
const expect = chai.expect;

export default () => {
  it("gets downstream version history for a kots app", async done => {

    await global.provider.addInteraction(getKotsDownstreamHistoryInteraction);
    const result = await getShipClient("get-kots-downstream-history-user-session").mutate({
      mutation: getKotsDownstreamHistory,
      variables: {
        clusterSlug: "get-kots-downstream-history-cluster-slug",
        upstreamSlug: "get-kots-downstream-history-app-slug"
      },
    });
    expect(result.data.getKotsDownstreamHistory[0].title).to.equal("my-awesome-version");
    expect(result.data.getKotsDownstreamHistory[0].status).to.equal("pending");
    expect(typeof result.data.getKotsDownstreamHistory[0].createdOn).to.equal("string");
    expect(result.data.getKotsDownstreamHistory[0].sequence).to.equal(0);
    expect(typeof result.data.getKotsDownstreamHistory[0].deployedAt).to.equal("string");
    expect(typeof result.data.getKotsDownstreamHistory[0].preflightResult).to.equal("string");
    expect(typeof result.data.getKotsDownstreamHistory[0].preflightResultCreatedAt).to.equal("string");

    global.provider.verify().then(() => done());

  });

  const getKotsDownstreamHistoryInteraction = new Pact.GraphQLInteraction()
    .uponReceiving("A query to get kots downstream version history")
    .withRequest({
      path: "/graphql",
      method: "POST",
      headers: {
        "Authorization": createSessionToken("get-kots-downstream-history-user-session"),
        "Content-Type": "application/json",
      }
    })
    .withOperation("getKotsDownstreamHistory")
    .withQuery(getKotsDownstreamHistoryRaw)
    .withVariables({
      clusterSlug: "get-kots-downstream-history-cluster-slug",
      upstreamSlug: "get-kots-downstream-history-app-slug"
    })
    .willRespondWith({
      status: 200,
      headers: { "Content-Type": "application/json" },
      body: {
        data: {
          getKotsDownstreamHistory: [
            {
              title: "my-awesome-version",
              status: "pending",
              createdOn: Matchers.like("date"),
              sequence: 0,
              deployedAt: Matchers.like("date"),
              preflightResult: Matchers.like("JSONPreflightResult"),
              preflightResultCreatedAt: Matchers.like("date")
            }
          ],
        },
      },
    });
};
