import gql from "graphql-tag";

export const kurlRaw = `
  query kurl {
    kurl {
      ha
      isKurlEnabled
      nodes {
        name
        isConnected
        canDelete
        kubeletVersion
        cpu {
          capacity
          available
        }
        memory {
          capacity
          available
        }
        pods {
          capacity
          available
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
