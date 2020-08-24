import gql from "graphql-tag";

export const snapshotConfigRaw = `
  query snapshotConfig($slug: String!) {
    snapshotConfig(slug: $slug) {
      autoEnabled
      autoSchedule {
        schedule
      }
      ttl {
        inputValue
        inputTimeUnit
        converted
      }
    }
  }
`;

export const snapshotConfig = gql(snapshotConfigRaw);
