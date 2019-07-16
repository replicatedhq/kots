import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import chaiString from "chai-string";
import { getWatchVersion } from "../../../queries/WatchQueries";
import { getShipClient, createSessionToken } from "../utils";
import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { getWatchContributors, getWatchContributorsRaw } from "../../../queries/WatchQueries";

chai.use(chaiAsPromised);
chai.use(chaiString);
const expect = chai.expect;

export default () => {
  beforeEach((done) => {
    global.provider.removeInteractions().then(() => {
      done();
    });
  });

  it("gets a watch's contributors", (done) => {
    global.provider.addInteraction(getWatchContributorsInteraction).then(() => {
      done();
    });

    getShipClient("get-watch-contributors-session-1").query({
      query: getWatchContributors,
      variables: {
        id: "get-watch-contributors-watch"
      }
    })
      .then(result => {
        // expect(result.data.getWatchVersion.sequence).to.equal(1);
        // expect(result.data.getWatchVersion.title).to.equal("3.3.1");
        // expect(result.data.getWatchVersion.rendered).to.equal("downstream-output-1\n")

        global.provider.verify();
        done();
      });
  });
}

const getWatchContributorsInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a query to get a watch's contributors")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("get-watch-contributors-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withQuery(getWatchContributorsRaw)
  .withOperation("watchContributors")
  .withVariables({
    id: "get-watch-contributors-watch"
  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        watchContributors: [
          {
            "id": "get-watch-contributors-user",
            "createdAt": Pact.Matchers.like("Thu Apr 18 2019 12:34:56 GMT+0000 (UTC)"),
            "githubId": 1235,
            "login": "get-watch-contributors-username",
            "avatar_url": Pact.Matchers.like("https://avatars3.githubusercontent.com/u/234567?v=4")
          },
        ]
      },
    }
  });
