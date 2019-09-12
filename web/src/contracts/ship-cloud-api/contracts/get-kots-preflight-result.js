/* global
  it
*/
import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { getShipClient, createSessionToken } from "../utils";
import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { getKotsPreflightResult, getKotsPreflightResultRaw } from "../../../queries/AppsQueries";

chai.use(chaiAsPromised);
const expect = chai.expect;

export default () => {
  it("Gets a preflight result for a kots app", async done => {
    await global.provider.addInteraction(getKotsPreflightResultInteraction);
    const result = await getShipClient("get-kots-preflight-result-user-session").query({
      query: getKotsPreflightResult,
      variables: {
        appSlug: "get-kots-preflight-result-app-slug",
        clusterSlug: "get-kots-preflight-result-cluster-slug",
        sequence: 0
      },
    });

    global.provider.verify().then(() => done());
  });
};

const getKotsPreflightResultInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a query to get a kots preflight result")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("get-kots-preflight-result-user-session"),
      "Content-Type": "application/json",
    }
  })
  .withOperation("getKotsPreflightResult")
  .withQuery(getKotsPreflightResultRaw)
  .withVariables({
    appSlug: "get-kots-preflight-result-app-slug",
    clusterSlug: "get-kots-preflight-result-cluster-slug",
    sequence: 0

  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        getKotsPreflightResult: {
          appSlug: "get-kots-preflight-result-app-slug",
          clusterSlug: "get-kots-preflight-result-cluster-slug",
          result: "{\"results\":[{\"isWarn\":true,\"title\":\"Required Kubernetes Version\",\"message\":\"Your cluster meets the minimum version of Kubernetes, but we recommend you update to 1.15.0 or later.\",\"uri\":\"https://kubernetes.io\"}]}",
          createdAt: Matchers.like("date")
        },
      },
    },
  });
