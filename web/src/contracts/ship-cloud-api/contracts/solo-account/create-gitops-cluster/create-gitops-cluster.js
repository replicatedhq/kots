import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import * as _ from "lodash";
import { getShipClient } from "../../../utils";
import { listClusters } from "../../../../../queries/ClusterQueries";
import { createGitOpsCluster } from "../../../../../mutations/ClusterMutations";
import { createGitOpsClusterInteraction } from "./interactions";
import { listClustersAfterCreatingGitOpsInteraction } from "../list-clusters/interactions";

chai.use(chaiAsPromised);
const expect = chai.expect;

const MOCK_SERVER_PORT = 3333;

export default () => {
  let createdClusterId;

  it("creates a gitops cluster for solo dev", (done) => {
    global.provider.addInteraction(createGitOpsClusterInteraction).then(() => {
      getShipClient("solo-account-session-1").mutate({
        mutation: createGitOpsCluster,
        variables: {
          title: "FooBarGit Cluster",
          installationId: 987654,
          gitOpsRef: {
            owner: "me",
            repo: "myself",
            branch: "i",
          },
        },
      })
      .then(result => {
        // expect(result.data.createGitOpsCluster).to.deep.equal({"id": "generated", "slug": "foobargit-cluster"})
        // createdClusterId = result.data.createGitOpsCluster.id;
        global.provider.verify();
        done();
      })
      .catch(err => {
        console.error(err);
      })
    });
  });

  // it("lists ship clusters for solo dev after creation", (done) => {
  //   global.provider.addInteraction(listClustersAfterCreatingGitOpsInteraction).then(() => {
  //     const shipClient = ShipClientGQL(`http://localhost:${MOCK_SERVER_PORT}/graphql`, async () => { return createSessionToken("solo-account-session-1") }, fetch);
  //     shipClient.query({
  //       query: listClusters,
  //     })
  //     .then(result => {
  //       // const createdCluster = _.find(result.data.listClusters, {id: createdClusterId});
  //       // expect(createdCluster).to.not.be.undefined;

  //       // expect(createdCluster.title).to.equal("FooBarGit Cluster");
  //       // expect(createdCluster.slug).to.equal("foobargit-cluster");
  //       // expect(createdCluster.gitOpsRef).to.deep.equal({"owner": "me", "repo": "myself", "branch": "i"});
  //       // expect(createdCluster.shipOpsRef).to.be.null;

  //       global.provider.verify();
  //       done();
  //     });
  //   });
  // });
}
