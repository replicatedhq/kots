import React from "react";
import Modal from "react-modal";
import CodeSnippet from "@src/components/shared/CodeSnippet";

function makeDeployCommand({
  appSlug,
  chartPath,
  valuesFilePath,
}) {
  if (valuesFilePath) {
    return `helm upgrade ${appSlug} ${chartPath} -f <path-to-values-yaml>`;
  }

  return `helm upgrade ${appSlug} ${chartPath} --reuse-values`;
}

function makeLoginCommand({
  registryHostname = "",
  registryUsername,
  registryPassword,
} = {}) {
  return `helm registry login ${registryHostname.slice(6)} --username ${registryUsername} --password ${registryPassword}`
}

export default function HelmDeployModal({
  appSlug,
  chartPath,
  hideHelmDeployModal = () => { },
  showHelmDeployModal,
  subtitle,
  registryUsername = "myUsername",
  registryPassword = "myPassword",
  title,
  upgradeTitle,
  valuesFilePath = null,
}) {

  return (
    <Modal
      isOpen={showHelmDeployModal}
      onRequestClose={hideHelmDeployModal}
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
          <div className="u-marginBottom--30 flex flex-row">
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
                  registryUsername,
                  registryPassword,
                })}
              </CodeSnippet>
            </div>
          </div>
          {valuesFilePath && <div className="u-marginBottom--30 flex flex-row">
            <span className="Title step-number u-marginRight--15">2</span>
            <div className="flex1">
              <span className="Title u-marginBottom--10 u-display--block">
                Download your new values.yaml file
              </span>
              <button
                className="btn secondary blue large flex alignItems--center"
              >
                <span
                  className="icon blue-yaml-icon u-marginRight--10"
                />
                <span className="flex1">
                  Download values.yaml
                </span>
              </button>
            </div>
          </div>
          }
          <div className="u-marginBottom--30 flex flex-row">
            <span className="Title step-number u-marginRight--15">{valuesFilePath === null ? "2" : "3"}</span>
            <div className="flex1">
              <span className="Title u-marginBottom--5 u-display--block">
                {upgradeTitle}
              </span>
              {valuesFilePath && <p className="flex1 subtitle u-marginBottom--15">
                Ensure you replace <code>{"<path-to-values-yaml>"}</code> with the path to your saved file.
              </p>
              }
              <CodeSnippet
                language="bash"
                canCopy={true}
                onCopyText={<span className="u-textColor--success">Command has been copied to your clipboard</span>}
              >
                {makeDeployCommand({
                  appSlug,
                  chartPath,
                  valuesFilePath,
                })}
              </CodeSnippet>
            </div>
          </div>
        </div>
        <button
          onClick={hideHelmDeployModal}
          className="btn blue primary large">
          Ok, got it!
        </button>
      </div>
    </Modal>
  );
}

export { HelmDeployModal }