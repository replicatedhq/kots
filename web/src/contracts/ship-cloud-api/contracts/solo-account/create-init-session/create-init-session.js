import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import fetch from "node-fetch";
import * as _ from "lodash";
import { ShipClientGQL } from "../../../../../ShipClientGQL";
import { createSessionToken } from "../../../utils";

import { createInitSession } from "../../../../../mutations/WatchMutations";

import { createHelmInitSessionInteraction } from "./interactions";

chai.use(chaiAsPromised);
const expect = chai.expect;

const MOCK_SERVER_PORT = 3333;

export default () => {
  it("creates a midstream watch for solo dev", (done) => {
    global.provider.addInteraction(createHelmInitSessionInteraction).then(() => {
      const shipClient = ShipClientGQL(`http://localhost:${MOCK_SERVER_PORT}/graphql`, async () => { return createSessionToken("solo-account-session-1") }, fetch);
      shipClient.mutate({
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
