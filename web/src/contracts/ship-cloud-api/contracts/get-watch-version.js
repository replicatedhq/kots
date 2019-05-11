import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import chaiString from "chai-string";
import { getShipClient, createSessionToken } from "../utils";
import { getWatchVersion } from "../../../queries/WatchQueries";
import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { getWatchVersionRaw } from "../../../queries/WatchQueries";

chai.use(chaiAsPromised);
chai.use(chaiString);
const expect = chai.expect;

export default () => {

  it("get a single watch version for solo dev", async (done) => {
    await global.provider.addInteraction(getWatchVersionInteraction);
    const result = await getShipClient("solo-account-session-1").query({
      query: getWatchVersion,
      variables: {
        id: "solo-account-watch-1",
        sequence: 0,
      }
    });
    expect(result.data.getWatchVersion.sequence).to.equal(0);
    expect(result.data.getWatchVersion.rendered).to.equalIgnoreSpaces(`apiVersion: v1\ndata:\n  factorio-password: eW91ci5wYXNzd29yZA==\n  factorio-username: eW91ci51c2VybmFtZQ==\n  rcon-password: \"\"\n  server-password: \"\"\nkind: Secret\nmetadata:\n labels:\n    app: factorio-factorio\n    release: factorio\n  name: factorio-factorio\ntype: Opaque\n---\napiVersion: v1\nkind: Service\nmetadata:\n  labels:\n    app: factorio-factorio\n    release: factorio\n  name: factorio-factorio\nspec:\n  ports:\n  - name: factorio\n    port: 34197\n    protocol: UDP\ntargetPort: factorio\n  selector:\n    app: factorio-factorio\n  type: LoadBalancer\n---\napiVersion: extensions/v1beta1\nkind: Deployment\nmetadata:\n  labels:\n    app: factorio-factorio\n    release: factorio\n  name: factorio-factorio\nspec:\n  template:\n    metadata:\n      labels:\n        app: factorio-factorio\n    spec:\n      containers:\n      - env:\n        - name: FACTORIO_SERVER_NAME\n          value: Kubernetes Server\n        - name: FACTORIO_DESCRIPTION\n          value: Factorio running on Kubernetes\n        - name: FACTORIO_PORT\n          value: \"34197\"\n        - name: FACTORIO_MAX_PLAYERS\n    value: \"255\"\n        - name: FACTORIO_IS_PUBLIC\n          value: \"false\"\n        - name: FACTORIO_REQUIRE_USER_VERIFICATION\n          value: \"false\"\n        - name: FACTORIO_ALLOW_COMMANDS\n          value: admins-only\n        - name: FACTORIO_NO_AUTO_PAUSE\n          value: \"false\"\n        - name: FACTORIO_AUTOSAVE_INTERVAL\nvalue: \"2\"\n        - name: FACTORIO_AUTOSAVE_SLOTS\n          value: \"3\"\n        image: quay.io/games_on_k8s/factorio:0.14.22\n        imagePullPolicy: Always\n        name: factorio-factorio\n        ports:\n        - containerPort: 34197\n          name: factorio\n          protocol: UDP\n        resources:\n          requests:\n            cpu: 500m\n            memory: 512Mi\n        volumeMounts:\n- mountPath: /opt/factorio/saves\n          name: saves\n        - mountPath: /opt/factorio/mods\n   name: mods\n      volumes:\n      - name: saves\n        persistentVolumeClaim:\n          claimName:factorio-factorio-savedgames\n      - emptyDir: {}\n        name: mods\n---\napiVersion: v1\nkind: PersistentVolumeClaim\nmetadata:\n  labels:\n    app: factorio-factorio\n    release: factorio\n  name: factorio-factorio-savedgames\nspec:\n  accessModes:\n  - ReadWriteOnce\n  resources:\n    requests:\n      storage: 1Gi\n`);
    global.provider.verify().then(() => done());
  });
}

const getWatchVersionInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a query to get a single watch version for solo account")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("solo-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withQuery(getWatchVersionRaw)
  .withOperation("getWatchVersion")
  .withVariables({
    id: "solo-account-watch-1",
    sequence: 0,
  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        getWatchVersion: {
          title: Matchers.like("string"),
          status: "deployed",
          createdOn: Matchers.like("2019-04-10 12:34:56.789"),
          sequence: 0,
          pullrequestNumber: null,
          rendered: Matchers.like(`apiVersion: v1\ndata:\n  factorio-password: eW91ci5wYXNzd29yZA==\n  factorio-username: eW91ci51c2VybmFtZQ==\n  rcon-password: \"\"\n  server-password: \"\"\nkind: Secret\nmetadata:\n labels:\n    app: factorio-factorio\n    release: factorio\n  name: factorio-factorio\ntype: Opaque\n---\napiVersion: v1\nkind: Service\nmetadata:\n  labels:\n    app: factorio-factorio\n    release: factorio\n  name: factorio-factorio\nspec:\n  ports:\n  - name: factorio\n    port: 34197\n    protocol: UDP\ntargetPort: factorio\n  selector:\n    app: factorio-factorio\n  type: LoadBalancer\n---\napiVersion: extensions/v1beta1\nkind: Deployment\nmetadata:\n  labels:\n    app: factorio-factorio\n    release: factorio\n  name: factorio-factorio\nspec:\n  template:\n    metadata:\n      labels:\n        app: factorio-factorio\n    spec:\n      containers:\n      - env:\n        - name: FACTORIO_SERVER_NAME\n          value: Kubernetes Server\n        - name: FACTORIO_DESCRIPTION\n          value: Factorio running on Kubernetes\n        - name: FACTORIO_PORT\n          value: \"34197\"\n        - name: FACTORIO_MAX_PLAYERS\n    value: \"255\"\n        - name: FACTORIO_IS_PUBLIC\n          value: \"false\"\n        - name: FACTORIO_REQUIRE_USER_VERIFICATION\n          value: \"false\"\n        - name: FACTORIO_ALLOW_COMMANDS\n          value: admins-only\n        - name: FACTORIO_NO_AUTO_PAUSE\n          value: \"false\"\n        - name: FACTORIO_AUTOSAVE_INTERVAL\nvalue: \"2\"\n        - name: FACTORIO_AUTOSAVE_SLOTS\n          value: \"3\"\n        image: quay.io/games_on_k8s/factorio:0.14.22\n        imagePullPolicy: Always\n        name: factorio-factorio\n        ports:\n        - containerPort: 34197\n          name: factorio\n          protocol: UDP\n        resources:\n          requests:\n            cpu: 500m\n            memory: 512Mi\n        volumeMounts:\n- mountPath: /opt/factorio/saves\n          name: saves\n        - mountPath: /opt/factorio/mods\n   name: mods\n      volumes:\n      - name: saves\n        persistentVolumeClaim:\n          claimName:factorio-factorio-savedgames\n      - emptyDir: {}\n        name: mods\n---\napiVersion: v1\nkind: PersistentVolumeClaim\nmetadata:\n  labels:\n    app: factorio-factorio\n    release: factorio\n  name: factorio-factorio-savedgames\nspec:\n  accessModes:\n  - ReadWriteOnce\n  resources:\n    requests:\n      storage: 1Gi\n`),
        },
      },
    }
  });
