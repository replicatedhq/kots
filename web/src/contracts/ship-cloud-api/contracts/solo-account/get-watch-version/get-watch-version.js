import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import chaiString from "chai-string";
import fetch from "node-fetch";
import { createSessionToken } from "../../../utils";
import { ShipClientGQL } from "../../../../../ShipClientGQL";
import { getWatchVersion } from "../../../../../queries/WatchQueries";
import { getWatchVersionInteraction } from "./interactions";

chai.use(chaiAsPromised);
chai.use(chaiString);
const expect = chai.expect;

const MOCK_SERVER_PORT = 3333;

export default () => {
  beforeEach((done) => {
    global.provider.addInteraction(getWatchVersionInteraction).then(() => {
      done();
    })
  });

  it("get a single watch version for solo dev", (done) => {
    const shipClient = ShipClientGQL(`http://localhost:${MOCK_SERVER_PORT}/graphql`, async () => { return createSessionToken("solo-account-session-1") }, fetch);
    shipClient.query({
      query: getWatchVersion,
      variables: {
        id: "solo-account-watch-1",
        sequence: 0,
      }
    })
    .then(result => {
      expect(result.data.getWatchVersion.sequence).to.equal(0);
      expect(result.data.getWatchVersion.rendered).to.equalIgnoreSpaces(`apiVersion: v1\ndata:\n  factorio-password: eW91ci5wYXNzd29yZA==\n  factorio-username: eW91ci51c2VybmFtZQ==\n  rcon-password: \"\"\n  server-password: \"\"\nkind: Secret\nmetadata:\n labels:\n    app: factorio-factorio\n    release: factorio\n  name: factorio-factorio\ntype: Opaque\n---\napiVersion: v1\nkind: Service\nmetadata:\n  labels:\n    app: factorio-factorio\n    release: factorio\n  name: factorio-factorio\nspec:\n  ports:\n  - name: factorio\n    port: 34197\n    protocol: UDP\ntargetPort: factorio\n  selector:\n    app: factorio-factorio\n  type: LoadBalancer\n---\napiVersion: extensions/v1beta1\nkind: Deployment\nmetadata:\n  labels:\n    app: factorio-factorio\n    release: factorio\n  name: factorio-factorio\nspec:\n  template:\n    metadata:\n      labels:\n        app: factorio-factorio\n    spec:\n      containers:\n      - env:\n        - name: FACTORIO_SERVER_NAME\n          value: Kubernetes Server\n        - name: FACTORIO_DESCRIPTION\n          value: Factorio running on Kubernetes\n        - name: FACTORIO_PORT\n          value: \"34197\"\n        - name: FACTORIO_MAX_PLAYERS\n    value: \"255\"\n        - name: FACTORIO_IS_PUBLIC\n          value: \"false\"\n        - name: FACTORIO_REQUIRE_USER_VERIFICATION\n          value: \"false\"\n        - name: FACTORIO_ALLOW_COMMANDS\n          value: admins-only\n        - name: FACTORIO_NO_AUTO_PAUSE\n          value: \"false\"\n        - name: FACTORIO_AUTOSAVE_INTERVAL\nvalue: \"2\"\n        - name: FACTORIO_AUTOSAVE_SLOTS\n          value: \"3\"\n        image: quay.io/games_on_k8s/factorio:0.14.22\n        imagePullPolicy: Always\n        name: factorio-factorio\n        ports:\n        - containerPort: 34197\n          name: factorio\n          protocol: UDP\n        resources:\n          requests:\n            cpu: 500m\n            memory: 512Mi\n        volumeMounts:\n- mountPath: /opt/factorio/saves\n          name: saves\n        - mountPath: /opt/factorio/mods\n   name: mods\n      volumes:\n      - name: saves\n        persistentVolumeClaim:\n          claimName:factorio-factorio-savedgames\n      - emptyDir: {}\n        name: mods\n---\napiVersion: v1\nkind: PersistentVolumeClaim\nmetadata:\n  labels:\n    app: factorio-factorio\n    release: factorio\n  name: factorio-factorio-savedgames\nspec:\n  accessModes:\n  - ReadWriteOnce\n  resources:\n    requests:\n      storage: 1Gi\n`);

      global.provider.verify();
      done();
    });
  });
}
