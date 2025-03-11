export const REDACT_SPEC = `apiVersion: troubleshoot.sh/v1beta2
kind: Redactor
metadata:
  name: kotsadm-redact
spec:
  redactors:
  - name: redact cluster version info
    fileSelector:
      file: cluster-info/cluster_version.json
    removals:
      regex:
      - redactor: ("major"?:) (")(?P<mask>.*)(")
      - redactor: ("minor"?:) (")(?P<mask>.*)(")
`;
