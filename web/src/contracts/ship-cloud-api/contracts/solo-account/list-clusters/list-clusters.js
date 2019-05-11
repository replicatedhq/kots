import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { getShipClient } from "../../../utils";
import { listClusters } from "../../../../../queries/ClusterQueries";
import { listClustersInteraction } from "./interactions";

chai.use(chaiAsPromised);
const expect = chai.expect;

export default () => {

  it("lists ship clusters for solo dev", async (done) => {
    await global.provider.addInteraction(listClustersInteraction);
    const result = await getShipClient("solo-account-session-1").query({
      query: listClusters,
    });
    expect(result.data.listClusters).to.have.lengthOf(2);

    expect(result.data.listClusters[0].id).to.equal("solo-account-cluster-1");
    expect(result.data.listClusters[0].title).to.equal("Solo Cluster");
    expect(result.data.listClusters[0].slug).to.equal("solo-cluster");
    expect(result.data.listClusters[0].gitOpsRef).to.be.null;
    expect(result.data.listClusters[0].shipOpsRef).to.deep.equal({"token": "solo-account-cluster-token"});

    expect(result.data.listClusters[1].id).to.equal("solo-account-cluster-2");
    expect(result.data.listClusters[1].title).to.equal("Solo GitHub Cluster");
    expect(result.data.listClusters[1].slug).to.equal("solo-cluster-2");
    expect(result.data.listClusters[1].gitOpsRef).to.deep.equal({"owner": "lonely-github-dev", "repo": "gitops-deploy", "branch": "master"});
    expect(result.data.listClusters[1].shipOpsRef).to.be.null;

    global.provider.verify().then(() => done());
  });
}
