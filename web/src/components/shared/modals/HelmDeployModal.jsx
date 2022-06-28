import React from "react";
import Modal from "react-modal";
import CodeSnippet from "@src/components/shared/CodeSnippet";

export default function SkipPreflightsModal(props) {
  const { showHelmDeployModal, hideDeployModal, deployKotsDownstream, onForceDeployClick } = props;
  console.log("test")

  return (
    <Modal
      isOpen={showHelmDeployModal}
      onRequestClose={hideDeployModal}
      shouldReturnFocusAfterClose={false}
      contentLabel="Deploy helm chart or something "
      ariaHideApp={false}
      className="Modal PreflightModal"
    >
      <div className="Modal-body">
        <div className="flex flex-column justifyContent--center alignItems--center">
          <span className="icon yellowWarningIcon" />
          <CodeSnippet
            language="bash"
            canCopy={true}
            onCopyText={<span className="u-textColor--success">Command has been copied to your clipboard</span>}
          >
            helm urpgrade -f values.yaml --set "kots.skipPreflights=true" kots-app-preflights
          </CodeSnippet>
        </div>
      </div>
    </Modal>
  );
}