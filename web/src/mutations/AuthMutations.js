import gql from "graphql-tag";

export const trackScmLead = gql`
  mutation trackScmLead($deploymentPreference: String!, $emailAddress: String!, $scmProvider: String!) {
    trackScmLead(deploymentPreference: $deploymentPreference, emailAddress: $emailAddress, scmProvider: $scmProvider)
  }
`;
