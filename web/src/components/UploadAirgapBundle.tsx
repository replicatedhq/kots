import React, { useEffect, useReducer } from "react";
import { useHistory, useRouteMatch } from "react-router";
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

import "../scss/components/troubleshoot/UploadSupportBundleModal.scss";
import "../scss/components/Login.scss";

const COMMON_ERRORS = {
  "HTTP 401": "Registry credentials are invalid",
  "invalid username/password": "Registry credentials are invalid",
  "no such host": "No such host",
};

import { KotsParams } from "@types";

type Props = {
  appName: string | null;
  appsListLength: number;
  logo: string | null;
  fetchingMetadata: boolean;
  onUploadSuccess: () => Promise<void>;
  showRegistry: boolean;
};

type RegistryDetails = {
  hostname: string;
  isReadOnly: boolean;
  namespace: string;
  password: string;
  username: string;
};

type ResumeResult = {
  error?: string;
  hasPreflight: boolean;
  isConfigurable: boolean;
};

type State = {
  airgapUploader: AirgapUploader | null;
  bundleFile: {
    name: string;
  } | null;
  displayErrorModal?: boolean;
  errorMessage: string;
  fileUploading: boolean;
  registryDetails: RegistryDetails | null;
  preparingOnlineInstall: boolean;
  supportBundleCommand?: string | string[];
  showSupportBundleCommand: boolean;
  simultaneousUploads?: number;
  onlineInstallErrorMessage: string;
  uploadProgress: number;
  uploadSize: number;
  uploadResuming: boolean;
  viewOnlineInstallErrorMessage: boolean;
};
const UploadAirgapBundle = (props: Props) => {
  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      airgapUploader: null,
      bundleFile: null,
      errorMessage: "",
      fileUploading: false,
      registryDetails: null,
      preparingOnlineInstall: false,
      showSupportBundleCommand: false,
      onlineInstallErrorMessage: "",
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
      viewOnlineInstallErrorMessage: false,
    }
  );

  const emptyHostnameErrMessage = 'Please enter a value for "Hostname" field';
  const match = useRouteMatch<KotsParams>();
  const history = useHistory();
  const appSlug = match.params.slug;

  const onDropBundle = async (file: { name: string }) => {
    setState({
      bundleFile: file,
      onlineInstallErrorMessage: "",
      errorMessage: "",
    });
  };

  const getAirgapConfig = async () => {
    const configUrl = `${process.env.API_ENDPOINT}/app/${appSlug}/airgap/config`;
    let simultaneousUploads = 3;
    try {
      let res = await fetch(configUrl, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          Authorization: Utilities.getToken(),
        },
      });
      if (res.ok) {
        const response = await res.json();
        simultaneousUploads = response.simultaneousUploads;
      }
    } catch {
      // no-op
    }

    setState({
      airgapUploader: new AirgapUploader(
        false,
        appSlug,
        onDropBundle,
        simultaneousUploads
      ),
    });
  };

  useEffect(() => {
    getAirgapConfig();
  }, []);

  const clearFile = () => {
    setState({ bundleFile: null });
  };

  const toggleShowRun = () => {
    setState({ showSupportBundleCommand: true });
  };

  const onUploadProgress = (
    progress: number,
    size: number,
    resuming = false
  ) => {
    setState({
      uploadProgress: progress,
      uploadSize: size,
      uploadResuming: resuming,
    });
  };

  const onUploadError = (message?: string) => {
    setState({
      fileUploading: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
      errorMessage: message || "Error uploading bundle, please try again",
    });
  };

  const uploadAirgapBundle = async () => {
    const { showRegistry } = props;

    // Reset the airgap upload state
    const resetUrl = `${process.env.API_ENDPOINT}/app/${appSlug}/airgap/reset`;
    try {
      await fetch(resetUrl, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: Utilities.getToken(),
        },
      });
    } catch (error) {
      console.error(error);
      setState({
        fileUploading: false,
        uploadProgress: 0,
        uploadSize: 0,
        uploadResuming: false,
        errorMessage:
          "An error occurred while uploading your airgap bundle. Please try again",
      });
      return;
    }

    setState({
      fileUploading: true,
      errorMessage: "",
      showSupportBundleCommand: false,
      onlineInstallErrorMessage: "",
    });

    if (showRegistry) {
      // TODO: remove isEmpty
      if (isEmpty(state.registryDetails?.hostname)) {
        setState({
          fileUploading: false,
          uploadProgress: 0,
          uploadSize: 0,
          uploadResuming: false,
          errorMessage: emptyHostnameErrMessage,
        });
        return;
      }

      let res;
      try {
        res = await fetch(
          `${process.env.API_ENDPOINT}/app/${appSlug}/registry/validate`,
          {
            method: "POST",
            headers: {
              Authorization: Utilities.getToken(),
              "Content-Type": "application/json",
            },
            body: JSON.stringify({
              hostname: state.registryDetails?.hostname,
              namespace: state.registryDetails?.namespace,
              username: state.registryDetails?.username,
              password: state.registryDetails?.password,
              isReadOnly: state.registryDetails?.isReadOnly,
            }),
          }
        );
      } catch (err) {
        if (err instanceof Error) {
          setState({
            fileUploading: false,
            uploadProgress: 0,
            uploadSize: 0,
            uploadResuming: false,
            errorMessage: err.message,
          });
          return;
        }

        setState({
          fileUploading: false,
          uploadProgress: 0,
          uploadSize: 0,
          uploadResuming: false,
          errorMessage: "Something went wrong when uploading Airgap bundle.",
        });
      }

      const response = await res?.json();
      if (!response.success) {
        let msg =
          "An error occurred while uploading your airgap bundle. Please try again";
        if (response.error) {
          msg = response.error;
        }
        setState({
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
      registryHost: state.registryDetails?.hostname,
      namespace: state.registryDetails?.namespace,
      username: state.registryDetails?.username,
      password: state.registryDetails?.password,
      isReadOnly: state.registryDetails?.isReadOnly,
      simultaneousUploads: state.simultaneousUploads,
    };
    state?.airgapUploader?.upload(params, onUploadProgress, onUploadError);
  };

  const getRegistryDetails = (fields: RegistryDetails) => {
    setState({
      ...state,
      registryDetails: {
        hostname: fields.hostname,
        username: fields.username,
        password: fields.password,
        namespace: fields.namespace,
        isReadOnly: fields.isReadOnly,
      },
    });
  };

  const moveBar = (count: number) => {
    const elem = document.getElementById("myBar");
    const percent = count > 3 ? 96 : count * 30;
    if (elem) {
      elem.style.width = percent + "%";
    }
  };

  console.log("current error state", state.onlineInstallErrorMessage);
  const handleOnlineInstall = async () => {
    setState({
      preparingOnlineInstall: true,
      onlineInstallErrorMessage: "",
    });

    let resumeResult: ResumeResult;
    fetch(`${process.env.API_ENDPOINT}/license/resume`, {
      method: "PUT",
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        slug: appSlug,
      }),
    })
      .then(async (result) => {
        resumeResult = await result.json();
        console.log("resumeResult received", resumeResult);
      })
      .catch((err) => {
        setState({
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
      console.log("interval called", count);
      console.log(state.onlineInstallErrorMessage);
      const { onlineInstallErrorMessage } = state;
      console.log(onlineInstallErrorMessage);
      if (state.onlineInstallErrorMessage.length) {
        console.log("interval cleared");
        clearInterval(interval);
      }
      count += 1;
      moveBar(count);
      if (count > 3) {
        if (!resumeResult) {
          return;
        }
        console.log("resumeResult error", resumeResult.error);

        clearInterval(interval);

        if (resumeResult.error) {
          setState({
            // TODO: use fewer flags
            fileUploading: false,
            errorMessage: resumeResult.error,
            preparingOnlineInstall: false,
            onlineInstallErrorMessage: resumeResult.error,
          });
          return;
        }

        console.log("onUploadSuccess called", props);
        props.onUploadSuccess().then(() => {
          // When successful, refetch all the user's apps with onUploadSuccess
          const hasPreflight = resumeResult.hasPreflight;
          const isConfigurable = resumeResult.isConfigurable;
          if (isConfigurable) {
            history.replace(`/${appSlug}/config`);
          } else if (hasPreflight) {
            history.replace(`/${appSlug}/preflight`);
          } else {
            history.replace(`/app/${appSlug}`);
          }
        });
      }
    }, 1000);
  };

  const getSupportBundleCommand = async () => {
    const res = await fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/app/${appSlug}/supportbundlecommand`,
      {
        method: "POST",
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
        body: JSON.stringify({
          origin: window.location.origin,
        }),
      }
    );
    if (!res.ok) {
      throw new Error(`Unexpected status code: ${res.status}`);
    }
    const response = await res.json();
    return response.command;
  };

  const onProgressError = async (errorMessage: string) => {
    let supportBundleCommand: string[] = [];
    try {
      supportBundleCommand = await getSupportBundleCommand();
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

      setState({
        errorMessage,
        fileUploading: false,
        uploadProgress: 0,
        uploadSize: 0,
        uploadResuming: false,
        supportBundleCommand,
      });
    }, 0);
  };

  const getApp = async () => {
    try {
      const res = await fetch(`${process.env.API_ENDPOINT}/app/${appSlug}`, {
        headers: {
          Authorization: Utilities.getToken(),
          "Content-Type": "application/json",
        },
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

  const onProgressSuccess = async () => {
    const { onUploadSuccess } = props;

    await onUploadSuccess();

    // TODO: refactor to use app hook
    const app = await getApp();

    if (app?.isConfigurable) {
      history.replace(`/${app.slug}/config`);
    } else if (app?.hasPreflight) {
      history.replace(`/${app.slug}/preflight`);
    } else {
      history.replace(`/app/${app.slug}`);
    }
  };

  const toggleViewOnlineInstallErrorMessage = () => {
    setState({
      viewOnlineInstallErrorMessage: !state.viewOnlineInstallErrorMessage,
    });
  };

  const { appName, logo, fetchingMetadata, showRegistry, appsListLength } =
    props;

  const { slug } = match.params;

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
  } = state;

  const hasFile = bundleFile && !isEmpty(bundleFile);

  if (fileUploading) {
    return (
      <AirgapUploadProgress
        appSlug={slug}
        total={uploadSize}
        progress={uploadProgress}
        resuming={uploadResuming}
        onProgressError={onProgressError}
        onProgressSuccess={onProgressSuccess}
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
    applicationName = appName || "";
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
                      applicationName?.length > 0 ? `in ${applicationName}` : ""
                    } will be retagged and pushed to the registry that you provide here.`
                  : `To install on an airgapped network, the images ${
                      applicationName?.length > 0 ? `in ${applicationName}` : ""
                    } will be uploaded from the bundle you provide to the cluster.`}
              </p>
              {showRegistry && (
                <div className="u-marginTop--30">
                  <AirgapRegistrySettings
                    app={null}
                    hideCta={true}
                    hideTestConnection={true}
                    namespaceDescription="What namespace do you want the application images pushed to?"
                    gatherDetails={getRegistryDetails}
                    registryDetails={registryDetails}
                    showHostnameAsRequired={
                      errorMessage === emptyHostnameErrMessage
                    }
                  />
                </div>
              )}
              <div className="u-marginTop--20 flex">
                {state.airgapUploader ? (
                  <MountAware
                    onMount={(el: HTMLDivElement) =>
                      state.airgapUploader?.assignElement(el)
                    }
                    className={classNames("FileUpload-wrapper", "flex1", {
                      "has-file": hasFile,
                      "has-error": errorMessage,
                    })}
                  >
                    {hasFile ? (
                      <div className="has-file-wrapper">
                        <p className="u-fontSize--normal u-fontWeight--medium">
                          {bundleFile.name}
                        </p>
                      </div>
                    ) : (
                      <div className="u-textAlign--center">
                        <p className="u-fontSize--normal u-textColor--secondary u-fontWeight--medium u-lineHeight--normal">
                          Drag your airgap bundle here or{" "}
                          <span className="u-linkColor u-fontWeight--medium u-textDecoration--underlineOnHover">
                            choose a bundle to upload
                          </span>
                        </p>
                        <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--normal u-lineHeight--normal u-marginTop--10">
                          This will be a .airgap file
                          {applicationName?.length > 0
                            ? ` ${applicationName} provided`
                            : ""}
                          . Please contact your account rep if you are unable to
                          locate your .airgap file.
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
                      onClick={uploadAirgapBundle}
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
                  {state.showSupportBundleCommand ? (
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
                        <a
                          href="#"
                          className="replicated-link"
                          onClick={toggleShowRun}
                        >
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
                    className="replicated-link u-fontSize--small"
                    onClick={clearFile}
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
          onClick={handleOnlineInstall}
        >
          Optionally you can{" "}
          <span className="replicated-link">
            download{" "}
            {applicationName?.length > 0 ? applicationName : "this application"}{" "}
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
            className="u-fontSize--small replicated-link"
            onClick={toggleViewOnlineInstallErrorMessage}
          >
            view more
          </span>
        </div>
      )}

      <Modal
        isOpen={viewOnlineInstallErrorMessage}
        onRequestClose={toggleViewOnlineInstallErrorMessage}
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
            onClick={toggleViewOnlineInstallErrorMessage}
          >
            Ok, got it!
          </button>
        </div>
      </Modal>
    </div>
  );
};

export default UploadAirgapBundle;
