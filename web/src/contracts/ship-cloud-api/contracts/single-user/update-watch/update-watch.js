import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import fetch from "node-fetch";
import * as _ from "lodash";
import { ShipClientGQL } from "../../../../../ShipClientGQL";
import { createSessionToken } from "../../../utils";

import { updateWatch } from "../../../../../mutations/WatchMutations";

import { updateWatchInteraction } from "./interactions";

chai.use(chaiAsPromised);
const expect = chai.expect;

const MOCK_SERVER_PORT = 3333;

export default () => {
  it("updates a watch for single user", (done) => {
    global.provider.addInteraction(updateWatchInteraction).then(() => {
      const shipClient = ShipClientGQL(`http://localhost:${MOCK_SERVER_PORT}/graphql`, async () => { return createSessionToken("single-user-account-session-1") }, fetch);
      shipClient.mutate({
        mutation: updateWatch,
        variables: {
            watchId: "single-user-watch-update-1",
            watchName: "Updated Single User Watch Update",
            iconUri: "http://ccsuppliersource.com/wp-content/uploads/2018/12/bigstock_online_update_11303201.jpg"
        },
      })
      .then(result => {
        expect(result.data.updateWatch.id).to.equal("single-user-watch-update-1");
        expect(result.data.updateWatch.watchName).to.equal("Updated Single User Watch Update");
        expect(result.data.updateWatch.watchIcon).to.equal("http://ccsuppliersource.com/wp-content/uploads/2018/12/bigstock_online_update_11303201.jpg");
        global.provider.verify();
        done();
      });
    });
  });
}
