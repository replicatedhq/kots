import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import * as _ from "lodash";
import { getShipClient } from "../../../utils";
import { deleteWatch } from "../../../../../mutations/WatchMutations";
import { listWatchesInteraction, deleteWatchInteraction, listWatchesAfterDeletionInteraction } from "./interactions";
import gql from "graphql-tag";

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
