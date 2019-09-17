
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
        slug: "kots-app-slug"
      },
    });
    // expect(result.data.getKotsApp.id).to.equal("get-kots-downstream-history-id");
    // expect(result.data.getKotsApp.name).to.equal("kots-app-name");
    // expect(result.data.getKotsApp.slug).to.equal("kots-app-slug");
    // expect(result.data.getKotsApp.currentSequence).to.equal(0);
    // expect(result.data.getKotsApp.hasPreflight).to.equal(false);
    // expect(result.data.getKotsApp.isAirgap).to.equal(false);
    // expect(result.data.getKotsApp.currentVersion).to.equal(null);

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
          getKotsDownstreamHistory: {
            title: "get-kots-downstream-history-cluster-title",
            status: "pending",
            createdOn: Matchers.like("date"),
            sequence: 0,
            deployedAt: Matchers.like("date"),
            preflightResult: Matchers.like("JSONPreflightResult"),
            preflightResultUpdateAt: Matchers.like("date")
          },
        },
      },
    });
};
