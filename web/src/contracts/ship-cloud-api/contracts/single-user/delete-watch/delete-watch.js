import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import fetch from "node-fetch";
import * as _ from "lodash";
import { ShipClientGQL } from "../../../../../ShipClientGQL";
import { createSessionToken } from "../../../utils";

import { deleteWatch } from "../../../../../mutations/WatchMutations";

import { listWatchesInteraction, deleteWatchInteraction, listWatchesAfterDeletionInteraction } from "./interactions";
import gql from "graphql-tag";

chai.use(chaiAsPromised);
const expect = chai.expect;

const MOCK_SERVER_PORT = 3333;

export default () => {

    afterEach(() => global.provider.verify());

    it("lists watches and then deletes one for single user", (done) => {
    global.provider.addInteraction(listWatchesInteraction).then(() => {
        const shipClient = ShipClientGQL(`http://localhost:${MOCK_SERVER_PORT}/graphql`, async () => { return createSessionToken("single-user-delete-watch-session-1") }, fetch);
        shipClient.query({
            query: gql(`
            query listWatchesBeforeDeletion {
                listWatches {
                id
                watchName
                }
            }
            `)
        })
        .then(result => {
            expect(result.data.listWatches).to.have.lengthOf(2);
            expect(result.data.listWatches[0].id).to.equal("single-user-delete-watch-1");
            expect(result.data.listWatches[0].watchName).to.equal("Single User Save This Watch");

            expect(result.data.listWatches[1].id).to.equal("single-user-delete-watch-2");
            expect(result.data.listWatches[1].watchName).to.equal("Single User Delete This Watch");
            done();
        });
    });

    global.provider.addInteraction(deleteWatchInteraction).then(() => {
        const shipClient = ShipClientGQL(`http://localhost:${MOCK_SERVER_PORT}/graphql`, async () => { return createSessionToken("single-user-delete-watch-session-1") }, fetch);
        shipClient.mutate({
            mutation: deleteWatch,
            variables: {
                watchId: "single-user-delete-watch-2"
            }
        })
        .then(result => {
            done();
        });
    });

    global.provider.addInteraction(listWatchesAfterDeletionInteraction).then(() => {
        const shipClient = ShipClientGQL(`http://localhost:${MOCK_SERVER_PORT}/graphql`, async () => { return createSessionToken("single-user-delete-watch-session-1") }, fetch);
        shipClient.query({
            query: gql(`
            query listWatchesAfterDeletion {
                listWatches {
                id
                watchName
                }
            }
            `)
        })
        .then(result => {
            expect(result.data.listWatches).to.have.lengthOf(1);
            expect(result.data.listWatches[0].id).to.equal("single-user-delete-watch-1");
            expect(result.data.listWatches[0].watchName).to.equal("Single User Save This Watch");
            done();
        });
    });
    });
}
