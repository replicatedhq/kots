const Kurl = `
type Kurl {
  nodes: [Node]
  ha: Boolean
  isKurlEnabled: Boolean
}
`;

const Node = `
type Node {
  name: String
  isConnected: Boolean
  canDelete: Boolean
  kubeletVersion: String
  cpu: CapacityAvailable
  memory: CapacityAvailable
  pods: CapacityAvailable
  conditions: NodeConditions
}
`;

const CapacityAvailable = `
type CapacityAvailable {
  capacity: Float
  available: Float
}
`;

const NodeConditions = `
type NodeConditions {
  memoryPressure: Boolean
  diskPressure: Boolean
  pidPressure: Boolean
  ready: Boolean
}
`;

const Command = `
type Command {
  command: [String]!
  expiry: Float!
}`;

export default [
  Kurl,
  Node,
  CapacityAvailable,
  NodeConditions,
  Command,
];
