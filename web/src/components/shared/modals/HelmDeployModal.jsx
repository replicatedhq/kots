import React from "react";
import Modal from "react-modal";
import CodeSnippet from "@src/components/shared/CodeSnippet";

function makeRedeployCommand({
  appSlug,
  chartPath,
  valuesFilePath,
}) {
  if (valuesFilePath) {
    return `helm upgrade ${appSlug} ${chartPath} -f ${valuesFilePath}`;
  }

  return `helm upgrade ${appSlug} ${chartPath} --reuse-values`;
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
  chartPath,
  hideDeployModal,
  showHelmDeployModal,
  subtitle,
  title,
  viewValuesClicked = () => { },
  valuesFilePath = null,
}) {

  return (
    <Modal
      isOpen={showHelmDeployModal}
      onRequestClose={hideDeployModal}
      shouldReturnFocusAfterClose={false}
      contentLabel=""
      ariaHideApp={false}
      className="Modal MediumSizeExtra helm"
    >
      <div className="Modal-header flex-row">
        <h3 className="flex1">{title}</h3>
        <p className="flex1 subtitle">{subtitle}</p>
      </div>
      <div className="Modal-body">
        <div className="flex flex-column">
          <div className="u-marginBottom--40 flex flex-row">
            <span className="Title step-number u-marginRight--15">1</span>
            <div className="flex1">
              <span className="Title u-marginBottom--10 u-display--block">
                Log in to the registry
              </span>
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
          </div>
          <div className="u-marginBottom--40 flex flex-row">
            <span className="Title step-number u-marginRight--15">2</span>
            <div className="flex1">
              <span className="Title u-marginBottom--10 u-display--block">
                Upgrade with Helm
              </span>
              <CodeSnippet
                language="bash"
                canCopy={true}
                onCopyText={<span className="u-textColor--success">Command has been copied to your clipboard</span>}
              >
                {makeRedeployCommand({
                  appSlug,
                  chartPath,
                  valuesFilePath,
                })}
              </CodeSnippet>
            </div>
          </div>
        </div>
      </div>
    </Modal>
  );
}

export { HelmDeployModal }