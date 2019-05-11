import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import * as _ from "lodash";
import { getShipClient } from "../../../utils";

import { updateWatch } from "../../../../../mutations/WatchMutations";

import { updateWatchInteraction } from "./interactions";

chai.use(chaiAsPromised);
const expect = chai.expect;

export default () => {
  it("updates a watch for single user", async (done) => {
    await global.provider.addInteraction(updateWatchInteraction);
    const result = await getShipClient("single-user-account-session-1").mutate({
      mutation: updateWatch,
      variables: {
          watchId: "single-user-watch-update-1",
          watchName: "Updated Single User Watch Update",
          iconUri: "http://ccsuppliersource.com/wp-content/uploads/2018/12/bigstock_online_update_11303201.jpg"
      },
    });
    expect(result.data.updateWatch.id).to.equal("single-user-watch-update-1");
    expect(result.data.updateWatch.watchName).to.equal("Updated Single User Watch Update");
    expect(result.data.updateWatch.watchIcon).to.equal("http://ccsuppliersource.com/wp-content/uploads/2018/12/bigstock_online_update_11303201.jpg");
    global.provider.verify().then(() => done());
  });
}
