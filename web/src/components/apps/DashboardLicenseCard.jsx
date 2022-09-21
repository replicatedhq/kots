import React from "react";
import size from "lodash/size";
import yaml from "js-yaml";
import classNames from "classnames";
import Loader from "../shared/Loader";
import Dropzone from "react-dropzone";
import Modal from "react-modal";
import {
  getFileContent,
  getLicenseExpiryDate,
  Utilities,
} from "@src/utilities/utilities";
import "../../scss/components/watches/DashboardCard.scss";
import "@src/scss/components/apps/AppLicense.scss";
import { Link } from "react-router-dom";
import LicenseFields from "./LicenseFields";

export default class DashboardLicenseCard extends React.Component {
  state = {
    syncingLicense: false,
    message: null,
    messageType: "",
    entitlementsToShow: [],
    isViewingLicenseEntitlements: false,
  };

  syncLicense = (licenseData) => {
    this.setState({
      syncingLicense: true,
      message: null,
      messageType: "info",
    });

    const { app } = this.props;

    const payload = {
      licenseData,
    };

    fetch(`${process.env.API_ENDPOINT}/app/${app?.slug}/license`, {
      method: "PUT",
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload),
    })
      .then(async (response) => {
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
          message = "License is already up to date";
        } else if (app?.isAirgap) {
          message = "License uploaded successfully";
        } else {
          message = "License synced successfully";
        }

        this.setState({
          appLicense: licenseResponse.license,
          isViewingLicenseEntitlements:
            size(licenseResponse.license?.entitlements) <= 5 ? true : false,
          message,
          messageType: "info",
          showNextStepModal: licenseResponse.synced,
        });

        if (this.props.syncCallback) {
          this.props.syncCallback();
        }
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          message: err ? err.message : "Something went wrong",
          messageType: "error",
        });
      })
      .finally(() => {
        this.setState({ syncingLicense: false });
        setTimeout(() => {
          this.setState({
            message: null,
            messageType: "",
          });
        }, 3000);
      });
  };

  onDrop = async (files) => {
    const content = await getFileContent(files[0]);
    const contentStr = new TextDecoder("utf-8").decode(content);
    const airgapLicense = await yaml.safeLoad(contentStr);
    const { appLicense } = this.state;

    if (airgapLicense.spec?.licenseID !== appLicense?.id) {
      this.setState({
        message: "Licenses do not match",
        messageType: "error",
      });
      return;
    }

    if (airgapLicense.spec?.licenseSequence === appLicense?.licenseSequence) {
      this.setState({
        message: "License is already up to date",
        messageType: "info",
      });
      return;
    }

    this.syncLicense(contentStr);
  };

  hideNextStepModal = () => {
    this.setState({ showNextStepModal: false });
  };

  toggleShowDetails = (entitlement) => {
    this.setState({
      entitlementsToShow: [...this.state.entitlementsToShow, entitlement],
    });
  };

  toggleHideDetails = (entitlement) => {
    let entitlementsToShow = [...this.state.entitlementsToShow];
    const index = this.state.entitlementsToShow?.indexOf(entitlement);
    entitlementsToShow.splice(index, 1);
    this.setState({ entitlementsToShow });
  };

  viewLicenseEntitlements = () => {
    this.setState({
      isViewingLicenseEntitlements: !this.state.isViewingLicenseEntitlements,
    });
  };

  render() {
    const { app, appLicense, getingAppLicenseErrMsg } = this.props;
    const { syncingLicense, showNextStepModal, message, messageType } =
      this.state;
    const expiresAt = getLicenseExpiryDate(appLicense);
    const isCommunityLicense = appLicense?.licenseType === "community";
    const gitops = app.downstream?.gitops;
    const appName = app?.name || "Your application";
    const expiredLicenseClassName = Utilities.checkIsDateExpired(expiresAt)
      ? "expired-license"
      : "";
    const appLicenseClassName =
      appLicense && size(appLicense) === 0 ? "no-license" : "dashboard-card";

    return (
      <div
        className={`${
          isCommunityLicense ? "CommunityLicense--wrapper" : appLicenseClassName
        } ${expiredLicenseClassName} flex-column`}
      >
        <div className="flex flex1 justifyContent--spaceBetween alignItems--center">
          <p
            className={`u-fontSize--large u-textColor--${
              Utilities.checkIsDateExpired(expiresAt) ? "error" : "primary"
            } u-fontWeight--bold`}
          >
            License {Utilities.checkIsDateExpired(expiresAt) && "is expired"}
            {isCommunityLicense && (
              <span className="CommunityEditionTag u-marginLeft--5">
                Community Edition
              </span>
            )}
          </p>
          {app?.isAirgap ? (
            <Dropzone
              className="Dropzone-wrapper flex alignItems--center"
              accept={["application/x-yaml", ".yaml", ".yml"]}
              onDropAccepted={this.onDrop}
              multiple={false}
            >
              <span className="icon clickable dashboard-card-upload-version-icon u-marginRight--5" />
              <span
                className="replicated-link u-fontSize--small"
                onClick={() => this.syncLicense("")}
              >
                Upload license
              </span>
            </Dropzone>
          ) : syncingLicense ? (
            <div className="flex alignItems--center">
              <Loader className="u-marginRight--5" size="15" />
              <span className="u-textColor--bodyCopy u-fontWeight--medium u-fontSize--small u-lineHeight--default">
                Syncing license
              </span>
            </div>
          ) : (
            <div className="flex alignItems--center">
              {message && (
                <p
                  className={classNames(
                    "u-fontWeight--bold u-fontSize--small u-marginRight--10",
                    {
                      "u-textColor--error": messageType === "error",
                      "u-textColor--primary": messageType === "info",
                    }
                  )}
                >
                  {message}
                </p>
              )}
              {appLicense?.lastSyncedAt && !message ? (
                <span className="u-fontSize--small u-textColor--header u-fontWeight--medium u-lineHeight--normal u-marginRight--10">
                  Last synced {Utilities.dateFromNow(appLicense.lastSyncedAt)}
                </span>
              ) : null}
              <span className="icon clickable dashboard-card-sync-icon u-marginRight--5" />
              <span
                className="replicated-link u-fontSize--small"
                onClick={() => this.syncLicense("")}
              >
                Sync license
              </span>
            </div>
          )}
        </div>
        <div className="LicenseCard-content--wrapper u-marginTop--10">
          {size(appLicense) > 0 ? (
            <div className="flex">
              <div className="flex-column flex1">
                <div className="flex alignItems--center">
                  <p className="u-fontSize--large u-fontWeight--medium u-textColor--header">
                    {" "}
                    {appLicense?.assignee}
                  </p>
                  {appLicense?.channelName && (
                    <span className="channelTag flex-auto alignItems--center u-fontWeight--medium u-marginLeft--10">
                      {" "}
                      {appLicense.channelName}{" "}
                    </span>
                  )}
                </div>
                <div className="flex flex1 alignItems--center u-marginTop--15">
                  <div
                    className={`LicenseTypeTag ${appLicense?.licenseType} flex-auto justifyContent--center alignItems--center`}
                  >
                    <span
                      className={`icon ${
                        appLicense?.licenseType === "---"
                          ? ""
                          : appLicense?.licenseType
                      }-icon`}
                    ></span>
                    {appLicense?.licenseType !== "---"
                      ? `${Utilities.toTitleCase(
                          appLicense.licenseType
                        )} license`
                      : `---`}
                  </div>
                  <p
                    className={`u-fontWeight--medium u-fontSize--small u-lineHeight--default u-marginLeft--10 ${
                      Utilities.checkIsDateExpired(expiresAt)
                        ? "u-textColor--error"
                        : "u-textColor--bodyCopy"
                    }`}
                  >
                    {expiresAt === "Never"
                      ? "Does not expire"
                      : Utilities.checkIsDateExpired(expiresAt)
                      ? `Expired ${expiresAt}`
                      : `Expires ${expiresAt}`}
                  </p>
                </div>
                {size(appLicense?.entitlements) >= 5 && (
                  <span
                    className="flexWrap--wrap flex u-fontSize--small u-lineHeight--normal u-color--doveGray u-fontWeight--medium u-marginRight--normal alignItems--center"
                    style={{ margin: "10px 0" }}
                  >
                    <span
                      className={`u-fontWeight--bold u-cursor--pointer`}
                      style={{ whiteSpace: "pre" }}
                      onClick={(e) => {
                        e.stopPropagation();
                        this.viewLicenseEntitlements();
                      }}
                    >
                      View {size(appLicense?.entitlements)} license entitlements
                      <span
                        className={`icon clickable ${
                          this.state.isViewingLicenseEntitlements
                            ? "up-arrow-icon"
                            : "down-arrow-icon"
                        } u-marginLeft--5`}
                      />
                    </span>
                  </span>
                )}
                {this.state.isViewingLicenseEntitlements ? (
                  <LicenseFields
                    entitlements={appLicense?.entitlements}
                    entitlementsToShow={this.state.entitlementsToShow}
                    toggleHideDetails={this.toggleHideDetails}
                    toggleShowDetails={this.toggleShowDetails}
                  />
                ) : (
                  appLicense.entitlements.length < 5 && (
                    <div style={{ marginTop: "15px" }}>
                      <LicenseFields
                        entitlements={appLicense?.entitlements}
                        entitlementsToShow={this.state.entitlementsToShow}
                        toggleHideDetails={this.toggleHideDetails}
                        toggleShowDetails={this.toggleShowDetails}
                      />
                    </div>
                  )
                )}
              </div>
            </div>
          ) : (
            <p className="u-textColor--error u-fontSize--small u-fontWeight--medium u-lineHeight--normal flex">
              {getingAppLicenseErrMsg}
            </p>
          )}
        </div>
        <div className="u-marginTop--10">
          <Link
            to={`/app/${app?.slug}/license`}
            className="replicated-link has-arrow u-fontSize--small"
          >
            See license details
          </Link>
        </div>
        <Modal
          isOpen={showNextStepModal}
          onRequestClose={this.hideNextStepModal}
          shouldReturnFocusAfterClose={false}
          contentLabel="Next step"
          ariaHideApp={false}
          className="Modal SmallSize"
        >
          {gitops?.isConnected ? (
            <div className="Modal-body">
              <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">
                License synced
              </p>
              <p className="u-fontSize--large u-textColor--bodyCopy u-lineHeight--medium u-marginBottom--20">
                The license for {appName} has been updated. A new commit has
                been made to the gitops repository with these changes. Please
                head to the{" "}
                <a
                  className="link"
                  target="_blank"
                  href={gitops?.uri}
                  rel="noopener noreferrer"
                >
                  repo
                </a>{" "}
                to see the diff.
              </p>
              <div>
                <button
                  type="button"
                  className="btn blue primary"
                  onClick={this.hideNextStepModal}
                >
                  Ok, got it!
                </button>
              </div>
            </div>
          ) : (
            <div className="Modal-body">
              <p className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-lineHeight--normal u-marginBottom--10">
                License synced
              </p>
              <p className="u-fontSize--large u-textColor--bodyCopy u-lineHeight--medium u-marginBottom--20">
                The license for {appName} has been updated. A new version is
                available for deploy with these changes from the Version card on
                the dashboard. To see a full list of versions visit the{" "}
                <Link to={`/app/${app?.slug}/version-history`}>
                  version history
                </Link>{" "}
                page.
              </p>
              <div>
                <button
                  type="button"
                  className="btn blue primary"
                  onClick={this.hideNextStepModal}
                >
                  Ok, got it!
                </button>
              </div>
            </div>
          )}
        </Modal>
      </div>
    );
  }
}
