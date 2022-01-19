import * as React from "react";
import { withRouter, Link } from "react-router-dom";
import { Helmet } from "react-helmet";
import Dropzone from "react-dropzone";
import yaml from "js-yaml";
import size from "lodash/size";
import isEmpty from "lodash/isEmpty";
import keyBy from "lodash/keyBy";
import Modal from "react-modal";
import Select from "react-select";
import { getFileContent, Utilities } from "../utilities/utilities";
import CodeSnippet from "./shared/CodeSnippet";
import LicenseUploadProgress from "./LicenseUploadProgress";

import "../scss/components/troubleshoot/UploadSupportBundleModal.scss";
import "../scss/components/UploadLicenseFile.scss";

class UploadLicenseFile extends React.Component {
  state = {
    licenseFile: {},
    licenseFileContent: null,
    fileUploading: false,
    errorMessage: "",
    viewErrorMessage: false,
    licenseExistErrData: {},
    selectedAppToInstall: {}
  }

  clearFile = () => {
    this.setState({ licenseFile: {}, licenseFileContent: null, errorMessage: "", viewErrorMessage: false });
  }

  moveBar = (count) => {
    const elem = document.getElementById("myBar");
    const percent = count > 3 ? 96 : count * 30;
    if (elem) {
      elem.style.width = percent + "%";
    }
  }

  componentDidMount() {
    const { appSlugFromMetadata } = this.props;

    if (appSlugFromMetadata) {
      const hasChannelAsPartOfASlug = appSlugFromMetadata.includes("/");
      let appSlug;
      if (hasChannelAsPartOfASlug) {
        const splitAppSlug = appSlugFromMetadata.split("/");
        appSlug = splitAppSlug[0]
      } else {
        appSlug = appSlugFromMetadata;
      }
      this.setState({
        selectedAppToInstall: {
          ...this.state.selectedAppToInstall,
          value: appSlug,
          label: appSlugFromMetadata
        }
      })
    }
  }

  uploadLicenseFile = async () => {
    const { onUploadSuccess, history } = this.props;
    const { licenseFile, licenseFileContent, hasMultiApp } = this.state;
    const isRliFile = licenseFile.name.substr(licenseFile.name.lastIndexOf(".")) === ".rli";
    let licenseText;

    let serializedLicense;
    if (isRliFile) {
      try {
        const base64String = btoa(String.fromCharCode.apply(null, new Uint8Array(licenseFileContent)));
        licenseText = await this.exchangeRliFileForLicense(base64String);
      } catch (err) {
        this.setState({
          fileUploading: false,
          errorMessage: err,
        });
        return;
      }
    } else {
      licenseText = hasMultiApp ? licenseFileContent[this.state.selectedAppToInstall.value] : licenseFileContent;
      serializedLicense = yaml.dump(licenseText)
    }

    this.setState({
      fileUploading: true,
      errorMessage: "",
    });

    let data;
    fetch(`${window.env.API_ENDPOINT}/license`, {
      method: "POST",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        licenseData: isRliFile ? licenseText : serializedLicense,
      }),
    })
      .then(async (result) => {
        data = await result.json();
      })
      .catch(err => {
        this.setState({
          fileUploading: false,
          errorMessage: err,
        });
        return;
      })

    let count = 0;
    const interval = setInterval(() => {
      if (this.state.errorMessage.length) {
        clearInterval(interval);
      }
      count++
      this.moveBar(count);
      if (count > 3) {
        if (data) {
          clearInterval(interval);

          if (!data.success) {
            const licenseExistErr = data?.error?.includes("License already exist");
            this.setState({
              fileUploading: false,
              errorMessage: data.error,
              licenseExistErrData: licenseExistErr ? data : ""
            });
            return;
          }

          // When successful, refetch all the user's apps with onUploadSuccess
          onUploadSuccess().then(() => {
            if (data.isAirgap) {
              if (data.needsRegistry) {
                history.replace(`/${data.slug}/airgap`);
              } else {
                history.replace(`/${data.slug}/airgap-bundle`);
              }
              return;
            }

            if (data.isConfigurable) {
              history.replace(`/${data.slug}/config`);
              return;
            }

            if (data.hasPreflight) {
              history.replace(`/${data.slug}/preflight`);
              return;
            }

            // No airgap, config or preflight? Go to the kotsApp detail view that was just uploaded
            if (data) {
              history.replace(`/app/${data.slug}`);
            }
          });
        }
      }
    }, 1000);
  }

  setAvailableAppOptions = (arr) => {
    let availableAppOptions = [];
    arr.map((option) => {
      const label = option.spec.channelName !== "Stable" ? `${option.spec.appSlug}/${option.spec.channelName}` : option.spec.appSlug;
      availableAppOptions.push({
        value: option.spec.appSlug,
        label: label
      });
    });
    this.setState({
      selectedAppToInstall: availableAppOptions[0],
      availableAppOptions: availableAppOptions
    });
  }

  onDrop = async (files) => {
    const content = await getFileContent(files[0]);
    const parsedLicenseYaml = (new TextDecoder("utf-8")).decode(content);
    let licenseYamls;
    try {
      licenseYamls = yaml.loadAll(parsedLicenseYaml);
    } catch (e) {
      console.log(e);
      this.setState({ errorMessage: "Faild to parse license file" });
      return;
    }
    const hasMultiApp = licenseYamls.length > 1;
    if (hasMultiApp) {
      this.setAvailableAppOptions(licenseYamls);
    }
    this.setState({
      licenseFile: files[0],
      licenseFileContent: hasMultiApp ? keyBy(licenseYamls, (option) => { return option.spec.appSlug }) : licenseYamls[0],
      errorMessage: "",
      hasMultiApp,
    });
  }

  exchangeRliFileForLicense = async (content) => {
    return new Promise((resolve, reject) => {
      const payload = {
        licenseData: content,
      };

      fetch(`${window.env.API_ENDPOINT}/license/platform`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "Accept": "application/json",
          "Authorization": Utilities.getToken(),
        },
        body: JSON.stringify(payload),
      })
        .then(async (res) => {
          if (!res.ok) {
            reject(res.status === 401 ? "Invalid license. Please try again" : "There was an error uploading your license. Please try again");
            return;
          }
          resolve((await res.json()).licenseData);
        })
        .catch((err) => {
          console.log(err);
          reject("There was an error uploading your license. Please try again");
        });
    });
  }

  toggleViewErrorMessage = () => {
    this.setState({
      viewErrorMessage: !this.state.viewErrorMessage
    });
  }

  startRestore = snapshot => {
    this.setState({ startingRestore: true, startingRestoreMsg: "" });

    const payload = {
      license: this.state.licenseFile
    }

    fetch(`${window.env.API_ENDPOINT}/snapshot/${snapshot.name}/restore`, {
      method: "POST",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload)
    })
      .then(async (res) => {
        const startRestoreResponse = await res.json();
        if (!res.ok) {
          this.setState({
            startingRestore: false,
            startingRestoreMsg: startRestoreResponse.error
          })
          return;
        }

        if (startRestoreResponse.success) {
          this.setState({
            startingRestore: false
          });
        } else {
          this.setState({
            startingRestore: false,
            startingRestoreMsg: startRestoreResponse.error
          })
        }
      })
      .catch((err) => {
        this.setState({
          startingRestore: false,
          startingRestoreMsg: err.message ? err.message : "Something went wrong, please try again."
        });
      });
  }

  handleUploadStatusErr = (errMessage) => {
    this.setState({
      fileUploading: false,
      errorMessage: errMessage
    })
  }

  getLabel = (label) => {
    return (
      <div style={{ alignItems: "center", display: "flex" }}>
        <span style={{ fontSize: 18, marginRight: "10px" }}><span className="app-icon" /></span>
        <span style={{ fontSize: 14 }}>{label}</span>
      </div>
    );
  }

  onAppToInstallChange = (selectedAppToInstall) => {
    this.setState({ selectedAppToInstall });
  }


  render() {
    const {
      appName,
      logo,
      fetchingMetadata,
      appsListLength,
      isBackupRestore,
      snapshot,
      appSlugFromMetadata
    } = this.props;
    const { licenseFile, fileUploading, errorMessage, viewErrorMessage, licenseExistErrData, selectedAppToInstall, hasMultiApp } = this.state;
    const hasFile = licenseFile && !isEmpty(licenseFile);

    let logoUri;
    let applicationName;
    if (appsListLength && appsListLength > 1) {
      logoUri = "https://cdn2.iconfinder.com/data/icons/mixd/512/16_kubernetes-512.png";
      applicationName = "";
    } else {
      logoUri = logo;
      applicationName = appSlugFromMetadata ? appSlugFromMetadata : appName;
    }

    // TODO remove when restore is enabled
    const isRestoreEnabled = false;

    return (
      <div className={`UploadLicenseFile--wrapper ${isBackupRestore ? "" : "container"} flex-column flex1 u-overflow--auto Login-wrapper justifyContent--center alignItems--center`}>
        <Helmet>
          <title>{`${applicationName ? `${applicationName} Admin Console` : "Admin Console"}`}</title>
        </Helmet>
        <div className="LoginBox-wrapper u-flexTabletReflow  u-flexTabletReflow flex-auto">
          <div className="flex-auto flex-column login-form-wrapper secure-console justifyContent--center">
            <div className="flex-column alignItems--center">
              {logo
                ? <span className="icon brand-login-icon" style={{ backgroundImage: `url(${logoUri})` }} />
                : !fetchingMetadata ? <span className="icon kots-login-icon" />
                  : <span style={{ width: "60px", height: "60px" }} />
              }
            </div>
            {!fileUploading ?
              <div className="flex flex-column">
                <p className="u-fontSize--header u-textColor--primary u-fontWeight--bold u-textAlign--center u-marginTop--10 u-paddingTop--5"> {`${isBackupRestore ? "Verify your license" : "Upload your license file"}`} </p>
                <div className="u-marginTop--30">
                  <div className={`FileUpload-wrapper flex1 ${hasFile ? "has-file" : ""}`}>
                    {hasFile ?
                      <div className="has-file-wrapper">
                        <div className="flex">
                          <div className="icon u-yamlLtGray-small u-marginRight--10" />
                          <div>
                            <p className="u-fontSize--normal u-textColor--primary u-fontWeight--medium">{licenseFile.name}</p>
                            <span className="replicated-link u-fontSize--small" onClick={this.clearFile}>Select a different file</span>
                          </div>
                        </div>
                        {hasMultiApp &&
                          <div className="u-marginTop--15 u-paddingTop--15 u-borderTop--gray">
                            <div>
                              <p className="u-fontSize--small u-fontWeight--medium u-textColor--primary u-lineHeight--normal">Your license has access to {this.state.availableAppOptions.length} applications</p>
                              <p className="u-fontSize--small u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--10">Select the application that you want to install.</p>
                              <Select
                                className="replicated-select-container"
                                classNamePrefix="replicated-select"
                                options={this.state.availableAppOptions}
                                getOptionLabel={(option) => this.getLabel(option.label)}
                                getOptionValue={(option) => option.value}
                                value={selectedAppToInstall}
                                onChange={this.onAppToInstallChange}
                                isOptionSelected={(option) => { option.value === selectedAppToInstall.value }}
                              />
                            </div>
                          </div>
                        }
                      </div>
                      :
                      <Dropzone
                        className="Dropzone-wrapper"
                        accept={["application/x-yaml", ".yaml", ".yml", ".rli"]}
                        onDropAccepted={this.onDrop}
                        multiple={false}
                      >
                        <div className="u-textAlign--center">
                          <div className="icon u-yamlLtGray-lrg u-marginBottom--10" />
                          <p className="u-fontSize--normal u-textColor--secondary u-fontWeight--medium u-lineHeight--normal">Drag your license here or <span className="u-linkColor u-fontWeight--medium u-textDecoration--underlineOnHover">choose a file</span></p>
                          <p className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--normal u-lineHeight--normal u-marginTop--10">This will be a .yaml file. Please contact your account rep if you are unable to locate your license file.</p>
                        </div>
                      </Dropzone>
                    }
                  </div>
                  {hasFile && !isBackupRestore &&
                    <div className="flex-auto flex-column">
                      <div>
                        <button
                          type="button"
                          className="btn primary large flex-auto"
                          onClick={this.uploadLicenseFile}
                          disabled={fileUploading || !hasFile}
                        >
                          {fileUploading ? "Uploading" : "Upload license"}
                        </button>
                      </div>
                    </div>
                  }
                </div>
                {errorMessage && (
                  <div className="u-marginTop--10">
                    <span className="u-fontSize--small u-textColor--error u-marginRight--5 u-fontWeight--bold">Unable to install license</span>
                    <span
                      className="u-fontSize--small replicated-link"
                      onClick={this.toggleViewErrorMessage}>
                      view more
                    </span>
                  </div>
                )}
              </div>
              :
              <div><LicenseUploadProgress onError={this.handleUploadStatusErr} /></div>
            }
          </div>
        </div>

        {!isBackupRestore && isRestoreEnabled &&
          <div className="flex u-marginTop--15 alignItems--center">
            <span className="icon restore-icon" />
            <Link className="u-fontSize--normal u-linkColor u-fontWeight--medium u-textDecoration--underlineOnHover u-marginRight--5" to="/restore">{`Restore ${applicationName ? `${applicationName}` : "app"} from a snapshot`} </Link>
            <span className="icon u-arrow" style={{ marginTop: "2px" }} />
          </div>}
        {isBackupRestore ?
          <button className="btn primary u-marginTop--20" onClick={() => this.startRestore(snapshot)} disabled={!hasFile}> Start restore </button>
          : null}

        <Modal
          isOpen={viewErrorMessage}
          onRequestClose={this.toggleViewErrorMessage}
          contentLabel="Online install error message"
          ariaHideApp={false}
          className="Modal"
        >
          <div className="Modal-body">
            <div className="ExpandedError--wrapper u-marginTop--10 u-marginBottom--10">
              <p className="u-fontSize--small u-fontWeight--bold u-textColor--primary u-marginBottom--5">Error description</p>
              <p className="u-fontSize--small u-textColor--error">{typeof errorMessage === "object" ? "An unknown error orrcured while trying to upload your license. Please try again." : errorMessage}</p>
              {!size(licenseExistErrData) ?
                <div className="flex flex-column">
                  <p className="u-fontSize--small u-fontWeight--bold u-marginTop--15 u-textColor--primary">Run this command to generate a support bundle</p>
                  <CodeSnippet
                    language="bash"
                    canCopy={true}
                    onCopyText={<span className="u-textColor--success">Command has been copied to your clipboard</span>}
                  >
                    kubectl support-bundle https://kots.io
              </CodeSnippet>
                </div> :
                <div className="flex flex-column">
                  <p className="u-fontSize--small u-fontWeight--bold u-marginTop--15 u-textColor--primary">Run this command to remove the app</p>
                  <CodeSnippet
                    language="bash"
                    canCopy={true}
                    onCopyText={<span className="u-textColor--success">Command has been copied to your clipboard</span>}
                  >
                    {licenseExistErrData?.deleteAppCommand}
                  </CodeSnippet>
                </div>
              }
            </div>
            <button type="button" className="btn primary u-marginTop--15" onClick={this.toggleViewErrorMessage}>Ok, got it!</button>
          </div>
        </Modal>
      </div >
    );
  }
}

export default withRouter(UploadLicenseFile);
