import listClustersSolo from "./contracts/solo-account/list-clusters/list-clusters";
import createShipCluster from "./contracts/solo-account/create-ship-cluster/create-ship-cluster";
import createGitOpsCluster from "./contracts/solo-account/create-gitops-cluster/create-gitops-cluster";
import createMidstreamWatch from "./contracts/solo-account/create-midstream-watch/create-midstream-watch";
import listWatchesShipCluster from "./contracts/ship-cluster-only/list-watches/list-watches";
import createInitSession from "./contracts/solo-account/create-init-session/create-init-session";
import getWatchVersion from "./contracts/solo-account/get-watch-version/get-watch-version";
import getImageWatchItems from "./contracts/solo-account/list-image-watch-items/list-image-watch-items";
import getWatchVersionGitOps from "./contracts/gitops-cluster-only/get-watch-version/get-watch-version";

describe("ShipAPI GraphQL Pact", () => {
  afterEach(() => global.provider.verify())

  describe("solo-account:listClusters", () => listClustersSolo() );
  describe("solo-account:createShipCluster", () => createShipCluster() );
  describe("solo-account:createGitOpsCluster", () => createGitOpsCluster() );
  describe("solo-account:createMidstreamWatch", () => createMidstreamWatch() );
  describe("solo-account:createInitSession", () => createInitSession() );
  describe("solo-account:getWatchVersion", () => getWatchVersion() );
  describe("solo-account:getImageWatchItems", () => getImageWatchItems() );

  describe("ship-cluster-account:listWatches", () => listWatchesShipCluster() );

  // describe("gitops-cluster-account:getWatchVersion", () => getWatchVersionGitOps() );
});
