import { Component } from "react";
import Modal from "react-modal";
import semverjs from "semver";
import SnapshotInstallationBox from "./SnapshotInstallationBox";
import CodeSnippet from "../shared/CodeSnippet";
import {
  FILE_SYSTEM_NFS_TYPE,
  FILE_SYSTEM_HOSTPATH_TYPE,
} from "./SnapshotStorageDestination.data";
import Icon from "../Icon";

const VELERO_IS_NOT_INSTALLED_TAB = "velero-not-installed";
const VELERO_IS_INSTALLED_TAB = "velero-installed";

class ConfigureSnapshots extends Component {
  constructor(props) {
    super(props);

    this.state = {
      activeTab: VELERO_IS_NOT_INSTALLED_TAB,
    };
  }

  toggleScheduleAction = (active) => {
    this.setState({
      activeTab: active,
    });
  };

  isVelero10OrNewer = (
    veleroVersion = this.props.snapshotSettings?.veleroVersion
  ) => {
    if (!semverjs.valid(veleroVersion)) {
      return true;
    }

    const velero10Semver = semverjs.coerce("1.10");
    const actualVeleroSemver = semverjs.coerce(veleroVersion);

    return semverjs.gte(actualVeleroSemver, velero10Semver);
  };

  getFSBackupComponentName = () =>
    this.isVelero10OrNewer() ? "Node Agent" : "Restic";

  getFSBackupComponentFlags = () =>
    this.isVelero10OrNewer()
      ? ["--use-node-agent", "--uploader-type=restic"]
      : ["--use-restic"];

  render() {
    const { activeTab } = this.state;
    const {
      showConfigureSnapshotsModal,
      toggleConfigureSnapshotsModal,
      kotsadmRequiresVeleroAccess,
      minimalRBACKotsadmNamespace,
      snapshotSettings,
      hideCheckVeleroButton,
      fetchSnapshotSettings,
      renderNotVeleroMessage,
      openConfigureFileSystemProviderModal,
      isKurlEnabled,
    } = this.props;

    return (
      <Modal
        isOpen={showConfigureSnapshotsModal}
        shouldReturnFocusAfterClose={false}
        onRequestClose={() => {
          toggleConfigureSnapshotsModal();
        }}
        ariaHideApp={false}
        contentLabel="Modal"
        className="Modal ConfigureSnapshots"
      >
        <div className="Modal-body" data-testid="configure-snapshots-modal">
          <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-marginBottom--20">
            Add a new destination
          </p>
          <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-lineHeight--normal">
            In order to configure and use Snapshots (backup and restore), please
            install Velero in the cluster. Once Velero is installed, click the
            button below and the Admin Console will verify the installation and
            begin configuring Snapshots.
          </p>
          {kotsadmRequiresVeleroAccess && (
            <div className="ConfigureSnapshotsTabs--wrapper flex-column u-marginTop--20 u-marginBottom--20">
              <div className="tab-items flex">
                <span
                  className={`${
                    this.state.activeTab === VELERO_IS_NOT_INSTALLED_TAB
                      ? "is-active"
                      : ""
                  } tab-item blue`}
                  data-testid="velero-not-installed-tab"
                  onClick={() =>
                    this.toggleScheduleAction(VELERO_IS_NOT_INSTALLED_TAB)
                  }
                >
                  I need to install Velero
                </span>
                <span
                  className={`${
                    this.state.activeTab === VELERO_IS_INSTALLED_TAB
                      ? "is-active"
                      : ""
                  } tab-item blue`}
                  data-testid="velero-already-installed-tab"
                  onClick={() =>
                    this.toggleScheduleAction(VELERO_IS_INSTALLED_TAB)
                  }
                >
                  I've already installed Velero
                </span>
              </div>
            </div>
          )}
          {activeTab === VELERO_IS_INSTALLED_TAB ? (
            <div className="flex-column u-marginTop--12">
              <p
                className="u-fontSize--large u-fontWeight--bold u-textColor--secondary u-marginBottom--10"
                data-testid="velero-namespace-access-required"
              >
                Velero namespace access required
              </p>
              <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--10">
                {" "}
                We’ve detected that the Admin Console is running with minimal
                role-based-access-control (RBAC) privileges, meaning that the
                Admin Console is limited to a single namespace. To use the
                snapshots functionality, the Admin Console requires access to
                the namespace Velero is installed in. Please make sure Velero is
                installed, then use the following command to provide the Admin
                Console with the necessary permissions to access it:{" "}
              </p>
              <CodeSnippet
                language="bash"
                canCopy={true}
                onCopyText={
                  <span className="u-textColor--success">
                    Command has been copied to your clipboard
                  </span>
                }
                dataTestId="ensure-permissions-command"
              >
                {`kubectl kots velero ensure-permissions --namespace ${minimalRBACKotsadmNamespace} --velero-namespace <velero-namespace>`}
              </CodeSnippet>
              <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy u-lineHeight--normal u-marginTop--20 u-marginBottom--20">
                {" "}
                <span className="u-fontWeight--bold u-textColor--secondary">
                  Note:
                </span>{" "}
                Please replace {`"<velero-namespace>"`} with the actual
                namespace Velero is installed in, which is {`'velero'`} by
                default.{" "}
              </p>
            </div>
          ) : (
            <div
              className="flex-column u-marginTop--12"
              data-testid="velero-install-instructions"
            >
              <div className="InstallVelero--wrapper flex flex-column">
                <p className="u-textColor--secondary u-fontSize--large u-fontWeight--bold">
                  To install Velero
                </p>
                <div className="flex1 flex-column u-marginBottom--30">
                  {isKurlEnabled ? (
                    <p className="u-fontSize--small flex-auto alignItems--center u-fontWeight--medium u-textColor--bodyCopy u-marginTop--20">
                      <span className="circleNumberGray u-marginRight--10">
                        {" "}
                        1{" "}
                      </span>
                      Install the CLI on your machine by following the Velero
                      installation instructions at:{" "}
                      <a
                        href="https://docs.replicated.com/enterprise/snapshots-velero-cli-installing"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="link u-marginLeft--5"
                      >
                        https://docs.replicated.com/enterprise/snapshots-velero-cli-installing
                      </a>{" "}
                    </p>
                  ) : (
                    <p className="u-fontSize--small flex-auto alignItems--center u-fontWeight--medium u-textColor--bodyCopy u-marginTop--20">
                      <span className="circleNumberGray u-marginRight--10">
                        {" "}
                        1{" "}
                      </span>
                      Install the CLI on your machine by following the{" "}
                      <a
                        href="https://docs.replicated.com/enterprise/snapshots-velero-cli-installing"
                        target="_blank"
                        rel="noopener noreferrer"
                        className="link u-marginLeft--5"
                      >
                        Velero installation instructions
                      </a>{" "}
                    </p>
                  )}
                  <div className="flex flex1 u-marginTop--20">
                    <div className="flex">
                      <span className="circleNumberGray u-marginRight--10">
                        {" "}
                        2{" "}
                      </span>
                    </div>
                    <div className="flex flex-column">
                      <p className="u-fontSize--small flex alignItems--center u-fontWeight--medium u-textColor--bodyCopy">
                        {" "}
                        Run the commands from the instructions for your provider{" "}
                      </p>
                      <div
                        className="flex flexWrap--wrap"
                        style={{ width: "500px" }}
                      >
                        <a
                          href="https://github.com/vmware-tanzu/velero-plugin-for-aws#setup"
                          target="_blank"
                          rel="noopener noreferrer"
                          className="snapshotOptions"
                        >
                          <span style={{ width: "130px" }}>
                            {" "}
                            <span className="icon awsIcon u-cursor--pointer u-marginRight--5" />
                            Amazon AWS{" "}
                          </span>
                          <Icon
                            icon="external-page"
                            size={13}
                            className="clickable justifyContent--flexEnd u-marginLeft--30"
                          />
                        </a>
                        <a
                          href="https://github.com/vmware-tanzu/velero-plugin-for-microsoft-azure#setup"
                          target="_blank"
                          rel="noopener noreferrer"
                          className="snapshotOptions"
                        >
                          <span style={{ width: "130px" }}>
                            {" "}
                            <span className="icon azureIcon u-cursor--pointer u-marginRight--5" />
                            Microsoft Azure{" "}
                          </span>
                          <Icon
                            icon="external-page"
                            size={13}
                            className="clickable u-marginLeft--30"
                          />
                        </a>
                        <a
                          href="https://github.com/vmware-tanzu/velero-plugin-for-gcp#setup"
                          target="_blank"
                          rel="noopener noreferrer"
                          className="snapshotOptions"
                        >
                          <span style={{ width: "130px" }}>
                            {" "}
                            <span className="icon googleCloudIcon u-cursor--pointer u-marginRight--5" />
                            Google Cloud{" "}
                          </span>
                          <Icon
                            icon="external-page"
                            size={13}
                            className="clickable u-marginLeft--30"
                          />
                        </a>
                        <a
                          href="https://velero.io/docs/v1.10/supported-providers/"
                          target="_blank"
                          rel="noopener noreferrer"
                          className="snapshotOptions"
                        >
                          <span style={{ width: "130px" }}>
                            {" "}
                            <span className="icon cloudIcon u-cursor--pointer u-marginRight--5" />{" "}
                            Other provider{" "}
                          </span>
                          <Icon
                            icon="external-page"
                            size={13}
                            className="clickable u-marginLeft--30"
                          />
                        </a>
                        {snapshotSettings?.isMinioDisabled ? (
                          <>
                            <a
                              href="https://github.com/replicatedhq/local-volume-provider"
                              target="_blank"
                              rel="noopener noreferrer"
                              className="snapshotOptions"
                            >
                              <span style={{ width: "130px" }}>
                                {" "}
                                <span className="icon nfsIcon u-cursor--pointer u-marginRight--5" />{" "}
                                NFS{" "}
                              </span>
                              <Icon
                                icon="external-page"
                                size={13}
                                className="clickable u-marginLeft--30"
                              />
                            </a>
                            <a
                              href="https://github.com/replicatedhq/local-volume-provider"
                              target="_blank"
                              rel="noopener noreferrer"
                              className="snapshotOptions"
                            >
                              <span style={{ width: "130px" }}>
                                {" "}
                                <span className="icon hostpathIcon u-cursor--pointer u-marginRight--5" />{" "}
                                Host Path{" "}
                              </span>
                              <Icon
                                icon="external-page"
                                size={13}
                                className="clickable u-marginLeft--30"
                              />
                            </a>
                          </>
                        ) : (
                          <>
                            <a
                              className="snapshotOptions"
                              onClick={() =>
                                openConfigureFileSystemProviderModal(
                                  FILE_SYSTEM_NFS_TYPE
                                )
                              }
                            >
                              <span style={{ width: "130px" }}>
                                {" "}
                                <span className="icon nfsIcon u-cursor--pointer u-marginRight--5" />{" "}
                                NFS{" "}
                              </span>
                            </a>
                            <a
                              className="snapshotOptions"
                              onClick={() =>
                                openConfigureFileSystemProviderModal(
                                  FILE_SYSTEM_HOSTPATH_TYPE
                                )
                              }
                            >
                              <span style={{ width: "130px" }}>
                                {" "}
                                <span className="icon hostpathIcon u-cursor--pointer u-marginRight--5" />{" "}
                                Host Path{" "}
                              </span>
                            </a>
                          </>
                        )}
                      </div>
                      <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginTop--20">
                        {" "}
                        With all providers, you must install using the{" "}
                        <span className="inline-code u-marginLeft--5 u-marginRight--5">
                          {this.getFSBackupComponentFlags().join(" ")}{" "}
                        </span>{" "}
                        flag
                        {this.getFSBackupComponentFlags().length > 1
                          ? "s"
                          : ""}{" "}
                        for snapshots to work.{" "}
                      </p>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          )}
          <SnapshotInstallationBox
            fetchSnapshotSettings={fetchSnapshotSettings}
            renderNotVeleroMessage={renderNotVeleroMessage}
            snapshotSettings={snapshotSettings}
            hideCheckVeleroButton={hideCheckVeleroButton}
            fsBackupComponentName={this.getFSBackupComponentName()}
          />
          <div className="flex justifyContent--flexStart u-marginTop--20">
            <button
              className="btn primary blue"
              onClick={toggleConfigureSnapshotsModal}
            >
              {" "}
              Ok, got it!{" "}
            </button>
          </div>
        </div>
      </Modal>
    );
  }
}

export default ConfigureSnapshots;
