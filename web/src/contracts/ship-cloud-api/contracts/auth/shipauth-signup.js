import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { shipAuthSignupInteraction } from "./interactions";
import { shipAuthSignup } from "../../../../mutations/AuthMutations";
import { getShipClient } from "../../utils";

chai.use(chaiAsPromised);
const expect = chai.expect;

const MOCK_SERVER_PORT = 3333;

export default () => {
  beforeEach((done) => {
    global.provider.removeInteractions().then(() => done());
  });

  it("signs up as a ship auth user", async (done) => {
    await global.provider.addInteraction(shipAuthSignupInteraction);

    getShipClient().mutate({
      mutation: shipAuthSignup,
      variables: {
        input: {
          email: "test-ship-auth-signup@gmail.com",
          password: "password",
        },
      }
    })
    .then(result => {
      expect(result.data.signup.token).to.equal("generated");
      global.provider.verify();
      done();
    })
    .catch(err => {
      console.error(err);
    })
  });

};
