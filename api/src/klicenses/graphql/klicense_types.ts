const KEntitlement = `
  type KEntitlement {
    title: String
    value: String
    label: String
  }
`;

const KLicense = `
type KLicense {
  id: String
  expiresAt: String
  channelName: String
  entitlements: [KEntitlement]
}`;

export default [
  KEntitlement,
  KLicense,
];
