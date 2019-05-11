import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import * as _ from "lodash";
import { getShipClient } from "../../../utils";
import { listWatches } from "../../../../../queries/WatchQueries";
import { listWatchesInteraction } from "./interactions";

chai.use(chaiAsPromised);
const expect = chai.expect;

export default () => {
  beforeEach((done) => {
    global.provider.addInteraction(listWatchesInteraction).then(() => {
      done();
    })
  });

  it("lists watches for ship-cluster account", (done) => {
    getShipClient("ship-cluster-account-session-1").query({
      query: listWatches,
    })
    .then(result => {
      expect(result.data.listWatches).to.have.lengthOf(1);

      expect(result.data.listWatches[0].id).to.equal("better-db-midstream");
      expect(result.data.listWatches[0].watchName).to.equal("Better DB Midstream");
      expect(result.data.listWatches[0].slug).to.equal("ship-cluster-account/better-db-midstream");
      expect(result.data.listWatches[0].watches).to.have.length(1);

      const childWatch = result.data.listWatches[0].watches[0];

      expect(childWatch.id).to.equal("better-db-prod");
      expect(childWatch.watchName).to.equal("Better DB Ship 1");
      expect(childWatch.slug).to.equal("ship-cluster-account/better-db-prod");
      expect(childWatch.cluster.id).to.equal("ship-cluster-1");

      expect(childWatch.pastVersions).to.have.lengthOf(0);

      expect(childWatch.pendingVersions).to.have.lengthOf(1);
      expect(childWatch.pendingVersions[0].title).to.equal("0.1.4");
      expect(childWatch.pendingVersions[0].status).to.equal("pending");
      expect(childWatch.pendingVersions[0].sequence).to.equal(1);

      global.provider.verify();
      done();
    });
  });
}
