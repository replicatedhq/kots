import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import chaiString from "chai-string";
import { getWatchVersion } from "../../../queries/WatchQueries";
import { getShipClient, createSessionToken } from "../utils";
import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { getWatchVersionRaw } from "../../../queries/WatchQueries";

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

const getWatchVersionCurrentGitopsInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a query to get the current watch version for gitops account")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("gitops-cluster-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withQuery(getWatchVersionRaw)
  .withOperation("getWatchVersion")
  .withVariables({
    id: "gitops-grafana-downstream",
    sequence: 1,
  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        getWatchVersion: {
          title: "3.3.1",
          status: "merged",
          createdOn: Matchers.like("2019-04-10 12:34:56.789"),
          sequence: 1,
          pullrequestNumber: 89,
          rendered: "downstream-output-1\n"
        },
      },
    }
  });

export const getWatchVersionNextGitopsInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a query to get the next watch version for gitops account")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("gitops-cluster-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withQuery(getWatchVersionRaw)
  .withOperation("getWatchVersion")
  .withVariables({
    id: "gitops-grafana-downstream",
    sequence: 2,
  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        getWatchVersion: {
          title: "3.3.2",
          status: "open",
          createdOn: Matchers.like("2019-04-10 12:34:56.789"),
          sequence: 2,
          pullrequestNumber: 90,
          rendered: "downstream-output-2\n"
        },
      },
    }
  });
