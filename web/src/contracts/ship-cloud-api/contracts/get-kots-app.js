import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";

import { getKotsApp, getKotsAppRaw } from "../../../queries/AppsQueries";
import { getShipClient, createSessionToken } from "../utils";

chai.use(chaiAsPromised);

export default () => {
  it("gets a kots app", async done => {
    await global.provider.addInteraction(getKotsAppInteraction);
    const result = await getShipClient("solo-account-session-1").mutate({
      mutation: getKotsApp,
      variables: {
        slug: "kots-app-slug"
      },
    });

    global.provider.verify().then(() => done());

  });

  const getKotsAppInteraction = new Pact.GraphQLInteraction()
    .uponReceiving("A query to get a kots app")
    .withRequest({
      path: "/graphql",
      method: "POST",
      headers: {
        "Authorization": createSessionToken("kots-app-session-1"),
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
            name: "",
            createdOn: Matchers.like("date????"),
          },
        },
      },
    });
};