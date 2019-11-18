import gql from "graphql-tag";

export const drainNodeRaw = `
  mutation drainNode($name: String!) {
    drainNode(name: $name)
  }
`;

export const drainNode = gql(drainNodeRaw);

export const deleteNodeRaw = `
  mutation deleteNode($name: String!) {
    deleteNode(name: $name)
  }
`;

export const deleteNode = gql(deleteNodeRaw);

export const generateWorkerAddNodeCommandRaw = `
  mutation {
    generateWorkerAddNodeCommand {
      command
      expiry
    }
  }
`;

export const generateMasterAddNodeCommandRaw = `
  mutation {
    generateMasterAddNodeCommand {
      command
      expiry
    }
  }
`;

export const generateWorkerAddNodeCommand = gql(generateWorkerAddNodeCommandRaw);
export const generateMasterAddNodeCommand = gql(generateMasterAddNodeCommandRaw);
