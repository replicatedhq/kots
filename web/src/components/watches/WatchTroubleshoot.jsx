import React from "react";

export default function WatchTroubleshoot() {
  const command = "kubectl exec `kubectl get pods --selector=tier=support-bundle -o jsonpath='{.items[*].metadata.name}'` -- support-bundle generate --out - --quiet --yes-upload > support-bundle.tar.gz";
  return (
    <div className="TroubleshootCode--wrapper flex-auto">
      <div className="flex1 flex-column">
        <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--normal">Generate a support bundle</p>
        <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal u-marginBottom--5">If you’re having issues with your application, run the command below to generate a support bundle to send to the vendor for analysis.</p>
        <code className="u-lineHeight--normal u-fontSize--small u-overflow--auto">
          {command}
        </code>
      </div>
      <div>
        <span className="replicated-link u-fontSize--small">Copy command</span>
      </div>
    </div>
  );
}