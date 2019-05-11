import listClusters from "./contracts/list-clusters";
import createShipCluster from "./contracts/create-ship-cluster";
import createGitOpsCluster from "./contracts/create-gitops-cluster";
import createMidstreamWatch from "./contracts/create-midstream-watch";
import listWatchesShipCluster from "./contracts/list-watches-ship-cluster";
import createInitSession from "./contracts/create-init-session";
import getWatchVersion from "./contracts/get-watch-version";
import listImageWatchItems from "./contracts/list-image-watch-items";
import getWatchVersionGitOpsCluster from "./contracts/get-watch-version-gitops-cluster";
import shipauthSignup from "./contracts/shipauth-signup";
import updateWatch from "./contracts/update-watch";
import deleteWatch from "./contracts/delete-watch";
// import getWatchVersionGitOps from "./contracts/get-watch-version-gitops-cluster";

describe("ShipAPI GraphQL Pact", () => {

  describe("solo-account:listClusters", () => listClusters() );
  describe("solo-account:createShipCluster", () => createShipCluster() );
  describe("solo-account:createGitOpsCluster", () => createGitOpsCluster() );
  describe("solo-account:createMidstreamWatch", () => createMidstreamWatch() );
  describe("solo-account:createInitSession", () => createInitSession() );
  describe("solo-account:getWatchVersion", () => getWatchVersion() );
  // describe("solo-account:listImageWatchItems", () => listImageWatchItems() );

  describe("single-user:updateWatch", () => updateWatch() );
  // describe("single-user:deleteWatch", () => deleteWatch() );
  
  // describe("ship-cluster-account:listWatches", () => listWatchesShipCluster() );

  // describe("auth:shipAuthSignup", () => shipauthSignup() );
  // describe("ship-cluster-account:listWatches", () => listWatchesShipCluster() );
  // describe("gitops-cluster-account:getWatchVersion", () => getWatchVersionGitOpsCluster() );
});
