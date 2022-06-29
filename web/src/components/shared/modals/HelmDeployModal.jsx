import React from "react";
import Modal from "react-modal";
import CodeSnippet from "@src/components/shared/CodeSnippet";

function makeRedeployCommand() {
  return  `helm upgrade -f values.yaml --set "kots.skipPreflights=true" kots-app-preflights`
}

function makeLoginCommand({
  registryHostname = "hostname.com",
  registryUsername = "myUsername",
  registryPassword = "myPassword",
} = {}) {
  return `helm registry login ${registryHostname} --username ${registryUsername} --password ${registryPassword}`
}

export default function HelmDeployModal(props) {
  const { showHelmDeployModal, hideDeployModal, deployKotsDownstream, onForceDeployClick } = props;

  return (
    <Modal
      isOpen={showHelmDeployModal}
      onRequestClose={hideDeployModal}
      shouldReturnFocusAfterClose={false}
      contentLabel="Deploy helm chart or something "
      ariaHideApp={false}
      className="Modal PreflightModal"
    >
      <div className="Modal-header">
          <h3>Redeploy helm chart</h3>
      </div>
      <div className="Modal-body">
        <div className="flex flex-column justifyContent--center alignItems--center">
          <CodeSnippet
            language="bash"
            canCopy={true}
            onCopyText={<span className="u-textColor--success">Command has been copied to your clipboard</span>}
          >
            {makeLoginCommand()}
          </CodeSnippet>
          <CodeSnippet
            language="bash"
            canCopy={true}
            onCopyText={<span className="u-textColor--success">Command has been copied to your clipboard</span>}
          >
            {makeRedeployCommand()}
          </CodeSnippet>
        </div>
      </div>
    </Modal>
  );
}

export { HelmDeployModal }