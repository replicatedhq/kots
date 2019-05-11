import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import * as _ from "lodash";
import { getShipClient } from "../../../utils";

import { createInitSession } from "../../../../../mutations/WatchMutations";

import { createHelmInitSessionInteraction } from "./interactions";

chai.use(chaiAsPromised);

export default () => {
  it("creates an init session for solo dev", async (done) => {
    await global.provider.addInteraction(createHelmInitSessionInteraction);
    const result = await getShipClient("solo-account-session-1").mutate({
      mutation: createInitSession,
      variables: {
        upstreamUri: "https://github.com/helm/charts/stable/grafana",
        clusterID: null,
        githubPath: null,
      },
    });
    global.provider.verify().then(() => done());
    });
}
