import * as Pact from "@pact-foundation/pact";
import { createSessionToken } from "../../../utils";
import { deleteWatchRaw } from "../../../../../mutations/WatchMutations";

export const listWatchesInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a query to list watches for a single user")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("single-user-delete-watch-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withQuery(`
    query listWatchesBeforeDeletion {
      listWatches {
        id
        watchName
      }
    }
  `)
  .withOperation("listWatchesBeforeDeletion")
  .withVariables({})
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        listWatches: [
          {
            "id": "single-user-delete-watch-1",
            "watchName": "Single User Save This Watch"
          },
          {
            "id": "single-user-delete-watch-2",
            "watchName": "Single User Delete This Watch"
          }
        ]
      }
    }
  });

export const deleteWatchInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a mutation to delete a watch for single user")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("single-user-delete-watch-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withOperation("deleteWatch")
  .withQuery(deleteWatchRaw)
  .withVariables({
    watchId: "single-user-delete-watch-2",
  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        deleteWatch: true
      },
    },
  });

  export const listWatchesAfterDeletionInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a query to list watches after a delete occurs for a single user")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("single-user-delete-watch-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withQuery(`
    query listWatchesAfterDeletion {
      listWatches {
        id
        watchName
      }
    }
  `)
  .withOperation("listWatchesAfterDeletion")
  .withVariables({})
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        listWatches: [
          {
            "id": "single-user-delete-watch-1",
            "watchName": "Single User Save This Watch"
          }
        ]
      }
    }
  });