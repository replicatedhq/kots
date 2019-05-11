import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { shipAuthSignup } from "../../../mutations/AuthMutations";
import { getShipClient } from "../utils";
import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";

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
          firstName: "First",
          lastName: "Last",
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

const shipAuthSignupInteraction = new Pact.Interaction()
  .uponReceiving("a query to sign up for a new shipauth account")
  .withRequest({
    path: "/api/v1/signup",
    method: "POST",
    headers: {
      "Content-Type": "application/json",
    },
    body: {
        email: "test-ship-auth-signup@gmail.com",
        firstName: "First",
        lastName: "Last",
        password: "password",
    },
  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json" },
    body: {
      token: Matchers.like("generated"),
      signup: {
        email: "test-ship-auth-signup@gmail.com",
        id: Matchers.like("generated"),
      },
    }
  });

