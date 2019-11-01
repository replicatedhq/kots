import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { getShipClient, createSessionToken } from "../utils";
import { getImageWatch } from "../../../queries/ImageWatchQueries";
import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { getImageWatchRaw } from "../../../queries/ImageWatchQueries";

chai.use(chaiAsPromised);
const expect = chai.expect;

export default () => {

  it("lists image batch watches for solo dev", async (done) => {
    await global.provider.addInteraction(listImageWatchesInteraction);
    const result = await getShipClient("solo-account-session-1").query({
      query: getImageWatch,
      variables: {
        batchId: "solo-account-image-batch-watch-1"
      }
    });
    expect(result.data.imageWatches).to.have.lengthOf(3);

    expect(result.data.imageWatches[0].id).to.equal("solo-account-image-watch-1");
    expect(result.data.imageWatches[0].name).to.equal("postgres:10.7,docker-pullable://postgres@sha256:810537dac6e7420c72a23b043b5dfe3ab493808e211f4ce69f7d1b7da1717cee");
    expect(result.data.imageWatches[0].isPrivate).to.be.false;
    expect(result.data.imageWatches[0].versionDetected).to.equal("10.7");
    expect(result.data.imageWatches[0].latestVersion).to.equal("11.2.0");
    expect(result.data.imageWatches[0].versionsBehind).to.equal(3);
    expect(JSON.parse(result.data.imageWatches[0].path)).to.deep.equal([
      {"sort": 3, "version": "11.2", "date": "2019-04-25T00:27:55.227279355Z"},
      {"sort": 2, "version": "11.1", "date": "2019-02-06T08:16:44.722701909Z"},
      {"sort": 1, "version": "11.0", "date": "2018-10-18T23:39:51.864511929Z"},
      {"sort": 0, "version": "10.7", "date": "2019-04-25T00:28:05.286902175Z"}
    ]);

    expect(result.data.imageWatches[1].id).to.equal("solo-account-image-watch-2");
    expect(result.data.imageWatches[1].name).to.equal("quay.io/kubernetes-ingress-controller/nginx-ingress-controller-amd64:0.22.0");
    expect(result.data.imageWatches[1].isPrivate).to.be.false;
    expect(result.data.imageWatches[1].versionDetected).to.equal("0.22.0");
    expect(result.data.imageWatches[1].latestVersion).to.equal("");
    expect(result.data.imageWatches[1].versionsBehind).to.equal(0);
    expect(result.data.imageWatches[1].path).to.equal("");

    expect(result.data.imageWatches[2].id).to.equal("solo-account-image-watch-3");
    expect(result.data.imageWatches[2].name).to.equal("localhost:32000/kotsadm-worker:c7d3ee4@sha256:3af0e0a451dbc4c8a6d541e94ebbac59612f1c2fba7fec5a61f7dfc5ed9f343e");
    expect(result.data.imageWatches[2].isPrivate).to.be.true;
    expect(result.data.imageWatches[2].versionDetected).to.equal("c7d3ee4@sha256:3af0e0a451dbc4c8a6d541e94ebbac59612f1c2fba7fec5a61f7dfc5ed9f343e");
    expect(result.data.imageWatches[2].latestVersion).to.equal("");
    expect(result.data.imageWatches[2].versionsBehind).to.equal(0);
    expect(result.data.imageWatches[2].path).to.equal("");

    global.provider.verify().then(() => done());
  });
}

export const listImageWatchesInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a query to list image watches from a cluster for solo account")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("solo-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withQuery(getImageWatchRaw)
  .withOperation("imageWatches")
  .withVariables({
    batchId: "solo-account-image-batch-watch-1"
  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        imageWatches: [
          {
            "id": "solo-account-image-watch-1",
            "name": "postgres:10.7,docker-pullable://postgres@sha256:810537dac6e7420c72a23b043b5dfe3ab493808e211f4ce69f7d1b7da1717cee",
            "lastCheckedOn": Matchers.like("Tue May 07 2019 22:43:05 GMT+0000 (Coordinated Universal Time)"),
            "isPrivate": false,
            "versionDetected": "10.7",
            "latestVersion": "11.2.0",
            "compatibleVersion": "",
            "versionsBehind": 3,
            "path": "[{\"sort\":3,\"version\":\"11.2\",\"date\":\"2019-04-25T00:27:55.227279355Z\"},{\"sort\":2,\"version\":\"11.1\",\"date\":\"2019-02-06T08:16:44.722701909Z\"},{\"sort\":1,\"version\":\"11.0\",\"date\":\"2018-10-18T23:39:51.864511929Z\"},{\"sort\":0,\"version\":\"10.7\",\"date\":\"2019-04-25T00:28:05.286902175Z\"}]"
          },
          {
            "id": "solo-account-image-watch-2",
            "name": "quay.io/kubernetes-ingress-controller/nginx-ingress-controller-amd64:0.22.0",
            "lastCheckedOn": Matchers.like("Tue May 07 2019 22:43:05 GMT+0000 (Coordinated Universal Time)"),
            "isPrivate": false,
            "versionDetected": "0.22.0",
            "latestVersion": "",
            "compatibleVersion": "",
            "versionsBehind": 0,
            "path": ""
          },
          {
            "id": "solo-account-image-watch-3",
            "name": "localhost:32000/kotsadm-worker:c7d3ee4@sha256:3af0e0a451dbc4c8a6d541e94ebbac59612f1c2fba7fec5a61f7dfc5ed9f343e",
            "lastCheckedOn": Matchers.like("Tue May 07 2019 22:43:05 GMT+0000 (Coordinated Universal Time)"),
            "isPrivate": true,
            "versionDetected": "c7d3ee4@sha256:3af0e0a451dbc4c8a6d541e94ebbac59612f1c2fba7fec5a61f7dfc5ed9f343e",
            "latestVersion": "",
            "compatibleVersion": "",
            "versionsBehind": 0,
            "path": ""
          }
      ]
      }
    }
  });
