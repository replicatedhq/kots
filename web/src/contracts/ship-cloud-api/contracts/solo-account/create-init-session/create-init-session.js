import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import * as _ from "lodash";
import { getShipClient } from "../../../utils";

import { createInitSession } from "../../../../../mutations/WatchMutations";

import { createHelmInitSessionInteraction } from "./interactions";

chai.use(chaiAsPromised);
const expect = chai.expect;

const MOCK_SERVER_PORT = 3333;

export default () => {
  it("creates a midstream watch for solo dev", (done) => {
    global.provider.addInteraction(createHelmInitSessionInteraction).then(() => {
      getShipClient("solo-account-session-1").mutate({
        mutation: createInitSession,
        variables: {
          upstreamUri: "https://github.com/helm/charts/stable/grafana",
          clusterID: null,
          githubPath: null,
        },
      })
      .then(result => {
        global.provider.verify();
        done();
      });
    });
  });
}
