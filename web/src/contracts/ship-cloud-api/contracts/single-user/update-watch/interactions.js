import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { createSessionToken } from "../../../utils";
import { updateWatchRaw } from "../../../../../mutations/WatchMutations";

export const updateWatchInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a mutation to update a watch for single user")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("single-user-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withOperation("updateWatch")
  .withQuery(updateWatchRaw)
  .withVariables({
    watchId: "single-user-watch-update-1",
    watchName: "Updated Single User Watch Update",
    iconUri: "http://ccsuppliersource.com/wp-content/uploads/2018/12/bigstock_online_update_11303201.jpg"
  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        updateWatch: {
          id: "single-user-watch-update-1",
          slug: "single-user/single-user-watch-update-1",
          watchName: "Updated Single User Watch Update",
          watchIcon: "http://ccsuppliersource.com/wp-content/uploads/2018/12/bigstock_online_update_11303201.jpg",
          createdOn: Matchers.like("2019-04-10 12:34:56.789"),
          lastUpdated: Matchers.like("generated"),
          stateJSON: Matchers.like("string"),
          contributors: Matchers.like([{
            avatar_url: "string",
            createdAt: "0001-01-01T00:00:00Z",
            githubId: 1234,
            id: "string",
            login: "string",
          }])
        },
      },
    },
  });