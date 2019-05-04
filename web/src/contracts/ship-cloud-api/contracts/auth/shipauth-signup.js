import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import fetch from "node-fetch";
import { shipAuthSignupInteraction } from "./interactions";
import { getShipClient } from "../../../utils";

chai.use(chaiAsPromised);
const expect = chai.expect;

const MOCK_SERVER_PORT = 3333;

export default () => {
  beforeEach((done) => {
    global.provider.removeInteractions().then(() => done());
  });

  it("signs up as a ship auth user", async (done) => {
    // await global.provider.addInteraction(shipAuthSignupInteraction);

    // const result = await doSignUp(`http://localhost:${MOCK_SERVER_PORT}/api/v1/signup`, {
    //   "email": "test-ship-auth-signup@gmail.com",
    //   "password": "password",
    // }, fetch);

    // const body = await result.json();
    // expect(body.token).to.equal("generated");
    // expect(body.signup.email).to.equal("test-ship-auth-signup@gmail.com");
    // expect(body.signup.id).to.equal("generated");

    // global.provider.verify();
    done();
  });

};
