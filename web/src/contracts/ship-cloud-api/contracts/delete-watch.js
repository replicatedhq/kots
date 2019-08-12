import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { getShipClient, createSessionToken } from "../utils";
import { deleteWatch } from "../../../mutations/WatchMutations";
import gql from "graphql-tag";
import * as Pact from "@pact-foundation/pact";
import { deleteWatchRaw } from "../../../mutations/WatchMutations";

chai.use(chaiAsPromised);
const expect = chai.expect;

export default () => {

  it("lists apps before one is deleted for single user", async (done) => {
    await global.provider.addInteraction(listAppsInteraction);
    const result = await getShipClient("single-user-delete-watch-session-1").query({
      query: gql(`
        query listAppsBeforeDeletion {
          listApps {
            watches {
              id
              watchName
            }
          }
        }
      `)
    });

    expect(result.data.listApps.watches).to.have.lengthOf(2);
    expect(result.data.listApps.watches[0].id).to.equal("single-user-delete-watch-1");
    expect(result.data.listApps.watches[0].watchName).to.equal("Single User Save This Watch");

    expect(result.data.listApps.watches[1].id).to.equal("single-user-delete-watch-2");
    expect(result.data.listApps.watches[1].watchName).to.equal("Single User Delete This Watch");
    
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

  it("lists apps after one has been deleted for single user", async (done) => {
    await global.provider.addInteraction(listAppsAfterDeletionInteraction);
    const result = await getShipClient("single-user-delete-watch-session-1").query({
      query: gql(`
        query listAppsAfterDeletion {
          listApps {
            watches {
              id
              watchName
            }
          }
        }
      `)
    })
    expect(result.data.listApps.watches).to.have.lengthOf(1);
    expect(result.data.listApps.watches[0].id).to.equal("single-user-delete-watch-1");
    expect(result.data.listApps.watches[0].watchName).to.equal("Single User Save This Watch");

    global.provider.verify().then(() => done());
  });
}

const listAppsInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a query to list apps for a single user")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("single-user-delete-watch-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withQuery(`
    query listAppsBeforeDeletion {
      listApps {
        watches {
          id
          watchName
        }
      }
    }
  `)
  .withOperation("listAppsBeforeDeletion")
  .withVariables({})
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        listApps: {
          watches: [
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

  const listAppsAfterDeletionInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a query to list apps after a delete occurs for a single user")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("single-user-delete-watch-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withQuery(`
    query listAppsAfterDeletion {
      listApps {
        watches {
          id
          watchName
        }
      }
    }
  `)
  .withOperation("listAppsAfterDeletion")
  .withVariables({})
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        listApps: {
          watches: [
            {
              "id": "single-user-delete-watch-1",
              "watchName": "Single User Save This Watch"
            }
          ]
        }
      }
    }
  });