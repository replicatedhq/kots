import gql from "graphql-tag";

export const trackScmLead = gql`
  mutation trackScmLead($deploymentPreference: String!, $emailAddress: String!, $scmProvider: String!) {
    trackScmLead(deploymentPreference: $deploymentPreference, emailAddress: $emailAddress, scmProvider: $scmProvider)
  }
`;

export const createAdminConsolePasswordRaw = `
  mutation createAdminConsolePassword($password: String!) {
    createAdminConsolePassword(password: $password)
  }
`;
export const createAdminConsolePassword = gql(createAdminConsolePasswordRaw);

export const loginToAdminConsoleRaw = `
  mutation loginToAdminConsole($password: String!) {
    loginToAdminConsole(password: $password)
  }
`;
export const loginToAdminConsole = gql(loginToAdminConsoleRaw);

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

export const shipAuthLoginRaw = `
mutation login($input: LoginInput) {
  login(input: $input)
    @rest(
      type: "Login",
      method: "POST",
      path: "/login",
      endpoint: "v1") {
        token
      }
}
`;
export const shipAuthLogin = gql(shipAuthLoginRaw);
