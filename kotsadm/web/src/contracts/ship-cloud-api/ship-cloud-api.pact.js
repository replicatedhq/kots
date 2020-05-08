/* global
  describe
*/
import listClusters from "./contracts/list-clusters";
import getKotsAppCheck from "./contracts/get-kots-app";
import getKotsPreflightResult from "./contracts/get-kots-preflight-result";
import getLatestKotsPreflightResult from "./contracts/get-latest-kots-preflight-result";
import deployKotsVersion from "./contracts/deploy-kots-version";
import getKotsDownstreamHistory from "./contracts/get-kots-downstream-history";
import kotsDownstreamHistoryWithNotes from "./contracts/kots-downstream-history-with-notes";
import kotsDownstreamHistoryNoNotes from "./contracts/kots-downstream-history-no-notes";
import kotsDownstreamHistoryDifferentSequences from "./contracts/kots-downstream-history-different-sequences";

describe("ShipAPI GraphQL Pact", () => {

  describe("solo-account:listClusters", () => listClusters() );

  describe("get-kots-app", () => getKotsAppCheck());
  describe("get-kots-preflight-result", () => getKotsPreflightResult());
  describe("get-latest-kots-preflight-result", () => getLatestKotsPreflightResult());
  describe("deploy-kots-version", () => deployKotsVersion());
  describe("get-kots-downstream-history", () => getKotsDownstreamHistory());
  describe("kots-downstream-history-with-notes", () => kotsDownstreamHistoryWithNotes());
  describe("kots-downstream-history-no-notes", () => kotsDownstreamHistoryNoNotes());
  describe("kots-downstream-history-different-sequences", () => kotsDownstreamHistoryDifferentSequences());

});
