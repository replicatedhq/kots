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

  it("gets a watch's contributors", done => {
    global.provider.addInteraction(getWatchContributorsInteraction).then(() => {
      getShipClient("get-watch-contributors-session-1").query({
        query: getWatchContributors,
        variables: {
          id: "get-watch-contributors-watch"
        }
      }).then(result => {
        const [user] = result.data.watchContributors;

        expect(user.id).to.equal("get-watch-contributors-user");
        expect(user.createdAt).to.equal("Thu Apr 18 2019 12:34:56 GMT+0000 (UTC)");
        expect(user.githubId).to.equal(1235);
        expect(user.login).to.equal("get-watch-contributors-username");
        expect(user.avatar_url).to.equal("https://avatars3.githubusercontent.com/u/234567?v=4");

        global.provider.verify();
        done();
      });
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
            "createdAt": "Thu Apr 18 2019 12:34:56 GMT+0000 (UTC)",
            "githubId": 1235,
            "login": "get-watch-contributors-username",
            "avatar_url": "https://avatars3.githubusercontent.com/u/234567?v=4"
          },
        ]
      },
    }
  });
