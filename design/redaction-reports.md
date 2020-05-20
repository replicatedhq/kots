# Redaction Reports

Support bundles are sanitized with user-provided redactors.
Statistics on what redactors were activated and where should be provided.

## Goals

- Users can see what redactors were activated, and what lines they affected.

## Non Goals

- Users cannot see what, specifically, was redacted.
- If multiple redactors triggered on the same line, users cannot see what redaction is attributable to which redactor.

## Background

Users create custom redactors for support bundles, and it can be difficult to validate that they are functioning.
Redaction reports allow users to see that their redactors are having an effect.

## High-Level Design

Troubleshoot is modified to collect information on what redactions were applied, and where.
This information is then returned when generating a support bundle via the API, or POSTed with the completed bundle if generated from the CLI.
Kots stores this in postgres, and makes it available to the UI via a REST api.

## Detailed Design

Redactions are collected and stored as the following objects by troubleshoot:
```go
type RedactionList struct {
	ByRedactor map[string][]Redaction
	ByFile     map[string][]Redaction
}

type Redaction struct {
	RedactorName      string
	CharactersRemoved int
	Line              int
	File              string
}
```

These are then exposed via GET at `/api/v1/troubleshoot/supportbundle/{bundleId}/redactions` with a response type that includes error/success:

```go
type GetSupportBundleRedactionsResponse struct {
	Redactions redact.RedactionList `json:"redactions"`

	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}
```

Redactions can be set for a bundle with a PUT to the same path (`/api/v1/troubleshoot/supportbundle/{bundleId}/redactions`) with the following structure:
```go
type PutSupportBundleRedactions struct {
	Redactions redact.RedactionList `json:"redactions"`
}
```

Redaction reports will be stored as a new mediumtext column 'redactions' in the 'supportbundle' table.

Within troubleshoot, the ResultRequest type is modified to add a URI to upload redaction reports to:
```go
type ResultRequest struct {
	URI       string `json:"uri" yaml:"uri"`
	Method    string `json:"method" yaml:"method"`
	RedactURI string `json:"redactUri" yaml:"redactUri"` // the URI to POST redaction reports to
}
```

When kotsadm generates troubleshoot specs, RedactURI will be populated with the proper value. (This is already done for URI here)

## Alternatives Considered

## Security Considerations

Some information leakage from redaction reports is possible, but should be minimal - limited to 'this was an IP address' and similar.
