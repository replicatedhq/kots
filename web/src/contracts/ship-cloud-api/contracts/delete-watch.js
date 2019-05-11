import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import * as _ from "lodash";
import { getShipClient, createSessionToken } from "../utils";
import { deleteWatch } from "../../../mutations/WatchMutations";
import gql from "graphql-tag";
import * as Pact from "@pact-foundation/pact";
import { deleteWatchRaw } from "../../../mutations/WatchMutations";

chai.use(chaiAsPromised);
const expect = chai.expect;

export default () => {

  it("lists watches before one is deleted for single user", async (done) => {
    await global.provider.addInteraction(listWatchesInteraction);
    const result = await getShipClient("single-user-delete-watch-session-1").query({
      query: gql(`
        query listWatchesBeforeDeletion {
          listWatches {
            id
            watchName
          }
        }
      `)
    });

    expect(result.data.listWatches).to.have.lengthOf(2);
    expect(result.data.listWatches[0].id).to.equal("single-user-delete-watch-1");
    expect(result.data.listWatches[0].watchName).to.equal("Single User Save This Watch");

    expect(result.data.listWatches[1].id).to.equal("single-user-delete-watch-2");
    expect(result.data.listWatches[1].watchName).to.equal("Single User Delete This Watch");
    
    global.provider.verify().then(() => done());
  });
  
  it("deletes a watch for single user", async (done) => {
    await global.provider.addInteraction(deleteWatchInteraction);
    await getShipClient("single-user-delete-watch-session-1").mutate({
      mutation: deleteWatch,
      variables: {
        watchId: "single-user-delete-watch-2"
      }
    });
    global.provider.verify().then(() => done());
  });

  it("lists watches after one has been deleted for single user", async (done) => {
    await global.provider.addInteraction(listWatchesAfterDeletionInteraction);
    const result = await getShipClient("single-user-delete-watch-session-1").query({
      query: gql(`
        query listWatchesAfterDeletion {
          listWatches {
            id
            watchName
          }
        }
      `)
    })
    expect(result.data.listWatches).to.have.lengthOf(1);
    expect(result.data.listWatches[0].id).to.equal("single-user-delete-watch-1");
    expect(result.data.listWatches[0].watchName).to.equal("Single User Save This Watch");

    global.provider.verify().then(() => done());
  });
}

const listWatchesInteraction = new Pact.GraphQLInteraction()
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

const deleteWatchInteraction = new Pact.GraphQLInteraction()
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

  const listWatchesAfterDeletionInteraction = new Pact.GraphQLInteraction()
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