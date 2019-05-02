import gql from "graphql-tag";

export const createNotification = gql`
  mutation createNotification($watchId: String!, $webhook: WebhookNotificationInput, $email: EmailNotificationInput) {
    createNotification(watchId: $watchId, webhook: $webhook, email: $email) {
      id
    }
  }
`;

export const updateNotification = gql`
  mutation updateNotification($watchId: String!, $notificationId: String!, $webhook: WebhookNotificationInput, $email: EmailNotificationInput) {
    updateNotification(watchId: $watchId, notificationId: $notificationId, webhook: $webhook, email: $email) {
      id
    }
  }
`;

export const enableNotification = gql`
  mutation enableNotification($watchId: String!, $notificationId: String!, $enabled: Int!) {
    enableNotification(watchId: $watchId, notificationId: $notificationId, enabled: $enabled) {
      id
      createdOn
      updatedOn
      triggeredOn
      enabled
      isDefault
      webhook {
        uri
      }
      email {
        recipientAddress
      }
    }
  }
`;
