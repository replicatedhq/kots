import React from "react";
import Modal from "react-modal";
import CodeSnippet from "@src/components/shared/CodeSnippet";

function makeRedeployCommand({
  appSlug,
  chartPath,
  valuesPath = "http://downloads.replicated.com/helm/D3akdj3.yaml"
}) {
  return `helm upgrade ${appSlug} ${chartPath} -f ${valuesPath}`;
}

function makeLoginCommand({
  registryHostname = "",
  registryUsername = "myUsername",
  registryPassword = "myPassword",
} = {}) {
  return `helm registry login ${registryHostname.slice(6)} --username ${registryUsername} --password ${registryPassword}`
}

export default function HelmDeployModal({
  appSlug,
  showHelmDeployModal,
  hideDeployModal,
  chartPath,
}) {

  return (
    <Modal
      isOpen={showHelmDeployModal}
      onRequestClose={hideDeployModal}
      shouldReturnFocusAfterClose={false}
      contentLabel=""
      ariaHideApp={false}
      className="Modal PreflightModal"
    >
      <div className="Modal-header has-border flex">
        <h3 className="flex1">Redeploy Helm chart</h3>
        <span className="icon u-grayX-icon u-cursor--pointer"></span>
      </div>
      <div className="Modal-body">
        <div className="flex flex-column">
          <div className="u-marginBottom--40">
            <span className="Title u-marginBottom--normal"><span className="step-number">1</span>Log in to the registry</span>
            <CodeSnippet
              language="bash"
              canCopy={true}
              onCopyText={<span className="u-textColor--success">Command has been copied to your clipboard</span>}
            >
              {makeLoginCommand({
                registryHostname: chartPath,
              })}
            </CodeSnippet>
          </div>
          <div className="u-marginBottom--40">
            <span className="Title u-marginBottom--normal"><span className="step-number">2</span>Upgrade with Helm</span>
            <CodeSnippet
              language="bash"
              canCopy={true}
              onCopyText={<span className="u-textColor--success">Command has been copied to your clipboard</span>}
            >
              {makeRedeployCommand({
                appSlug,
                chartPath
                })}
            </CodeSnippet>
          </div>
        </div>
      </div>
    </Modal>
  );
}

export { HelmDeployModal }