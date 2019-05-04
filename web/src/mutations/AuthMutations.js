import gql from "graphql-tag";

export const trackScmLead = gql`
  mutation trackScmLead($deploymentPreference: String!, $emailAddress: String!, $scmProvider: String!) {
    trackScmLead(deploymentPreference: $deploymentPreference, emailAddress: $emailAddress, scmProvider: $scmProvider)
  }
`;

export const shipAuthSignupRaw = `
mutation signup($input: SignupInput) {
  signup(input: $input)
    @rest(
      type: "Signup"
      method: "POST"
      path: "/signup"
      endpoint: "v1") {
        token
      }
}
`;
export const shipAuthSignup = gql(shipAuthSignupRaw);