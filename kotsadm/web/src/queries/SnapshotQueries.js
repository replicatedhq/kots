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

export const snapshotDetailRaw = `
  query snapshotDetail($slug: String!, $id: String!) {
    snapshotDetail(slug: $slug, id: $id) {
      name
      status
      volumeSizeHuman
      namespaces
      hooks {
        hookName
        phase
        command
        containerName
        podName
        namespace
        stdout
        stderr
        started
        finished
        warning {
          title
          message
        }
        error {
          title
          message
        }
      }
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
`;

export const snapshotDetail = gql(snapshotDetailRaw);

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
