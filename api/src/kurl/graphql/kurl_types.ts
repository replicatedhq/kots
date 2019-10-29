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
  cpu: CapacityAllocatable
  memory: CapacityAllocatable
  pods: CapacityAllocatable
  conditions: NodeConditions
}
`;

const CapacityAllocatable = `
type CapacityAllocatable {
  capacity: String
  allocatable: String
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
  CapacityAllocatable,
  NodeConditions
];
