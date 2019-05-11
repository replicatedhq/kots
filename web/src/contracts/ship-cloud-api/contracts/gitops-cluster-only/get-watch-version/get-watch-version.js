import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import chaiString from "chai-string";
import { getWatchVersion } from "../../../../../queries/WatchQueries";
import { getWatchVersionCurrentGitopsInteraction, getWatchVersionNextGitopsInteraction } from "./interactions";
import { getShipClient } from "../../../utils";

chai.use(chaiAsPromised);
chai.use(chaiString);
const expect = chai.expect;

export default () => {
  beforeEach((done) => {
    global.provider.removeInteractions().then(() => {
      done();
    });
  });

  it("get the current watch version for gitops dev", (done) => {
    global.provider.addInteraction(getWatchVersionCurrentGitopsInteraction).then(() => {
      done();
    });

    getShipClient("gitops-cluster-account-session-1").query({
      query: getWatchVersion,
      variables: {
        id: "gitops-grafana-downstream",
        sequence: 1,
      }
    })
    .then(result => {
      expect(result.data.getWatchVersion.sequence).to.equal(1);
      expect(result.data.getWatchVersion.title).to.equal("3.3.1");
      expect(result.data.getWatchVersion.rendered).to.equal("downstream-output-1\n")

      global.provider.verify();
      done();
    });
  });

  it("gets the next watch version for gitops dev", (done) => {
    global.provider.addInteraction(getWatchVersionNextGitopsInteraction).then(() => {
      done();
    });

    getShipClient("gitops-cluster-account-session-1").query({
      query: getWatchVersion,
      variables: {
        id: "gitops-grafana-downstream",
        sequence: 2,
      }
    })
    .then(result => {
      expect(result.data.getWatchVersion.sequence).to.equal(2);
      expect(result.data.getWatchVersion.title).to.equal("3.3.2");
      expect(result.data.getWatchVersion.rendered).to.equal("downstream-output-2\n")

      global.provider.verify();
      done();
    });
  });
}
