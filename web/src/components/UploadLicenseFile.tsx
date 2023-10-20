import { useEffect, useReducer } from "react";
import { Link, useNavigate } from "react-router-dom";
import { KotsPageTitle } from "@components/Head";
// TODO: upgrade this dependency
// @ts-ignore
import Dropzone from "react-dropzone";
import yaml from "js-yaml";
import isEmpty from "lodash/isEmpty";
import keyBy from "lodash/keyBy";
import size from "lodash/size";
// TODO: upgrade this dependency
// @ts-ignore
import Dropzone from "react-dropzone";
import Modal from "react-modal";
import Select from "react-select";

import { KotsPageTitle } from "@components/Head";
import { getFileContent } from "../utilities/utilities";
import Icon from "./Icon";
import LicenseUploadProgress from "./LicenseUploadProgress";
import CodeSnippet from "./shared/CodeSnippet";

import "../scss/components/troubleshoot/UploadSupportBundleModal.scss";
import "../scss/components/UploadLicenseFile.scss";

type LicenseYaml = {
  spec: {
    appSlug: string;
    channelName: string;
  };
};

type SelectedAppToInstall = {
  label: string;
  value: string;
};

type State = {
  availableAppOptions?: SelectedAppToInstall[];
  errorMessage: string;
  fileUploading: boolean;
  hasMultiApp?: boolean;
  licenseExistErrData: UploadLicenseResponse | null | string;
  licenseFile: { name: string } | null;
  licenseFileContent: {
    [key: string]: string;
  } | null;
  selectedAppToInstall: SelectedAppToInstall | null;
  startingRestore?: boolean;
  startingRestoreMsg?: string;
  viewErrorMessage: boolean;
};

type UploadLicenseResponse = {
  deleteAppCommand?: string;
  error?: string;
  hasPreflight?: boolean;
  isAirgap: boolean;
  isConfigurable: boolean;
  needsRegistry: boolean;
  slug: string;
  success?: boolean;
};

type Props = {
  appsListLength: number;
  appName: string;
  appSlugFromMetadata: string;
  fetchingMetadata: boolean;
  isBackupRestore?: boolean;
  onUploadSuccess: () => Promise<void>;
  logo: string | null;
  snapshot?: { name: string };
  isEmbeddedCluster: boolean;
};

const UploadLicenseFile = (props: Props) => {
  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      errorMessage: "",
      fileUploading: false,
      licenseExistErrData: null,
      licenseFile: null,
      licenseFileContent: null,
      selectedAppToInstall: null,
      viewErrorMessage: false,
    }
  );

  const navigate = useNavigate();

  const clearFile = () => {
    setState({
      licenseFile: null,
      licenseFileContent: null,
      errorMessage: "",
      viewErrorMessage: false,
    });
  };

  const moveBar = (count: number) => {
    const elem = document.getElementById("myBar");
    const percent = count > 3 ? 96 : count * 30;
    if (elem) {
      elem.style.width = percent + "%";
    }
  };

  useEffect(() => {
    const { appSlugFromMetadata } = props;

    if (appSlugFromMetadata) {
      const hasChannelAsPartOfASlug = appSlugFromMetadata.includes("/");
      let appSlug;
      if (hasChannelAsPartOfASlug) {
        const splitAppSlug = appSlugFromMetadata.split("/");
        appSlug = splitAppSlug[0];
      } else {
        appSlug = appSlugFromMetadata;
      }
      setState({
        selectedAppToInstall: {
          ...state.selectedAppToInstall,
          value: appSlug,
          label: appSlugFromMetadata,
        },
      });
    }
  }, []);

  const exchangeRliFileForLicense = async (content: string) => {
    return new Promise((resolve, reject) => {
      const payload = {
        licenseData: content,
      };

      fetch(`${process.env.API_ENDPOINT}/license/platform`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
        },
        credentials: "include",
        body: JSON.stringify(payload),
      })
        .then(async (res) => {
          if (!res.ok) {
            reject(
              res.status === 401
                ? "Invalid license. Please try again"
                : "There was an error uploading your license. Please try again"
            );
            return;
          }
          resolve((await res.json()).licenseData);
        })
        .catch((err) => {
          console.log(err);
          reject("There was an error uploading your license. Please try again");
        });
    });
  };

  const uploadLicenseFile = async () => {
    const { onUploadSuccess } = props;
    const { licenseFile, licenseFileContent, hasMultiApp } = state;
    const isRliFile =
      licenseFile?.name.substr(licenseFile.name.lastIndexOf(".")) === ".rli";
    let licenseText;

    let serializedLicense;
    if (isRliFile) {
      try {
        const base64String = btoa(
          // TODO: this is probably a bug
          // https://stackoverflow.com/questions/67057689/typscript-type-uint8array-is-missing-the-following-properties-from-type-numb
          // @ts-ignore
          String.fromCharCode.apply(null, new Uint8Array(licenseFileContent))
        );
        licenseText = await exchangeRliFileForLicense(base64String);
      } catch (err) {
        if (err instanceof Error) {
          setState({
            fileUploading: false,
            errorMessage: err.message,
          });
          return;
        }
        setState({
          fileUploading: false,
          errorMessage:
            "Something went wrong while uploading your license. Please try again",
        });
      }
    } else {
      licenseText =
        hasMultiApp && licenseFileContent && state.selectedAppToInstall?.value
          ? licenseFileContent[state.selectedAppToInstall.value]
          : licenseFileContent;
      serializedLicense = yaml.dump(licenseText);
    }

    setState({
      fileUploading: true,
      errorMessage: "",
    });

    let data: UploadLicenseResponse;
    fetch(`${process.env.API_ENDPOINT}/license`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      body: JSON.stringify({
        licenseData: isRliFile ? licenseText : serializedLicense,
      }),
    })
      .then(async (result) => {
        data = await result.json();
      })
      .catch((err) => {
        setState({
          fileUploading: false,
          errorMessage: err,
        });
        return;
      });

    let count = 0;
    const interval = setInterval(() => {
      if (state.errorMessage.length) {
        clearInterval(interval);
      }
      count += 1;
      moveBar(count);
      if (count > 3) {
        if (data) {
          clearInterval(interval);

          if (!data.success) {
            const licenseExistErr = data?.error?.includes(
              "License already exist"
            );
            setState({
              fileUploading: false,
              errorMessage: data.error,
              licenseExistErrData: licenseExistErr ? data : "",
            });
            return;
          }

          // When successful, refetch all the user's apps with onUploadSuccess
          onUploadSuccess().then(() => {
            if (data.isAirgap) {
              if (data.needsRegistry) {
                navigate(`/${data.slug}/airgap`, { replace: true });
              } else {
                navigate(`/${data.slug}/airgap-bundle`, { replace: true });
              }
              return;
            }

            if (props.isEmbeddedCluster) {
              navigate(`/${data.slug}/cluster/manage`, { replace: true });
              return;
            }

            if (data.isConfigurable) {
              navigate(`/${data.slug}/config`, { replace: true });
              return;
            }

            if (data.hasPreflight) {
              navigate(`/${data.slug}/preflight`, { replace: true });
              return;
            }

            // No airgap, config or preflight? Go to the kotsApp detail view that was just uploaded
            if (data) {
              navigate(`/app/${data.slug}`, { replace: true });
            }
          });
        }
      }
    }, 1000);
  };

  const setAvailableAppOptions = (arr: LicenseYaml[]) => {
    let availableAppOptions: SelectedAppToInstall[] = [];
    arr.map((option) => {
      const label =
        option.spec.channelName !== "Stable"
          ? `${option.spec.appSlug}/${option.spec.channelName}`
          : option.spec.appSlug;
      availableAppOptions.push({
        value: option.spec.appSlug,
        label: label,
      });
    });
    setState({
      selectedAppToInstall: availableAppOptions[0],
      availableAppOptions: availableAppOptions,
    });
  };

  const onDrop = async (files: { name: string }[]) => {
    const content = await getFileContent(files[0]);
    // TODO: this is probably a bug
    // @ts-ignore
    const parsedLicenseYaml = new TextDecoder("utf-8").decode(content);
    let licenseYamls;
    try {
      licenseYamls = yaml.loadAll(parsedLicenseYaml);
    } catch (e) {
      console.log(e);
      setState({ errorMessage: "Faild to parse license file" });
      return;
    }
    const hasMultiApp = licenseYamls.length > 1;
    if (hasMultiApp) {
      setAvailableAppOptions(licenseYamls);
    }
    setState({
      licenseFile: files[0],
      licenseFileContent: hasMultiApp
        ? keyBy(licenseYamls, (option) => {
            return option.spec.appSlug;
          })
        : licenseYamls[0],
      errorMessage: "",
      hasMultiApp,
    });
  };

  const toggleViewErrorMessage = () => {
    setState({
      viewErrorMessage: !state.viewErrorMessage,
    });
  };

  const startRestore = (snapshot: { name: string }) => {
    setState({ startingRestore: true, startingRestoreMsg: "" });

    const payload = {
      license: state.licenseFile,
    };

    fetch(`${process.env.API_ENDPOINT}/snapshot/${snapshot.name}/restore`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      body: JSON.stringify(payload),
    })
      .then(async (res) => {
        const startRestoreResponse = await res.json();
        if (!res.ok) {
          setState({
            startingRestore: false,
            startingRestoreMsg: startRestoreResponse.error,
          });
          return;
        }

        if (startRestoreResponse.success) {
          setState({
            startingRestore: false,
          });
        } else {
          setState({
            startingRestore: false,
            startingRestoreMsg: startRestoreResponse.error,
          });
        }
      })
      .catch((err) => {
        setState({
          startingRestore: false,
          startingRestoreMsg: err.message
            ? err.message
            : "Something went wrong, please try again.",
        });
      });
  };

  const handleUploadStatusErr = (errMessage: string) => {
    setState({
      fileUploading: false,
      errorMessage: errMessage,
    });
  };

  const getLabel = (label: string) => {
    return (
      <div style={{ alignItems: "center", display: "flex" }}>
        <span style={{ fontSize: 18, marginRight: "10px" }}>
          <span className="app-icon" />
        </span>
        <span style={{ fontSize: 14 }}>{label}</span>
      </div>
    );
  };

  const onAppToInstallChange = (selectedAppToInstall: SelectedAppToInstall) => {
    setState({ selectedAppToInstall });
  };

  const {
    appName,
    logo,
    fetchingMetadata,
    appsListLength,
    isBackupRestore,
    snapshot,
    appSlugFromMetadata,
  } = props;
  const {
    licenseFile,
    fileUploading,
    errorMessage,
    viewErrorMessage,
    licenseExistErrData,
    selectedAppToInstall,
    hasMultiApp,
  } = state;
  const hasFile = licenseFile && !isEmpty(licenseFile);

  let logoUri;
  let applicationName;
  if (appsListLength && appsListLength > 1) {
    logoUri =
      "https://cdn2.iconfinder.com/data/icons/mixd/512/16_kubernetes-512.png";
    applicationName = "";
  } else {
    logoUri = logo;
    applicationName = appSlugFromMetadata ? appSlugFromMetadata : appName;
  }

  // TODO remove when restore is enabled
  const isRestoreEnabled = false;

  return (
    <div
      className={`UploadLicenseFile--wrapper ${
        isBackupRestore ? "" : "container"
      } flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center`}
    >
      <KotsPageTitle pageName="Upload License" showAppSlug={false} />
      <div className="LoginBox-wrapper u-flexTabletReflow  u-flexTabletReflow flex-auto">
        <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
          <div className="flex-column alignItems--center">
            {logo ? (
              <span
                className="icon brand-login-icon"
                style={{ backgroundImage: `url(${logoUri})` }}
              />
            ) : !fetchingMetadata ? (
              <span className="icon kots-login-icon" />
            ) : (
              <span style={{ width: "60px", height: "60px" }} />
            )}
          </div>
          {!fileUploading ? (
            <div className="flex flex-column">
              <p className="u-fontSize--header u-textColor--primary u-fontWeight--bold u-textAlign--center u-marginTop--10 u-paddingTop--5">
                {" "}
                {`${
                  isBackupRestore
                    ? "Verify your license"
                    : "Upload your license file"
                }`}{" "}
              </p>
              <div className="u-marginTop--30">
                <div
                  className={`FileUpload-wrapper flex1 ${
                    hasFile ? "has-file" : ""
                  }`}
                >
                  {hasFile ? (
                    <div className="has-file-wrapper">
                      <div className="flex">
                        <Icon
                          icon="yaml-icon"
                          size={24}
                          className="u-marginRight--10 gray-color"
                        />
                        <div>
                          <p className="u-fontSize--normal u-textColor--primary u-fontWeight--medium">
                            {licenseFile.name}
                          </p>
                          <span
                            className="link u-fontSize--small"
                            onClick={clearFile}
                          >
                            Select a different file
                          </span>
                        </div>
                      </div>
                      {hasMultiApp && (
                        <div className="u-marginTop--15 u-paddingTop--15 u-borderTop--gray">
                          <div>
                            <p className="u-fontSize--small u-fontWeight--medium u-textColor--primary u-lineHeight--normal">
                              Your license has access to{" "}
                              {state?.availableAppOptions?.length} applications
                            </p>
                            <p className="u-fontSize--small u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--10">
                              Select the application that you want to install.
                            </p>
                            {/* TODO: there's probably a bug here*/}
                            {/*@ts-ignore*/}
                            <Select
                              className="replicated-select-container"
                              classNamePrefix="replicated-select"
                              options={state.availableAppOptions}
                              getOptionLabel={(option) =>
                                getLabel(option.label)
                              }
                              getOptionValue={(option) => option.value}
                              value={selectedAppToInstall}
                              onChange={onAppToInstallChange}
                              isOptionSelected={(option) => {
                                return (
                                  option.value === selectedAppToInstall?.value
                                );
                              }}
                            />
                          </div>
                        </div>
                      )}
                    </div>
                  ) : (
                    <Dropzone
                      className="Dropzone-wrapper"
                      accept={["application/x-yaml", ".yaml", ".yml", ".rli"]}
                      onDropAccepted={onDrop}
                      multiple={false}
                    >
                      <div className="u-textAlign--center">
                        <Icon
                          icon="yaml-icon"
                          size={40}
                          className="u-marginBottom--10 gray-color"
                        />
                        <p className="u-fontSize--normal u-textColor--secondary u-fontWeight--medium u-lineHeight--normal">
                          Drag your license here or{" "}
                          <span className="link u-textDecoration--underlineOnHover">
                            choose a file
                          </span>
                        </p>
                        <p className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--normal u-lineHeight--normal u-marginTop--10">
                          This will be a .yaml file. Please contact your account
                          rep if you are unable to locate your license file.
                        </p>
                      </div>
                    </Dropzone>
                  )}
                </div>
                {hasFile && !isBackupRestore && (
                  <div className="flex-auto flex-column">
                    <div>
                      <button
                        type="button"
                        className="btn primary large flex-auto"
                        onClick={uploadLicenseFile}
                        disabled={fileUploading || !hasFile}
                      >
                        {fileUploading ? "Uploading" : "Upload license"}
                      </button>
                    </div>
                  </div>
                )}
              </div>
              {errorMessage && (
                <div className="u-marginTop--10">
                  <span className="u-fontSize--small u-textColor--error u-marginRight--5 u-fontWeight--bold">
                    Unable to install license
                  </span>
                  <span
                    className="u-fontSize--small link"
                    onClick={toggleViewErrorMessage}
                  >
                    view more
                  </span>
                </div>
              )}
            </div>
          ) : (
            <div>
              <LicenseUploadProgress onError={handleUploadStatusErr} />
            </div>
          )}
        </div>
      </div>

      {!isBackupRestore && isRestoreEnabled && (
        <div className="flex u-marginTop--15 alignItems--center">
          <span className="icon restore-icon" />
          <Link
            className="u-fontSize--normal link u-textDecoration--underlineOnHover u-marginRight--5"
            to="/restore"
          >
            {`Restore ${
              applicationName ? `${applicationName}` : "app"
            } from a snapshot`}{" "}
          </Link>
          <Icon icon="next-arrow" style={{ marginTop: "2px" }} size={9} />
        </div>
      )}
      {isBackupRestore && snapshot ? (
        <button
          className="btn primary u-marginTop--20"
          onClick={() => startRestore(snapshot)}
          disabled={!hasFile}
        >
          {" "}
          Start restore{" "}
        </button>
      ) : null}

      <Modal
        isOpen={viewErrorMessage}
        onRequestClose={toggleViewErrorMessage}
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
              {typeof errorMessage === "object"
                ? "An unknown error orrcured while trying to upload your license. Please try again."
                : errorMessage}
            </p>
            {!size(licenseExistErrData) ? (
              <div className="flex flex-column">
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
            ) : (
              <div className="flex flex-column">
                <p className="u-fontSize--small u-fontWeight--bold u-marginTop--15 u-textColor--primary">
                  Run this command to remove the app
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
                  {typeof licenseExistErrData === "string"
                    ? licenseExistErrData
                    : licenseExistErrData?.deleteAppCommand}
                </CodeSnippet>
              </div>
            )}
          </div>
          <button
            type="button"
            className="btn primary u-marginTop--15"
            onClick={toggleViewErrorMessage}
          >
            Ok, got it!
          </button>
        </div>
      </Modal>
    </div>
  );
};

export default UploadLicenseFile;
