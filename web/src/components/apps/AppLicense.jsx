import React, { Component } from "react";
import Helmet from "react-helmet";
import Dropzone from "react-dropzone";
import yaml from "js-yaml";
import classNames from "classnames";
import size from "lodash/size";
import Modal from "react-modal";
import { Link } from "react-router-dom";
import { getFileContent, Utilities, getLicenseExpiryDate } from "../../utilities/utilities";
import Loader from "../shared/Loader";

import "@src/scss/components/apps/AppLicense.scss";

class AppLicense extends Component {

  constructor(props) {
    super(props);

    this.state = {
      appLicense: null,
      loading: false,
      message: "",
      messageType: "info",
      showNextStepModal: false,
      entitlementsToShow: []
    }
  }

  getAppLicense = async () => {
    await fetch(`${window.env.API_ENDPOINT}/app/${this.props.app.slug}/license`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    }).then(async (res) => {
      const body = await res.json();
      if (body === null) {
        this.setState({ appLicense: {} });
      } else if (body.success) {
        this.setState({ appLicense: body.license });
      } else if (body.error) {
        console.log(body.error);
      }
    }).catch((err) => {
      console.log(err)
    });
  }

  componentDidMount() {
    this.getAppLicense();
  }

  onDrop = async (files) => {
    const content = await getFileContent(files[0]);
    const contentStr = (new TextDecoder("utf-8")).decode(content)
    const airgapLicense = await yaml.safeLoad(contentStr);
    const { appLicense } = this.state;

    if (airgapLicense.spec?.licenseID !== appLicense?.id) {
      this.setState({
        message: "Licenses do not match",
        messageType: "error"
      });
      return;
    }

    if (airgapLicense.spec?.licenseSequence === appLicense?.licenseSequence) {
      this.setState({
        message: "License is already up to date",
        messageType: "info"
      });
      return;
    }

    this.syncAppLicense(contentStr);
  }

  syncAppLicense = (licenseData) => {
    this.setState({
      loading: true,
      message: "",
      messageType: "info",
    });

    const { app } = this.props;

    const payload = {
      licenseData,
    };

    fetch(`${window.env.API_ENDPOINT}/app/${app.slug}/license`, {
      method: "PUT",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload)
    })
      .then(async response => {
        if (!response.ok) {
          if (response.status == 401) {
            Utilities.logoutUser();
            return;
          }
          const res = await response.json();
          throw new Error(res?.error);
        }
        return response.json();
      })
      .then(async (licenseResponse) => {
        let message;
        if (!licenseResponse.synced) {
          message = "License is already up to date"
        } else if (app.isAirgap) {
          message = "License uploaded successfully"
        } else {
          message = "License synced successfully"
        }

        this.setState({
          appLicense: licenseResponse.license,
          message,
          messageType: "info",
          showNextStepModal: licenseResponse.synced
        });

        if (this.props.syncCallback) {
          this.props.syncCallback();
        }
      })
      .catch(err => {
        console.log(err);
        this.setState({
          message: err ? err.message : "Something went wrong",
          messageType: "error"
        });
      })
      .finally(() => {
        this.setState({ loading: false });
      });
  }

  hideNextStepModal = () => {
    this.setState({ showNextStepModal: false });
  }

  toggleShowDetails = (entitlement) => {
    this.setState({ entitlementsToShow: [...this.state.entitlementsToShow, entitlement] })
  }

  toggleHideDetails = (entitlement) => {
    let entitlementsToShow = [...this.state.entitlementsToShow];
    const index = this.state.entitlementsToShow.indexOf(entitlement);
    entitlementsToShow.splice(index, 1);
    this.setState({ entitlementsToShow })
  }

  render() {
    const { appLicense, loading, message, messageType, showNextStepModal } = this.state;

    if (!appLicense) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    const { app } = this.props;
    const expiresAt = getLicenseExpiryDate(appLicense);
    const gitops = app?.downstreams?.length && app.downstreams[0]?.gitops;
    const appName = app?.name || "Your application";


    return (
      <div className="flex flex-column justifyContent--center alignItems--center">
        <Helmet>
          <title>{`${appName} License`}</title>
        </Helmet>
        {size(appLicense) > 0 ?
          <div className="License--wrapper flex-column">
            <div className="flex flex-auto alignItems--center">
              <span className="u-fontSize--large u-fontWeight--bold u-lineHeight--normal u-textColor--primary"> License </span>
              {appLicense?.licenseType === "community" &&
                <div className="flex-auto">
                  <span className="CommunityEditionTag u-marginLeft--10"> Community Edition </span>
                  <span className="u-fontSize--small u-fontWeight--normal u-lineHeight--normal u-marginLeft--10" style={{ color: "#A5A5A5" }}> To change your license, please contact your account representative. </span>
                </div>}
            </div>
            <div className="LicenseDetails flex flex1 justifyContent--spaceBetween">
              <div className="Details--wrapper flex-auto flex-column">
                <div className="flex flex-auto alignItems--center">
                  <span className="u-fontSize--larger u-fontWeight--bold u-lineHeight--normal u-textColor--secondary"> {appLicense.assignee} </span>
                  {appLicense?.channelName &&
                    <span className="channelTag flex-auto alignItems--center u-fontWeight--medium u-marginLeft--10"> {appLicense.channelName} </span>}
                </div>
                <div className="flex flex1 alignItems--center u-marginTop--5">
                  <div className={`LicenseTypeTag ${appLicense?.licenseType} flex-auto flex-verticalCenter alignItems--center`}>
                    <span className={`icon ${appLicense?.licenseType === "---" ? "" : appLicense?.licenseType}-icon`}></span>
                    {appLicense?.licenseType !== "---"
                      ? `${Utilities.toTitleCase(appLicense.licenseType)} license`
                      : `---`}
                  </div>
                  <p className={`u-fontWeight--medium u-fontSize--small u-lineHeight--normal u-marginLeft--10 ${Utilities.checkIsDateExpired(expiresAt) ? "u-textColor--error" : "u-textColor--info"}`}>
                    {expiresAt === "Never" ? "Does not expire" : Utilities.checkIsDateExpired(expiresAt) ? `Expired ${expiresAt}` : `Expires ${expiresAt}`}
                  </p>
                </div>
                {size(appLicense?.entitlements) > 0 &&
                  <div className="flexWrap--wrap flex-auto flex1 u-marginTop--12">
                    {appLicense.entitlements?.map((entitlement, i) => {
                      const currEntitlement = this.state.entitlementsToShow.find(f => f === entitlement.title);
                      const isTextField = entitlement.valueType === "Text";
                      const isBooleanField = entitlement.valueType === "Boolean";
                      if (entitlement.value.length > 30 && (currEntitlement !== entitlement.title)) {
                        return (
                          <span key={entitlement.label} className={`u-fontSize--small u-lineHeight--normal u-textColor--secondary u-fontWeight--medium u-marginRight--10 ${i !== 0 ? "u-marginLeft--5" : ""}`}> {entitlement.title}: <span className={`u-fontWeight--bold ${isTextField && "u-fontFamily--monospace"}`}> {entitlement.value.slice(0, 30) + "..."} </span>
                            <span className="icon clickable down-arrow-icon" onClick={() => this.toggleShowDetails(entitlement.title)} />
                          </span>
                        )
                      } else if (entitlement.value.length > 30 && (currEntitlement === entitlement.title)) {
                        return (
                          <span key={entitlement.label} className={`flex-column u-fontSize--small u-lineHeight--normal u-textColor--secondary u-fontWeight--medium u-marginRight--10 ${i !== 0 ? "u-marginLeft--5" : ""}`}> {entitlement.title}: <span className={`u-fontWeight--bold ${isTextField && "u-fontFamily--monospace"}`} style={{whiteSpace: "pre"}}> {entitlement.value} </span>
                            <span className="icon clickable up-arrow-icon u-marginTop--5 u-marginLeft--5" onClick={() => this.toggleHideDetails(entitlement.title)} />
                          </span>
                        )
                      } else {
                        return (
                          <span key={entitlement.label} className={`u-fontSize--small u-lineHeight--normal u-textColor--secondary u-fontWeight--medium u-marginRight--10 ${i !== 0 ? "u-marginLeft--5" : ""}`}> {entitlement.title}: <span className={`u-fontWeight--bold ${isTextField && "u-fontFamily--monospace"}`}> {isBooleanField ? entitlement.value.toString() : entitlement.value} </span></span>
                        );
                      }
                    })}
                  </div>}
                <div className="flexWrap--wrap flex alignItems--center">
                  {appLicense?.isAirgapSupported ? <span className="flex alignItems--center u-fontWeight--medium u-fontSize--small u-lineHeight--normal u-textColor--secondary u-marginRight--10 u-marginTop--10"><span className="icon licenseAirgapIcon" /> Airgap enabled </span> : null}
                  {appLicense?.isSnapshotSupported ? <span className="flex alignItems--center u-fontWeight--medium u-fontSize--small u-lineHeight--normal u-textColor--secondary u-marginLeft--5 u-marginRight--10 u-marginTop--10"><span className="icon licenseVeleroIcon" /> Snapshots enabled </span> : null}
                  {appLicense?.isGitOpsSupported ? <span className="flex alignItems--center u-fontWeight--medium u-fontSize--small u-lineHeight--normal u-textColor--secondary u-marginLeft--5 u-marginRight--10 u-marginTop--10"><span className="icon licenseGithubIcon" /> GitOps enabled </span> : null}
                  {appLicense?.isIdentityServiceSupported ? <span className="flex alignItems--center u-fontWeight--medium u-fontSize--small u-lineHeight--normal u-textColor--secondary u-marginLeft--5 u-marginRight--10 u-marginTop--10"><span className="icon licenseIdentityIcon" /> Identity Service enabled </span> : null}
                  {appLicense?.isGeoaxisSupported ? <span className="flex alignItems--center u-fontWeight--medium u-fontSize--small u-lineHeight--normal u-textColor--secondary u-marginLeft--5 u-marginRight--10 u-marginTop--10"><span className="icon licenseGeoaxisIcon" /> GEOAxIS Provider enabled </span> : null}
                </div>
              </div>
              <div className="flex-column flex-auto alignItems--flexEnd justifyContent--center">
                {app.isAirgap ?
                  <Dropzone
                    className="Dropzone-wrapper"
                    accept={["application/x-yaml", ".yaml", ".yml"]}
                    onDropAccepted={this.onDrop}
                    multiple={false}
                  >
                    <button className="btn primary blue" disabled={loading}>{loading ? "Uploading" : "Upload license"}</button>
                  </Dropzone>
                  :
                  <button className="btn primary blue" disabled={loading} onClick={this.syncAppLicense.bind(this, "")}>{loading ? "Syncing" : "Sync license"}</button>
                }
                {message &&
                  <p className={classNames("u-fontWeight--bold u-fontSize--small", {
                    "u-textColor--error": messageType === "error",
                    "u-textColor--primary": messageType === "info",
                  })}>{message}</p>
                }
              </div>
            </div>
          </div>
          :
          <div>
            <p className="u-fontSize--large u-textColor--bodyCopy u-marginTop--15 u-lineHeight--more"> License data is not available on this application because it was installed via Helm </p>
          </div>
        }
        <Modal
          isOpen={showNextStepModal}
          onRequestClose={this.hideNextStepModal}
          shouldReturnFocusAfterClose={false}
          contentLabel="Next step"
          ariaHideApp={false}
          className="Modal MediumSize"
        >
          {gitops?.enabled ?
            <div className="Modal-body">
              <p className="u-fontSize--large u-textColor--primary u-lineHeight--medium u-marginBottom--20">
                The license for {appName} has been updated. A new commit has been made to the gitops repository with these changes. Please head to the <a className="link" target="_blank" href={gitops?.uri} rel="noopener noreferrer">repo</a> to see the diff.
              </p>
              <div className="flex justifyContent--flexEnd">
                <button type="button" className="btn blue primary" onClick={this.hideNextStepModal}>Ok, got it!</button>
              </div>
            </div>
            :
            <div className="Modal-body">
              <p className="u-fontSize--large u-textColor--primary u-lineHeight--medium u-marginBottom--20">
                The license for {appName} has been updated. A new version is available on the version history page with these changes.
              </p>
              <div className="flex justifyContent--flexEnd">
                <button type="button" className="btn blue secondary u-marginRight--10" onClick={this.hideNextStepModal}>Cancel</button>
                <Link to={`/app/${app?.slug}/version-history`}>
                  <button type="button" className="btn blue primary">Go to new version</button>
                </Link>
              </div>
            </div>
          }
        </Modal>
      </div>
    );
  }
}

export default (AppLicense);
