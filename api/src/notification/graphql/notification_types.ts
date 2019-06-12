const Notification = `
type Notification {
  id: ID
  webhook: WebhookNotification
  email: EmailNotification
  pullRequest: PullRequestNotification
  createdOn: String
  updatedOn: String
  triggeredOn: String
  enabled: Int
  pending: Boolean
}
`;

const WebhookNotification = `
type WebhookNotification {
  uri: String
}
`;

const EmailNotification = `
type EmailNotification {
  recipientAddress: String
}
`;

const PullRequestNotification = `
type PullRequestNotification {
  org: String!
  repo: String!
  branch: String
  rootPath: String
}
`;

const WebhookNotificationInput = `
input WebhookNotificationInput {
  uri: String
}
`;

const EmailNotificationInput = `
input EmailNotificationInput {
  recipientAddress: String
}
`;

const PullRequestNotificationInput = `
input PullRequestNotificationInput {
  org: String!
  repo: String!
  branch: String
  rootPath: String
  pullRequestId: String
}
`;

const PullRequestHistory = `
type PullRequestHistory {
  title: String!
  status: String!
  createdOn: String!
  number: Int
  uri: String
  sequence: Int
  sourceBranch: String
}
`;

// Not currently using, but here just in case
const PendingPR = `
type PendingPR {
  pullrequest_history_id: String!
  org: String
  repo: String
  branch: String
  root_path: String
  created_at: String
  github_installation_id: Int
  pullrequest_number: Int
  watch_id: String
}
`;

export default [
  WebhookNotification,
  WebhookNotificationInput,
  EmailNotification,
  EmailNotificationInput,
  PullRequestNotification,
  PullRequestNotificationInput,
  PullRequestHistory,
  Notification,
  PendingPR,
];
