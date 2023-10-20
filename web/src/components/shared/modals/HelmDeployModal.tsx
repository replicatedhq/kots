import Modal from "react-modal";
// TODO: add type to CodeSnippet
// @ts-ignore
import CodeSnippet from "@src/components/shared/CodeSnippet";
import "./styles/HelmDeployModal.scss";

function makeDeployCommand({
  appSlug,
  chartPath,
  revision = null,
  showDownloadValues,
  version,
  namespace,
}: {
  appSlug: string;
  chartPath: string;
  revision: number | null;
  showDownloadValues: boolean;
  version?: string;
  namespace: string;
}) {
  if (revision) {
    return `helm -n ${namespace} rollback ${appSlug} ${revision}`;
  }

  if (showDownloadValues) {
    return `helm -n ${namespace} upgrade ${appSlug} ${chartPath} --version ${version} -f <path-to-values-yaml>`;
  }

  return `helm -n ${namespace} upgrade ${appSlug} ${chartPath} --reuse-values --version ${version}`;
}

function makeLoginCommand({
  registryHostname = "",
  registryUsername,
  registryPassword,
}: {
  registryHostname?: string;
  registryUsername?: string;
  registryPassword?: string;
} = {}) {
  return `helm registry login ${
    registryHostname.slice(6).split("/")[0]
  } --username ${registryUsername} --password ${registryPassword}`;
}

function HelmDeployModal({
  appSlug,
  chartPath,
  downloadClicked = () => {},
  downloadError = false,
  hideHelmDeployModal = () => {},
  // TODO: add downloading state
  // isDownloading = false,
  saveError = false,
  showHelmDeployModal,
  subtitle,
  registryUsername = "myUsername",
  registryPassword = "myPassword",
  revision = null,
  title,
  upgradeTitle,
  showDownloadValues = false,
  version,
  namespace,
}: {
  appSlug: string;
  chartPath: string;
  downloadClicked: () => void;
  downloadError: boolean;
  hideHelmDeployModal: () => void;
  // isDownloading: boolean;
  saveError?: boolean;
  showHelmDeployModal?: boolean;
  subtitle: string;
  registryUsername: string;
  registryPassword: string;
  revision?: number | null;
  title: string;
  upgradeTitle: string;
  showDownloadValues: boolean;
  version?: string;
  namespace: string;
}) {
  return (
    <Modal
      // TODO: figure out if this prop is even needed
      // @ts-ignore
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
                onCopyText={
                  <span className="u-textColor--success">
                    Command has been copied to your clipboard
                  </span>
                }
              >
                {makeLoginCommand({
                  registryHostname: chartPath,
                  registryUsername,
                  registryPassword,
                })}
              </CodeSnippet>
            </div>
          </div>
          {showDownloadValues && (
            <div className="u-marginBottom--30 flex flex-row">
              <span className="Title step-number u-marginRight--15">2</span>
              <div className="flex1">
                <span className={`Title u-marginBottom--10 u-display--block `}>
                  Download your new values.yaml file
                </span>
                <button
                  className="btn secondary blue large flex alignItems--center u-marginBottom--5"
                  onClick={downloadClicked}
                >
                  <span className="icon blue-yaml-icon u-marginRight--10" />
                  <span className="flex1">Download values.yaml</span>
                </button>
                {downloadError && (
                  <span className="CodeSnippet-copy u-textColor--error is-copied">
                    There was a problem downloading your values.yaml file. Try
                    again.
                  </span>
                )}
              </div>
            </div>
          )}
          <div className="u-marginBottom--30 flex flex-row">
            <span className="Title step-number u-marginRight--15">
              {showDownloadValues ? "3" : "2"}
            </span>
            <div className="flex1">
              <span className="Title u-marginBottom--5 u-display--block">
                {upgradeTitle}
              </span>
              {showDownloadValues && (
                <p className="flex1 subtitle u-marginBottom--15">
                  Ensure you replace <code>{"<path-to-values-yaml>"}</code> with
                  the path to your saved file.
                </p>
              )}
              <CodeSnippet
                language="bash"
                canCopy={true}
                onCopyText={
                  <span className="u-textColor--success">
                    Command has been copied to your clipboard
                  </span>
                }
              >
                {makeDeployCommand({
                  appSlug,
                  chartPath,
                  revision,
                  showDownloadValues,
                  version,
                  namespace,
                })}
              </CodeSnippet>
            </div>
          </div>
        </div>
        <button
          onClick={hideHelmDeployModal}
          className="btn blue primary large"
        >
          Ok, got it!
        </button>
        {saveError && (
          <span className="CodeSnippet-copy u-textColor--error u-display--block is-copied u-marginTop--5">
            There was a problem saving your configuration. Close this modal and
            try again.
          </span>
        )}
      </div>
    </Modal>
  );
}

export { HelmDeployModal };
