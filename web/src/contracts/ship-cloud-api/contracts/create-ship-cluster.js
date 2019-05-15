import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import * as _ from "lodash";
import { createShipOpsCluster } from "../../../mutations/ClusterMutations";
import { getShipClient, createSessionToken } from "../utils";
import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { createShipOpsClusterRaw } from "../../../mutations/ClusterMutations";

chai.use(chaiAsPromised);

export default () => {
  it("creates a ship cluster for solo dev", async (done) => {
    await global.provider.addInteraction(createShipClusterInteraction);
    const result = await getShipClient("solo-account-session-1").mutate({
      mutation: createShipOpsCluster,
      variables: {
        title: "FooBarBaz Cluster",
      },
    });
    // expect(result.data.createShipOpsCluster).to.deep.equal({"id": "generated", "slug": "foobarbaz-cluster", "shipOpsRef": {"token": "generated"}})
    // createdClusterId = result.data.createShipOpsCluster.id;
    global.provider.verify().then(() => done());
  });
}

const createShipClusterInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a mutation to create a ship cluster for solo dev")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("solo-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withOperation("createShipOpsCluster")
  .withQuery(createShipOpsClusterRaw)
  .withVariables({
    title: "FooBarBaz Cluster",
  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        createShipOpsCluster: {
          id: Matchers.like("generated"),
          slug: "foobarbaz-cluster",
          shipOpsRef: {
            token: Matchers.like("generated"),
          },
        },
      },
    },
  });
