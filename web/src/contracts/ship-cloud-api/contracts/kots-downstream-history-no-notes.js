
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
  it("gets downstream version history kots app without release notes", async done => {

    await global.provider.addInteraction(getKotsDownstreamHistoryInteraction);
    const result = await getShipClient("kots-no-release-notes-user-session").mutate({
      mutation: getKotsDownstreamHistory,
      variables: {
        clusterSlug: "kots-no-release-notes-cluster-slug",
        upstreamSlug: "kots-no-release-notes-app-slug"
      },
    });
    const { data } = result;

    expect(data.getKotsDownstreamHistory[0].title).to.equal("kots-no-release-notes-version-label");
    expect(data.getKotsDownstreamHistory[0].status).to.equal("pending");
    expect(typeof data.getKotsDownstreamHistory[0].createdOn).to.equal("string");
    expect(data.getKotsDownstreamHistory[0].sequence).to.equal(0);
    expect(data.getKotsDownstreamHistory[0].releaseNotes).to.equal("");
    expect(data.getKotsDownstreamHistory[0].preflightResult).to.equal(null);
    expect(data.getKotsDownstreamHistory[0].preflightResultCreatedAt).to.equal(null);

    global.provider.verify().then(() => done());

  });

  const getKotsDownstreamHistoryInteraction = new Pact.GraphQLInteraction()
    .uponReceiving("A query to get kots downstream version history without release notes")
    .withRequest({
      path: "/graphql",
      method: "POST",
      headers: {
        "Authorization": createSessionToken("kots-no-release-notes-user-session"),
        "Content-Type": "application/json",
      }
    })
    .withOperation("getKotsDownstreamHistory")
    .withQuery(getKotsDownstreamHistoryRaw)
    .withVariables({
      clusterSlug: "kots-no-release-notes-cluster-slug",
      upstreamSlug: "kots-no-release-notes-app-slug"
    })
    .willRespondWith({
      status: 200,
      headers: { "Content-Type": "application/json" },
      body: {
        data: {
          getKotsDownstreamHistory: [
            {
              "title": "kots-no-release-notes-version-label",
              "status": "pending",
              "createdOn": "Fri Apr 19 2019 01:23:45 GMT+0000 (UTC)",
              "sequence": 0,
              "releaseNotes": "",
              "deployedAt": "Fri Apr 19 2019 01:23:45 GMT+0000 (UTC)",
              "preflightResult": null,
              "preflightResultCreatedAt": null
            }

          ],
        },
      },
    });
};
