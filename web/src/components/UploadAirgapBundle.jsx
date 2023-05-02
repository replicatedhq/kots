import * as React from "react";
import classNames from "classnames";
import { KotsPageTitle } from "@components/Head";
import isEmpty from "lodash/isEmpty";
import Modal from "react-modal";
import CodeSnippet from "@src/components/shared/CodeSnippet";
import MountAware from "@src/components/shared/MountAware";
import AirgapUploadProgress from "@features/Dashboard/components/AirgapUploadProgress";
import LicenseUploadProgress from "./LicenseUploadProgress";
import AirgapRegistrySettings from "./shared/AirgapRegistrySettings";
import { Utilities } from "../utilities/utilities";
import { AirgapUploader } from "../utilities/airgapUploader";
import { withRouter } from "@src/utilities/react-router-utilities";

import "../scss/components/troubleshoot/UploadSupportBundleModal.scss";
import "../scss/components/Login.scss";

const COMMON_ERRORS = {
  "HTTP 401": "Registry credentials are invalid",
  "invalid username/password": "Registry credentials are invalid",
  "no such host": "No such host",
};

class UploadAirgapBundle extends React.Component {
  state = {
    bundleFile: {},
    fileUploading: false,
    registryDetails: {},
    preparingOnlineInstall: false,
    supportBundleCommand:
      "curl https://krew.sh/support-bundle | bash\n kubectl support-bundle --load-cluster-specs",
    showSupportBundleCommand: false,
    onlineInstallErrorMessage: "",
    viewOnlineInstallErrorMessage: false,
    uploadProgress: 0,
    uploadSize: 0,
    uploadResuming: false,
  };

  emptyHostnameErrMessage = 'Please enter a value for "Hostname" field';

  componentDidMount() {
    if (!this.state.airgapUploader) {
      this.getAirgapConfig();
    }
  }

  clearFile = () => {
    this.setState({ bundleFile: {} });
  };

  toggleShowRun = () => {
    this.setState({ showSupportBundleCommand: true });
  };

  getAirgapConfig = async () => {
    const { match } = this.props;
    const configUrl = `${process.env.API_ENDPOINT}/app/${match.params.slug}/airgap/config`;
    let simultaneousUploads = 3;
    try {
      let res = await fetch(configUrl, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
      });
      if (res.ok) {
        const response = await res.json();
        simultaneousUploads = response.simultaneousUploads;
      }
    } catch {
      // no-op
    }

    this.setState({
      airgapUploader: new AirgapUploader(
        false,
        match.params.slug,
        this.onDropBundle,
        simultaneousUploads
      ),
    });
  };

  uploadAirgapBundle = async () => {
    const { match, showRegistry } = this.props;

    // Reset the airgap upload state
    const resetUrl = `${process.env.API_ENDPOINT}/app/${match.params.slug}/airgap/reset`;
    try {
      await fetch(resetUrl, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
      });
    } catch (error) {
      console.error(error);
      this.setState({
        fileUploading: false,
        uploadProgress: 0,
        uploadSize: 0,
        uploadResuming: false,
        errorMessage:
          "An error occurred while uploading your airgap bundle. Please try again",
      });
      return;
    }

    this.setState({
      fileUploading: true,
      errorMessage: "",
      showSupportBundleCommand: false,
      onlineInstallErrorMessage: "",
    });

    if (showRegistry) {
      const { slug } = this.props.match.params;

      if (isEmpty(this.state.registryDetails.hostname)) {
        this.setState({
          fileUploading: false,
          uploadProgress: 0,
          uploadSize: 0,
          uploadResuming: false,
          errorMessage: this.emptyHostnameErrMessage,
        });
        return;
      }

      let res;
      try {
        res = await fetch(
          `${process.env.API_ENDPOINT}/app/${slug}/registry/validate`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
            },
            body: JSON.stringify({
              hostname: this.state.registryDetails.hostname,
              namespace: this.state.registryDetails.namespace,
              username: this.state.registryDetails.username,
              password: this.state.registryDetails.password,
              isReadOnly: this.state.registryDetails.isReadOnly,
            }),
            credentials: "include",
          }
        );
      } catch (err) {
        this.setState({
          fileUploading: false,
          uploadProgress: 0,
          uploadSize: 0,
          uploadResuming: false,
          errorMessage: err,
        });
        return;
      }

      const response = await res.json();
      if (!response.success) {
        let msg =
          "An error occurred while uploading your airgap bundle. Please try again";
        if (response.error) {
          msg = response.error;
        }
        this.setState({
          fileUploading: false,
          uploadProgress: 0,
          uploadSize: 0,
          uploadResuming: false,
          errorMessage: msg,
        });
        return;
      }
    }

    const params = {
      registryHost: this.state.registryDetails.hostname,
      namespace: this.state.registryDetails.namespace,
      username: this.state.registryDetails.username,
      password: this.state.registryDetails.password,
      isReadOnly: this.state.registryDetails.isReadOnly,
      simultaneousUploads: this.state.simultaneousUploads,
    };
    this.state.airgapUploader.upload(
      params,
      this.onUploadProgress,
      this.onUploadError
    );
  };

  onUploadProgress = (progress, size, resuming = false) => {
    this.setState({
      uploadProgress: progress,
      uploadSize: size,
      uploadResuming: resuming,
    });
  };

  onUploadError = (message) => {
    this.setState({
      fileUploading: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
      errorMessage: message || "Error uploading bundle, please try again",
    });
  };

  getRegistryDetails = (fields) => {
    this.setState({
      ...this.state,
      registryDetails: {
        hostname: fields.hostname,
        username: fields.username,
        password: fields.password,
        namespace: fields.namespace,
        isReadOnly: fields.isReadOnly,
      },
    });
  };

  onDropBundle = async (file) => {
    this.setState({
      bundleFile: file,
      onlineInstallErrorMessage: "",
      errorMessage: "",
    });
  };

  moveBar = (count) => {
    const elem = document.getElementById("myBar");
    const percent = count > 3 ? 96 : count * 30;
    if (elem) {
      elem.style.width = percent + "%";
    }
  };

  handleOnlineInstall = async () => {
    const { slug } = this.props.match.params;

    this.setState({
      preparingOnlineInstall: true,
      onlineInstallErrorMessage: "",
    });

    let resumeResult;
    fetch(`${process.env.API_ENDPOINT}/license/resume`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      body: JSON.stringify({
        slug,
      }),
    })
      .then(async (result) => {
        resumeResult = await result.json();
      })
      .catch((err) => {
        this.setState({
          // TODO: use fewer flags
          fileUploading: false,
          errorMessage: err,
          preparingOnlineInstall: false,
          onlineInstallErrorMessage: err,
        });
        return;
      });

    let count = 0;
    const interval = setInterval(() => {
      if (this.state.onlineInstallErrorMessage.length) {
        clearInterval(interval);
      }
      count++;
      this.moveBar(count);
      if (count > 3) {
        if (!resumeResult) {
          return;
        }

        clearInterval(interval);

        if (resumeResult.error) {
          this.setState({
            // TODO: use fewer flags
            fileUploading: false,
            errorMessage: resumeResult.error,
            preparingOnlineInstall: false,
            onlineInstallErrorMessage: resumeResult.error,
          });
          return;
        }

        this.props.onUploadSuccess().then(() => {
          // When successful, refetch all the user's apps with onUploadSuccess
          const hasPreflight = resumeResult.hasPreflight;
          const isConfigurable = resumeResult.isConfigurable;
          if (isConfigurable) {
            this.props.history.replace(`/${slug}/config`);
          } else if (hasPreflight) {
            this.props.history.replace(`/${slug}/preflight`);
          } else {
            this.props.history.replace(`/app/${slug}`);
          }
        });
      }
    }, 1000);
  };

  onProgressError = async (errorMessage) => {
    const { slug } = this.props.match.params;

    let supportBundleCommand = [];
    try {
      supportBundleCommand = "kubectl support-bundle --load-cluster-specs";
    } catch (err) {
      console.log(err);
    }

    // Push this setState call to the end of the call stack
    setTimeout(() => {
      Object.entries(COMMON_ERRORS).forEach(([errorString, message]) => {
        if (errorMessage.includes(errorString)) {
          errorMessage = message;
        }
      });

      this.setState({
        errorMessage,
        fileUploading: false,
        uploadProgress: 0,
        uploadSize: 0,
        uploadResuming: false,
        supportBundleCommand: supportBundleCommand,
      });
    }, 0);
  };

  onProgressSuccess = async () => {
    const { onUploadSuccess, match } = this.props;

    await onUploadSuccess();

    const app = await this.getApp(match.params.slug);

    if (app?.isConfigurable) {
      this.props.history.replace(`/${app.slug}/config`);
    } else if (app?.hasPreflight) {
      this.props.history.replace(`/${app.slug}/preflight`);
    } else {
      this.props.history.replace(`/app/${app.slug}`);
    }
  };

  getApp = async (slug) => {
    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/app/${slug}`, {
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
        method: "GET",
      });
      if (res.ok && res.status == 200) {
        const app = await res.json();
        return app;
      }
    } catch (err) {
      console.log(err);
    }
    return null;
  };

  toggleViewOnlineInstallErrorMessage = () => {
    this.setState({
      viewOnlineInstallErrorMessage: !this.state.viewOnlineInstallErrorMessage,
    });
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  };

  render() {
    const { appName, logo, fetchingMetadata, showRegistry, appsListLength } =
      this.props;

    const { slug } = this.props.match.params;

    const {
      bundleFile,
      fileUploading,
      uploadProgress,
      uploadSize,
      uploadResuming,
      errorMessage,
      registryDetails,
      preparingOnlineInstall,
      onlineInstallErrorMessage,
      viewOnlineInstallErrorMessage,
      supportBundleCommand,
    } = this.state;

    const hasFile = bundleFile && !isEmpty(bundleFile);

    if (fileUploading) {
      return (
        <AirgapUploadProgress
          appSlug={slug}
          total={uploadSize}
          progress={uploadProgress}
          resuming={uploadResuming}
          onProgressError={this.onProgressError}
          onProgressSuccess={this.onProgressSuccess}
        />
      );
    }

    let logoUri;
    let applicationName;
    if (appsListLength && appsListLength > 1) {
      logoUri =
        "https://cdn2.iconfinder.com/data/icons/mixd/512/16_kubernetes-512.png";
      applicationName = "";
    } else {
      logoUri = logo;
      applicationName = appName;
    }

    return (
      <div className="UploadLicenseFile--wrapper container flex-column u-overflow--auto u-marginTop--auto u-marginBottom--auto alignItems--center">
        <KotsPageTitle pageName="Air Gap Installation" showAppSlug />
        <div className="LoginBox-wrapper u-flexTabletReflow flex-auto u-marginTop--20 u-marginBottom--5">
          <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
            <div className="flex-column alignItems--center">
              <div className="flex">
                {logo ? (
                  <span
                    className="icon brand-login-icon u-marginRight--10"
                    style={{ backgroundImage: `url(${logoUri})` }}
                  />
                ) : !fetchingMetadata ? (
                  <span className="icon kots-login-icon u-marginRight--10" />
                ) : (
                  <span style={{ width: "60px", height: "60px" }} />
                )}
                <span className="icon airgapBundleIcon" />
              </div>
            </div>
            {preparingOnlineInstall ? (
              <div className="flex-column alignItems--center u-marginTop--30">
                <LicenseUploadProgress hideProgressBar={true} />
              </div>
            ) : (
              <div>
                <p className="u-marginTop--10 u-paddingTop--5 u-fontSize--header u-textColor--primary u-fontWeight--bold">
                  Install in airgapped environment
                </p>
                <p className="u-marginTop--10 u-marginTop--5 u-fontSize--large u-textAlign--center u-fontWeight--medium u-lineHeight--normal u-textColor--bodyCopy">
                  {showRegistry
                    ? `To install on an airgapped network, you will need to provide access to a Docker registry. The images ${
                        applicationName?.length > 0
                          ? `in ${applicationName}`
                          : ""
                      } will be retagged and pushed to the registry that you provide here.`
                    : `To install on an airgapped network, the images ${
                        applicationName?.length > 0
                          ? `in ${applicationName}`
                          : ""
                      } will be uploaded from the bundle you provide to the cluster.`}
                </p>
                {showRegistry && (
                  <div className="u-marginTop--30">
                    <AirgapRegistrySettings
                      app={null}
                      hideCta={true}
                      hideTestConnection={true}
                      namespaceDescription="What namespace do you want the application images pushed to?"
                      gatherDetails={this.getRegistryDetails}
                      registryDetails={registryDetails}
                      showHostnameAsRequired={
                        errorMessage === this.emptyHostnameErrMessage
                      }
                    />
                  </div>
                )}
                <div className="u-marginTop--20 flex">
                  {this.state.airgapUploader ? (
                    <MountAware
                      onMount={(el) =>
                        this.state.airgapUploader.assignElement(el)
                      }
                      className={classNames("FileUpload-wrapper", "flex1", {
                        "has-file": hasFile,
                        "has-error": errorMessage,
                      })}
                    >
                      {hasFile ? (
                        <div className="has-file-wrapper">
                          <p className="u-fontSize--normal u-fontWeight--medium tw-pl-2">
                            {bundleFile.name}
                          </p>
                        </div>
                      ) : (
                        <div className="u-textAlign--center">
                          <p className="u-fontSize--normal u-textColor--secondary u-fontWeight--medium u-lineHeight--normal">
                            Drag your airgap bundle here or{" "}
                            <span className="link u-textDecoration--underlineOnHover">
                              choose a bundle to upload
                            </span>
                          </p>
                          <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--normal u-lineHeight--normal u-marginTop--10">
                            This will be a .airgap file.
                          </p>
                        </div>
                      )}
                    </MountAware>
                  ) : null}
                  {hasFile && (
                    <div className="flex-auto flex-column u-marginLeft--10 justifyContent--center">
                      <button
                        type="button"
                        className="btn primary large flex-auto"
                        onClick={this.uploadAirgapBundle}
                        disabled={fileUploading || !hasFile}
                      >
                        {fileUploading ? "Uploading" : "Upload airgap bundle"}
                      </button>
                    </div>
                  )}
                </div>
                {errorMessage && (
                  <div className="u-marginTop--10">
                    <span className="u-textColor--error">{errorMessage}</span>
                    {this.state.showSupportBundleCommand ? (
                      <div className="u-marginTop--10">
                        <h2 className="u-fontSize--larger u-fontWeight--bold u-textColor--primary">
                          Run this command in your cluster
                        </h2>
                        <CodeSnippet
                          language="bash"
                          canCopy={true}
                          onCopyText={
                            <span className="u-textColor--success">
                              Command has been copied to your clipboard
                            </span>
                          }
                        >
                          {supportBundleCommand}
                        </CodeSnippet>
                      </div>
                    ) : supportBundleCommand ? (
                      <div>
                        <div className="u-marginTop--10">
                          <a href="#" onClick={this.toggleShowRun}>
                            Click here
                          </a>{" "}
                          to get a command to generate a support bundle.
                        </div>
                      </div>
                    ) : null}
                  </div>
                )}
                {hasFile && (
                  <div className="u-marginTop--10">
                    <span
                      className="link u-fontSize--small"
                      onClick={this.clearFile}
                    >
                      Select a different bundle
                    </span>
                  </div>
                )}
              </div>
            )}
          </div>
        </div>
        <div
          className={classNames(
            "u-marginTop--10 u-textAlign--center",
            { "u-marginBottom--20": !onlineInstallErrorMessage },
            { "u-display--none": preparingOnlineInstall }
          )}
        >
          <span
            className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium"
            onClick={this.handleOnlineInstall}
          >
            Optionally you can{" "}
            <span className="link">
              download{" "}
              {applicationName?.length > 0
                ? applicationName
                : "this application"}{" "}
              from the Internet
            </span>
          </span>
        </div>
        {onlineInstallErrorMessage && (
          <div className="u-marginTop--10 u-marginBottom--20">
            <span className="u-fontSize--small u-textColor--error u-marginRight--5 u-fontWeight--bold">
              Unable to install license
            </span>
            <span
              className="u-fontSize--small link"
              onClick={this.toggleViewOnlineInstallErrorMessage}
            >
              view more
            </span>
          </div>
        )}

        <Modal
          isOpen={viewOnlineInstallErrorMessage}
          onRequestClose={this.toggleViewOnlineInstallErrorMessage}
          contentLabel="Online install error message"
          ariaHideApp={false}
          className="Modal"
        >
          <div className="Modal-body">
            <div className="ExpandedError--wrapper u-marginTop--10 u-marginBottom--10">
              <p className="u-fontSize--small u-fontWeight--bold u-textColor--primary u-marginBottom--5">
                Error description
              </p>
              <p className="u-fontSize--small u-textColor--error">
                {onlineInstallErrorMessage}
              </p>
              <p className="u-fontSize--small u-fontWeight--bold u-marginTop--15 u-textColor--primary">
                Run this command to generate a support bundle
              </p>
              <CodeSnippet
                language="bash"
                canCopy={true}
                onCopyText={
                  <span className="u-textColor--success">
                    Command has been copied to your clipboard
                  </span>
                }
              >
                kubectl support-bundle https://kots.io
              </CodeSnippet>
            </div>
            <button
              type="button"
              className="btn primary u-marginTop--15"
              onClick={this.toggleViewOnlineInstallErrorMessage}
            >
              Ok, got it!
            </button>
          </div>
        </Modal>
      </div>
    );
  }
}

// eslint-disable-next-line
export default withRouter(UploadAirgapBundle);
