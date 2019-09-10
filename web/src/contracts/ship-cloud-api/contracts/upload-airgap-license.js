/* global
  expect
  it
*/

import * as chai from "chai";
import chaiAsPromised from "chai-as-promised";
import { getShipClient, createSessionToken } from "../utils";
import * as Pact from "@pact-foundation/pact";
// import { Matchers } from "@pact-foundation/pact";
import { uploadKotsLicense, uploadKotsLicenseRaw} from "../../../mutations/AppsMutations";

chai.use(chaiAsPromised);

const YAML_LICENSE =
`
apiVersion: kots.io/v1beta1
kind: License
metadata:
  name: horseysuprise
spec:
  licenseID: valid-license-id-1
  appSlug: sentry-enterprise
  endpoint: https://replicated.app
  signature: IA==
  isAirgapSupported: true
`;

export default () => {
  it("uploads a license", async done => {

    await global.provider.addInteraction(uploadKotsLicenseInteraction);
    const result = await getShipClient("upload-license-session-1").mutate({
      mutation: uploadKotsLicense,
      variables: {
        value: YAML_LICENSE
      }
    }).catch(e => {
      console.log("something bad happen");
      console.log({ error: JSON.stringify(e) });
    });
    const { uploadKotsLicense: gqlResponse } = result.data;
    expect(gqlResponse.hasPreflight).toBe(false);
    expect(gqlResponse.slug).toBe("sentry-enterprise");

    global.provider.verify().then(() => done());
  });

}

const uploadKotsLicenseInteraction = new Pact.GraphQLInteraction()
  .uponReceiving("a mutation to upload a license")
  .withRequest({
    path: "/graphql",
    method: "POST",
    headers: {
      "Authorization": createSessionToken("upload-license-session-1"),
      "Content-Type": "application/json",
    }
  })
  .withOperation("uploadKotsLicense")
  .withQuery(uploadKotsLicenseRaw)
  .withVariables({
    value: YAML_LICENSE
  })
  .willRespondWith({
    status: 200,
    headers: { "Content-Type": "application/json"},
    body: {
      data: {
        uploadKotsLicense: {
          hasPreflight: false,
          slug: "sentry-enterprise"
        }
      }
    }
  });