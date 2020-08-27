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
      showNextStepModal: false
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
      } else {
        this.setState({ appLicense: body });
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
    const contentStr = String.fromCharCode.apply(null, new Uint8Array(content));
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
      .then(res => res.json())
      .then(async (latestLicense) => {
        const currentLicense = this.state.appLicense;

        let message;
        if (latestLicense.licenseSequence === currentLicense.licenseSequence) {
          message = "License is already up to date"
        } else if (app.isAirgap) {
          message = "License uploaded successfully"
        } else {
          message = "License synced successfully"
        }

        this.setState({
          appLicense: latestLicense,
          message,
          messageType: "info",
          showNextStepModal: latestLicense.licenseSequence !== currentLicense.licenseSequence
        });

        if (this.props.syncCallback) {
          this.props.syncCallback();
        }
      })
      .catch(err => {
        console.log(err);
      })
      .finally(() => {
        this.setState({ loading: false });
      });
  }

  hideNextStepModal = () => {
    this.setState({ showNextStepModal: false });
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
        {appLicense?.licenseType === "community" &&
          <div className="CommunityLicense--wrapper u-marginTop--30 flex flex1 alignItems--center">
            <div className="flex flex-auto">
              <span className="icon communityIcon"></span>
            </div>
            <div className="flex1 flex-column u-marginLeft--10">
              <p className="u-color--emperor u-fontSize--large u-fontWeight--bold u-lineHeight--medium u-marginBottom--5"> You are running a Community Edition of {appName} </p>
              <p className="u-color--silverChalice u-fontSize--normal u-lineHeight--medium"> To change your license, please contact your account representative. </p>
            </div>
          </div>
        }
        {size(appLicense) > 0 ?
          <div className="LicenseDetails--wrapper u-textAlign--left u-paddingRight--20 u-paddingLeft--20">
            <div className="flex u-marginBottom--20 u-paddingBottom--5 u-marginTop--20 alignItems--center">
              <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-lineHeight--normal u-marginRight--10">License details</p>
            </div>
            <div className="u-color--tundora u-fontSize--normal u-fontWeight--medium">
              <div className="flex u-marginBottom--20">
                <p className="u-marginRight--10">Expires:</p>
                <p className="u-fontWeight--bold u-color--tuna">{expiresAt}</p>
              </div>
              {appLicense.channelName &&
                <div className="flex u-marginBottom--20">
                  <p className="u-marginRight--10">Channel:</p>
                  <p className="u-fontWeight--bold u-color--tuna">{appLicense.channelName}</p>
                </div>
              }
              {appLicense.entitlements?.map(entitlement => {
                return (
                  <div key={entitlement.label} className="flex u-marginBottom--20">
                    <p className="u-marginRight--10">{entitlement.title}</p>
                    <p className="u-fontWeight--bold u-color--tuna">{entitlement.value}</p>
                  </div>
                );
              })}
              {app.isAirgap ?
                <Dropzone
                  className="Dropzone-wrapper"
                  accept={["application/x-yaml", ".yaml", ".yml"]}
                  onDropAccepted={this.onDrop}
                  multiple={false}
                >
                  <button className="btn secondary blue u-marginBottom--10" disabled={loading}>{loading ? "Uploading" : "Upload license"}</button>
                </Dropzone>
                :
                <button className="btn secondary blue u-marginBottom--10" disabled={loading} onClick={this.syncAppLicense.bind(this, "")}>{loading ? "Syncing" : "Sync license"}</button>
              }
              {message &&
                <p className={classNames("u-fontWeight--bold u-fontSize--small u-position--absolute", {
                  "u-color--red": messageType === "error",
                  "u-color--tuna": messageType === "info",
                })}>{message}</p>
              }
            </div>
          </div>
          :
          <div>
            <p className="u-fontSize--large u-color--dustyGray u-marginTop--15 u-lineHeight--more"> License data is not available on this application because it was installed via Helm </p>
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
              <p className="u-fontSize--large u-color--tuna u-lineHeight--medium u-marginBottom--20">
                The license for {appName} has been updated. A new commit has been made to the gitops repository with these changes. Please head to the <a className="link" target="_blank" href={gitops?.uri} rel="noopener noreferrer">repo</a> to see the diff.
              </p>
              <div className="flex justifyContent--flexEnd">
                <button type="button" className="btn blue primary" onClick={this.hideNextStepModal}>Ok, got it!</button>
              </div>
            </div>
            :
            <div className="Modal-body">
              <p className="u-fontSize--large u-color--tuna u-lineHeight--medium u-marginBottom--20">
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
