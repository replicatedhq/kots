import gql from "graphql-tag";

export const kurlRaw = `
  query kurl {
    kurl {
      addNodeCommand
      nodes {
        name
        isConnected
        canDelete
        kubeletVersion
        cpu {
          capacity
          allocatable
        }
        memory {
          capacity
          allocatable
        }
        pods {
          capacity
          allocatable
        }
        conditions {
          memoryPressure
          diskPressure
          pidPressure
          ready
        }
      }
    }
  }
`;

export const kurl = gql(kurlRaw);
