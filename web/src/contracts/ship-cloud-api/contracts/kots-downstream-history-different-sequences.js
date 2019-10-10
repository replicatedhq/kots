
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
  it("gets downstream version history for a kots app that has different midstream sequence than the downstream", async done => {

    await global.provider.addInteraction(getKotsDownstreamHistoryInteraction);
    const result = await getShipClient("kots-different-sequence-user-session").mutate({
      mutation: getKotsDownstreamHistory,
      variables: {
        clusterSlug: "kots-different-sequence-cluster-slug",
        upstreamSlug: "kots-different-sequence-app-slug"
      },
    });
    const{ data } = result;
    const downstream = data.getKotsDownstreamHistory[1];

    expect(downstream.title).to.equal("my-other-awesome-version");
    expect(downstream.status).to.equal("deployed");
    expect(typeof downstream.createdOn).to.equal("string");
    expect(downstream.sequence).to.equal(0);
    expect(downstream.releaseNotes).to.equal("# Markdown string here");
    expect(typeof downstream.preflightResult).to.equal("string");
    expect(typeof downstream.preflightResultCreatedAt).to.equal("string");

    global.provider.verify().then(() => done());
  });

  const getKotsDownstreamHistoryInteraction = new Pact.GraphQLInteraction()
    .uponReceiving("A query to get downstream version history for a kots app that has different midstream sequence than the downstream")
    .withRequest({
      path: "/graphql",
      method: "POST",
      headers: {
        "Authorization": createSessionToken("kots-different-sequence-user-session"),
        "Content-Type": "application/json",
      }
    })
    .withOperation("getKotsDownstreamHistory")
    .withQuery(getKotsDownstreamHistoryRaw)
    .withVariables({
      clusterSlug: "kots-different-sequence-cluster-slug",
      upstreamSlug: "kots-different-sequence-app-slug"
    })
    .willRespondWith({
      status: 200,
      headers: { "Content-Type": "application/json" },
      body: {
        data: {
          getKotsDownstreamHistory: [
            {
              "title": "my-other-awesome-version-2",
              "status": "deployed",
              "createdOn": "Fri Apr 19 2019 01:23:45 GMT+0000 (UTC)",
              "sequence": 1,
              "releaseNotes": "# Markdown string here",
              "deployedAt": "Fri Apr 19 2019 01:23:45 GMT+0000 (UTC)",
              "preflightResult": Matchers.like("string"),
              "preflightResultCreatedAt": "Fri Apr 19 2019 01:23:45 GMT+0000 (UTC)"
            },
            {
              "title": "my-other-awesome-version",
              "status": "deployed",
              "createdOn": "Fri Apr 19 2019 01:23:45 GMT+0000 (UTC)",
              "sequence": 0,
              "releaseNotes": "# Markdown string here",
              "deployedAt": "Fri Apr 19 2019 01:23:45 GMT+0000 (UTC)",
              "preflightResult": Matchers.like("string"),
              "preflightResultCreatedAt": "Fri Apr 19 2019 01:23:45 GMT+0000 (UTC)"
            }
          ],
        },
      },
    });
};
