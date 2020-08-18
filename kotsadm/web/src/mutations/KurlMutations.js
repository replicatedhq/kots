import gql from "graphql-tag";

export const deleteNodeRaw = `
  mutation deleteNode($name: String!) {
    deleteNode(name: $name)
  }
`;

export const deleteNode = gql(deleteNodeRaw);
