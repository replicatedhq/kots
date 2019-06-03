import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { getShipClient, createSessionToken } from "../utils";
import { createGitOpsCluster } from "../../../mutations/ClusterMutations";
import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";
import { createGitOpsClusterRaw } from "../../../mutations/ClusterMutations";

chai.use(chaiAsPromised);

export default () => {
  it("creates a gitops cluster for solo dev", async (done) => {
    await global.provider.addInteraction(createGitOpsClusterInteraction);
    const result = await getShipClient("solo-account-session-1").mutate({
      mutation: createGitOpsCluster,
      variables: {
        title: "FooBarGit Cluster",
        installationId: 987654,
        gitOpsRef: {
          owner: "me",
          repo: "myself",
          branch: "i",
        },
      },
    });
    // expect(result.data.createGitOpsCluster).to.deep.equal({"id": "generated", "slug": "foobargit-cluster"})
    // createdClusterId = result.data.createGitOpsCluster.id;
    global.provider.verify().then(() => done());
  });
}

const createGitOpsClusterInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a mutation to create a gitops cluster for solo dev")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("solo-account-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withOperation("createGitOpsCluster")
  .withQuery(createGitOpsClusterRaw)
  .withVariables({
    title: "FooBarGit Cluster",
    installationId: 987654,
    gitOpsRef: {
      owner: "me",
      repo: "myself",
      branch: "i",
    },
  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      data: {
        createGitOpsCluster: {
          id: Matchers.like("generated"),
          slug: "foobargit-cluster",
        },
      },
    },
  });

