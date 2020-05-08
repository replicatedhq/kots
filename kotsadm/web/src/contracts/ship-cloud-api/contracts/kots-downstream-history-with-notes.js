
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
  it("gets downstream version history for a kots app that has release notes", async done => {

    await global.provider.addInteraction(getKotsDownstreamHistoryInteraction);
    const result = await getShipClient("kots-release-notes-user-session").mutate({
      mutation: getKotsDownstreamHistory,
      variables: {
        clusterSlug: "kots-release-notes-cluster-slug",
        upstreamSlug: "kots-release-notes-app-slug"
      },
    });
    const{ data } = result;
    const downstream = data.getKotsDownstreamHistory[0];

    expect(downstream.title).to.equal("my-other-awesome-version");
    expect(downstream.status).to.equal("pending");
    expect(typeof downstream.createdOn).to.equal("string");
    expect(downstream.sequence).to.equal(0);
    expect(downstream.releaseNotes).to.equal("# Release Notes Markdown Text");
    expect(typeof downstream.preflightResult).to.equal("string");
    expect(typeof downstream.preflightResultCreatedAt).to.equal("string");

    global.provider.verify().then(() => done());
  });

  const getKotsDownstreamHistoryInteraction = new Pact.GraphQLInteraction()
    .uponReceiving("A query to get kots downstream version history that has release notes")
    .withRequest({
      path: "/graphql",
      method: "POST",
      headers: {
        "Authorization": createSessionToken("kots-release-notes-user-session"),
        "Content-Type": "application/json",
      }
    })
    .withOperation("getKotsDownstreamHistory")
    .withQuery(getKotsDownstreamHistoryRaw)
    .withVariables({
      clusterSlug: "kots-release-notes-cluster-slug",
      upstreamSlug: "kots-release-notes-app-slug"
    })
    .willRespondWith({
      status: 200,
      headers: { "Content-Type": "application/json" },
      body: {
        data: {
          getKotsDownstreamHistory: [
            {
              "title": "my-other-awesome-version",
              "status": "pending",
              "createdOn": Matchers.like("Fri Apr 19 2019 01:23:45 GMT+0000 (Coordinated Universal Time)"),
              "sequence": 0,
              "releaseNotes": "# Release Notes Markdown Text",
              "deployedAt": Matchers.like("Fri Apr 19 2019 01:23:45 GMT+0000 (Coordinated Universal Time)"),
              "preflightResult": Matchers.like("string"),
              "preflightResultCreatedAt": Matchers.like("Fri Apr 19 2019 01:23:45 GMT+0000 (Coordinated Universal Time)")
            }
          ],
        },
      },
    });
};
