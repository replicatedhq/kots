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

export const restoreDetail = gql`
query restoreDetail($appId: String!, $restoreName: String!) {
  restoreDetail(appId: $appId, restoreName: $restoreName) {
    name
    phase
    volumes {
      name
      sizeBytesHuman
      doneBytesHuman
      completionPercent
      timeRemainingSeconds
      started
      finished
      phase
    }
    errors {
      title
      message
    }
    warnings{
      title
      message
    }
  }
}
`
