import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import fetch from "node-fetch";
import * as _ from "lodash";
import { ShipClientGQL } from "../../../../../ShipClientGQL";
import { createSessionToken } from "../../../utils";

import { listClusters } from "../../../../../queries/ClusterQueries";
import { createShipOpsCluster } from "../../../../../mutations/ClusterMutations";

import { createShipClusterInteraction } from "./interactions";
import { listClustersAfterCreatingShipInteraction } from "../list-clusters/interactions";

chai.use(chaiAsPromised);
const expect = chai.expect;

const MOCK_SERVER_PORT = 3333;

export default () => {
  let createdClusterId;

  it("creates a ship cluster for solo dev", (done) => {
    global.provider.addInteraction(createShipClusterInteraction).then(() => {
      const shipClient = ShipClientGQL(`http://localhost:${MOCK_SERVER_PORT}/graphql`, async () => { return createSessionToken("solo-account-session-1") }, fetch);
      shipClient.mutate({
        mutation: createShipOpsCluster,
        variables: {
          title: "FooBarBaz Cluster",
        },
      })
      .then(result => {
        // expect(result.data.createShipOpsCluster).to.deep.equal({"id": "generated", "slug": "foobarbaz-cluster", "shipOpsRef": {"token": "generated"}})
        // createdClusterId = result.data.createShipOpsCluster.id;
        global.provider.verify();
        done();
      });
    });
  });

  // it("lists ship clusters for solo dev after creation", (done) => {
  //   global.provider.addInteraction(listClustersAfterCreatingShipInteraction).then(() => {
  //     const shipClient = ShipClientGQL(`http://localhost:${MOCK_SERVER_PORT}/graphql`, async () => { return createSessionToken("solo-account-session-1") }, fetch);
  //     shipClient.query({
  //       query: listClusters,
  //     })
  //     .then(result => {
  //       // const createdCluster = _.find(result.data.listClusters, {id: createdClusterId});
  //       // expect(createdCluster).to.not.be.null;

  //       // expect(createdCluster.title).to.equal("FooBarBaz Cluster");
  //       // expect(createdCluster.slug).to.equal("foobarbaz-cluster");
  //       // expect(createdCluster.gitOpsRef).to.be.null;
  //       // expect(createdCluster.shipOpsRef).to.be.deep.equal({"token": "generated"});

  //       global.provider.verify();
  //       done();
  //     });
  //   });
  // });
}
