import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { getShipClient, createSessionToken } from "../utils";
import { listClusters } from "../../../queries/ClusterQueries";
import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { listClustersRaw } from "../../../queries/ClusterQueries";

chai.use(chaiAsPromised);
const expect = chai.expect;

export default () => {
  it("lists ship clusters for solo dev", async (done) => {
    await global.provider.addInteraction(listClustersInteraction);
    const result = await getShipClient("solo-account-session-1").query({
      query: listClusters,
    });
    expect(result.data.listClusters).to.have.lengthOf(3);

    expect(result.data.listClusters[0].id).to.equal("solo-account-cluster-1");
    expect(result.data.listClusters[0].title).to.equal("Solo Cluster");
    expect(result.data.listClusters[0].slug).to.equal("solo-cluster");
    expect(result.data.listClusters[0].shipOpsRef).to.deep.equal({"token": "solo-account-cluster-token"});

    expect(result.data.listClusters[1].id).to.equal("solo-account-cluster-2");
    expect(result.data.listClusters[1].title).to.equal("Solo GitHub Cluster");
    expect(result.data.listClusters[1].slug).to.equal("solo-cluster-2");
    expect(result.data.listClusters[1].shipOpsRef).to.be.null;

    expect(result.data.listClusters[2].title).to.equal("this-cluster");
    expect(result.data.listClusters[2].slug).to.equal("this-cluster");

    global.provider.verify().then(() => done());
  });
}

const listClustersInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a query to list clusters for solo account")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("solo-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withQuery(listClustersRaw)
  .withOperation("listClusters")
  .withVariables({

  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        listClusters: [
          {
            "id": "solo-account-cluster-1",
            "title": "Solo Cluster",
            "slug": "solo-cluster",
            "totalApplicationCount": 1,
            "createdOn": Matchers.like("2019-04-10 12:34:56.789"),
            "lastUpdated": Matchers.like("2019-04-11 01:23:45.567"),
            "shipOpsRef": {
              "token": "solo-account-cluster-token",
            },
          },
          {
            "id": "solo-account-cluster-2",
            "title": "Solo GitHub Cluster",
            "slug": "solo-cluster-2",
            "totalApplicationCount": 0,
            "createdOn": Matchers.like("2019-04-10 12:34:56.789"),
            "lastUpdated": Matchers.like("2019-04-11 01:23:45.567"),
            "shipOpsRef": null,
          },
          {
            "id": Matchers.like("l9cpydvcf30ekf25pmokwn6qlnxscrci"),
            "title": "this-cluster",
            "slug": "this-cluster",
            "createdOn": Matchers.like("Wed Jan 22 2020 00:34:12 GMT+0000 (Coordinated Universal Time)"),
            "lastUpdated": null,
            "shipOpsRef": {
              "token": Matchers.like("g7yoarinjpaft3si5z1h5de1uam443bb")
            },
            "totalApplicationCount": 0
          }
        ]
      }
    }
  });
