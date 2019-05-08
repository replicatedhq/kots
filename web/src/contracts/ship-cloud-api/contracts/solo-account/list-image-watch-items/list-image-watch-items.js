import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import fetch from "node-fetch";
import { createSessionToken } from "../../../utils";
import { ShipClientGQL } from "../../../../../ShipClientGQL";
import { getImageWatch } from "../../../../../queries/ImageWatchQueries";
import { listImageWatchItemsInteraction } from "./interactions";

chai.use(chaiAsPromised);
const expect = chai.expect;

const MOCK_SERVER_PORT = 3333;

export default () => {
  beforeEach((done) => {
    global.provider.addInteraction(listImageWatchItemsInteraction).then(() => {
      done();
    })
  });

  it("lists image batch watches for solo dev", (done) => {
    const shipClient = ShipClientGQL(`http://localhost:${MOCK_SERVER_PORT}/graphql`, async () => { return createSessionToken("solo-account-session-1") }, fetch);
    shipClient.query({
      query: getImageWatch,
      variables: {
        batchId: "solo-account-image-batch-watch-1"
      }
    })
    .then(result => {
      expect(result.data.imageWatchItems).to.have.lengthOf(3);

      expect(result.data.imageWatchItems[0].id).to.equal("solo-account-image-watch-1");
      expect(result.data.imageWatchItems[0].name).to.equal("postgres:10.7,docker-pullable://postgres@sha256:810537dac6e7420c72a23b043b5dfe3ab493808e211f4ce69f7d1b7da1717cee");
      expect(result.data.imageWatchItems[0].isPrivate).to.be.false;
      expect(result.data.imageWatchItems[0].versionDetected).to.equal("10.7");
      expect(result.data.imageWatchItems[0].latestVersion).to.equal("11.2.0");
      expect(result.data.imageWatchItems[0].versionsBehind).to.equal(3);
      expect(JSON.parse(result.data.imageWatchItems[0].path)).to.deep.equal([
        {"sort": 3, "version": "11.2", "date": "2019-04-25T00:27:55.227279355Z"},
        {"sort": 2, "version": "11.1", "date": "2019-02-06T08:16:44.722701909Z"},
        {"sort": 1, "version": "11.0", "date": "2018-10-18T23:39:51.864511929Z"},
        {"sort": 0, "version": "10.7", "date": "2019-04-25T00:28:05.286902175Z"}
      ]);

      expect(result.data.imageWatchItems[1].id).to.equal("solo-account-image-watch-2");
      expect(result.data.imageWatchItems[1].name).to.equal("quay.io/kubernetes-ingress-controller/nginx-ingress-controller-amd64:0.22.0");
      expect(result.data.imageWatchItems[1].isPrivate).to.be.false;
      expect(result.data.imageWatchItems[1].versionDetected).to.equal("0.22.0");
      expect(result.data.imageWatchItems[1].latestVersion).to.equal("");
      expect(result.data.imageWatchItems[1].versionsBehind).to.equal(0);
      expect(result.data.imageWatchItems[1].path).to.equal("");

      expect(result.data.imageWatchItems[2].id).to.equal("solo-account-image-watch-3");
      expect(result.data.imageWatchItems[2].name).to.equal("localhost:32000/ship-cluster-worker:c7d3ee4@sha256:3af0e0a451dbc4c8a6d541e94ebbac59612f1c2fba7fec5a61f7dfc5ed9f343e");
      expect(result.data.imageWatchItems[2].isPrivate).to.be.true;
      expect(result.data.imageWatchItems[2].versionDetected).to.equal("c7d3ee4@sha256:3af0e0a451dbc4c8a6d541e94ebbac59612f1c2fba7fec5a61f7dfc5ed9f343e");
      expect(result.data.imageWatchItems[2].latestVersion).to.equal("");
      expect(result.data.imageWatchItems[2].versionsBehind).to.equal(0);
      expect(result.data.imageWatchItems[2].path).to.equal("");

      global.provider.verify();
      done();
    });
  });
}
