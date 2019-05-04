import * as Pact from "@pact-foundation/pact";
import { Matchers } from "@pact-foundation/pact";

export const shipAuthSignupInteraction = new Pact.Interaction()
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

