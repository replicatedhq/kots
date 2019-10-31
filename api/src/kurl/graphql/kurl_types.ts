const Kurl = `
type Kurl {
  nodes: [Node]
  addNodeCommand: String
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

export default [
  Kurl,
  Node,
  CapacityAvailable,
  NodeConditions
];
