/* global
  describe
*/
import getKotsAppCheck from "./contracts/get-kots-app";
import deployKotsVersion from "./contracts/deploy-kots-version";

describe("ShipAPI GraphQL Pact", () => {

  describe("get-kots-app", () => getKotsAppCheck());
  describe("deploy-kots-version", () => deployKotsVersion());

});
