import React from "react";
import { Link } from "react-router-dom";
import Dropzone from "react-dropzone";

import Select from "react-select";
import isEmpty from "lodash/isEmpty";
import moment from "moment";
import dayjs from "dayjs";
import size from "lodash/size";
import url from "url";

import AirgapUploadProgress from "../AirgapUploadProgress";
import Loader from "../shared/Loader";

import {
  Utilities,
  getLicenseExpiryDate,
} from "@src/utilities/utilities";

import "../../scss/components/watches/DashboardCard.scss";
import { isVeleroInstalled } from "../../queries/SnapshotQueries";

export default class DashboardCard extends React.Component {
  state = {
    selectedAction: ""
  }

  componentDidMount() {
    if (this.props.links && this.props.links.length > 0) {
      this.setState({ selectedAction: this.props.links[0] })
    }
  }

  componentDidUpdate(lastProps) {
    if (this.props.links !== lastProps.links && this.props.links && this.props.links.length > 0) {
      this.setState({ selectedAction: this.props.links[0] })
    }
  }

  onActionChange = (selectedOption) => {
    if (selectedOption.uri) {
      window.open(this.createDashboardActionLink(selectedOption.uri), "_blank");
    }
    this.setState({ selectedAction: selectedOption });
  }

  createDashboardActionLink = (uri) => {
    const parsedUrl = url.parse(uri);

    let port;
    if (!parsedUrl.port) {
      port = "";
    } else {
      port = ":" + parsedUrl.port;
    }

    return `${parsedUrl.protocol}//${window.location.hostname}${port}${parsedUrl.path}`;
  }

  renderApplicationCard = () => {
    const { selectedAction } = this.state;
    const { appStatus, url, links } = this.props;

    return (
      <div className="flex-column flex1">
        <p className="u-fontWeight--bold u-fontSize--normal u-color--tundora"> Status </p>
        {!isEmpty(appStatus)
          ?
          <div className="flex alignItems--center u-marginTop--5">
            {appStatus === "ready" ?
              <span className={`icon ${appStatus === "ready" ? "checkmark-icon" : ""}`}></span>
              :
              appStatus === "degraded" ?
                <Loader size="16" color="#DB9016" />
                :
                <Loader size="16" color="#BC4752" />
            }
            <span className={`u-marginLeft--5 u-fontSize--normal u-fontWeight--medium ${appStatus === "ready" ? "u-color--dustyGray" : appStatus === "degraded" ? "u-color--orange" : "u-color--chestnut"}`}>
              {Utilities.toTitleCase(appStatus)}
            </span>
            {appStatus !== "ready" ?
              <Link to={`${url}/troubleshoot`} className="card-link u-marginLeft--10"> Troubleshoot </Link>
              : null}
          </div>
          :
          <div className="flex alignItems--center u-marginTop--5">
            <span className="icon grayQuestionMark--icon"></span>
            <span className="u-marginLeft--5 u-fontSize--normal u-fontWeight--medium u-color--dustyGray">
              Unknown
            </span>
          </div>
        }
        {links?.length > 0 ?
          <div>
            {links?.length > 1 ?
              <div className="u-marginTop--15">
                <Select
                  className="replicated-select-container"
                  classNamePrefix="replicated-select"
                  options={links}
                  getOptionLabel={(link) => link.title}
                  getOptionValue={(option) => option.title}
                  value={selectedAction}
                  onChange={this.onActionChange}
                  isOptionSelected={(option) => { option.title === selectedAction.title }}
                />
              </div>
              :
              selectedAction?.uri ?
                <div className="u-marginTop--15">
                  <a href={this.createDashboardActionLink(selectedAction.uri)} target="_blank" rel="noopener noreferrer" className="btn secondary"> {selectedAction.title} </a>
                </div> : null
            }
          </div>
          : null
        }
      </div>
    )
  }

  renderVersionHistoryCard = () => {
    const { app, currentVersion, downstreams, checkingForUpdates, checkingForUpdateError, checkingUpdateText, errorCheckingUpdate, onCheckForUpdates, redirectToDiff, isBundleUploading } = this.props;
    const updatesText = downstreams?.pendingVersions?.length > 0 || app.isAirgap ? null : "No updates available.";
    const isUpdateAvailable = downstreams?.pendingVersions?.length > 0;

    let checkingUpdateTextShort = checkingUpdateText;
    if (checkingUpdateTextShort && checkingUpdateTextShort.length > 30) {
      checkingUpdateTextShort = checkingUpdateTextShort.slice(0, 30) + "...";
    }

    let updateText = <p className="u-marginTop--10 u-fontSize--small u-color--dustyGray u-fontWeight--medium">Last checked {dayjs(app.lastUpdateCheck).fromNow()}</p>;
    if (this.props.airgapUploadError) {
      updateText = <p className="u-marginTop--10 u-fontSize--small u-color--chestnut u-fontWeight--medium">Error uploading bundle <span className="u-color--royalBlue u-textDecoration--underlineOnHover" onClick={this.props.viewAirgapUploadError}>View details</span></p>
    } else if (this.props.uploadingAirgapFile) {
      updateText = (
        <AirgapUploadProgress
          total={this.props.uploadTotal}
          sent={this.props.uploadSent}
          onProgressError={this.props.onProgressError}
          smallSize={true}
        />
      );
    } else if (isBundleUploading) {
      updateText = (
        <AirgapUploadProgress
          unkownProgress={true}
          onProgressError={this.onProgressError}
          smallSize={true}
        />);
    } else if (errorCheckingUpdate) {
      updateText = <p className="u-marginTop--10 u-fontSize--small u-color--chestnut u-fontWeight--medium">Error checking for updates, please try again</p>
    } else if (checkingForUpdates) {
      updateText = <p className="u-marginTop--10 u-fontSize--small u-color--dustyGray u-fontWeight--medium">{checkingUpdateTextShort}</p>
    } else if (!app.lastUpdateCheck) {
      updateText = null;
    }

    const showAirgapUI = app.isAirgap && !isBundleUploading;
    const showOnlineUI = !app.isAirgap && !checkingForUpdates;


    return (
      <div>
        <p className="u-fontWeight--bold u-fontSize--normal u-color--tundora"> {currentVersion?.status === "deployed" ? "Installed" : ""} </p>
        <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginTop--5"> {moment(currentVersion?.createdOn).format("lll")} </p>

        <p className="u-fontSize--small u-color--dustyGray u-marginTop--15"> {updatesText} </p>
        {checkingForUpdates && !isBundleUploading
          ? <Loader className="flex justifyContent--center u-marginTop--10" size="32" />
          : showAirgapUI
            ?
            <Dropzone
              className="Dropzone-wrapper"
              accept=".airgap"
              onDropAccepted={this.props.onDropBundle}
              multiple={false}
            >
              <button className="btn secondary blue">Upload new version</button>
            </Dropzone>
            : showOnlineUI ?
              <button className="btn primary lightBlie blue u-marginTop--10" onClick={isUpdateAvailable ? redirectToDiff : onCheckForUpdates}>{isUpdateAvailable ? "Show Update" : "Check for update"}</button>
              : null
        }
        {updateText}
        {checkingForUpdateError &&
          <div className="flex-column flex-auto u-marginTop--5">
            <p className="u-marginTop--10 u-fontSize--small u-color--chestnut u-fontWeight--medium">Error updating version <span className="u-color--royalBlue u-textDecoration--underlineOnHover" onClick={() => this.props.viewAirgapUpdateError(checkingUpdateText)}>View details</span></p>
          </div>}
      </div>
    )
  }

  renderLicenseCard = () => {
    const { appLicense, isSnapshotAllowed } = this.props;
    const expiresAt = getLicenseExpiryDate(appLicense);

    return (
      <div>
        {isSnapshotAllowed ?
          null
          :
          size(appLicense) > 0 ?
            <div>
              {appLicense?.licenseType === "community" && <p className="u-fontSize--normal u-fontWeight--medium u-color--selectiveYellow u-marginBottom--15"> Community Edition </p>}
              <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray"> Channel: <span className="u-fontWeight--bold u-fontSize--normal u-color--tundora"> {appLicense?.channelName} </span></p>
              <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-marginTop--15"> Expires: <span className="u-fontWeight--bold u-fontSize--normal u-color--tundora"> {expiresAt} </span></p>
              <p className="u-fontSize--small u-color--dustyGray u-marginTop--15 u-lineHeight--medium"> Contact your account rep to update your License. </p>
            </div>
            :
            <div>
              <p className="u-fontSize--normal u-color--dustyGray u-marginTop--15 u-lineHeight--more"> License data is not available on this application because it was installed via Helm </p>
            </div>
        }
      </div>
    )
  }

  render() {
    const { cardName, cardIcon, application, versionHistory, url, app, appLicense, license, isSnapshotAllowed, startManualSnapshot, startSnapshotErr, startSnapshotErrorMsg } = this.props;

    return (
      <div className={`${isSnapshotAllowed ? "small-dashboard-card" : appLicense?.licenseType === "community" ? "community-dashboard-card" : appLicense && size(appLicense) === 0 ? "grayed-dashboard-card" : "dashboard-card"} flex-column flex1 flex`}>
        <div className="flex u-marginBottom--5">
          <span className={`icon ${cardIcon} u-marginRight--10`}></span>
          <div className="flex1 justifyContent--center">
            <div className={`flex justifyContent--spaceBetween ${appLicense && size(appLicense) === 0 ? "u-marginTop--10" : ""}`}>
              <p className={`flex1 u-fontWeight--bold u-fontSize--largest u-paddingRight--5 u-marginBottom--5 ${appLicense && size(appLicense) === 0 ? "u-color--doveGray" : "u-color--tundora"}`}>{cardName}</p>
            </div>
            {application ?
              app.isConfigurable ?
                <Link to={`${url}/config`} className="card-link"> Configure </Link>
                : null
              :
              versionHistory ?
                <Link to={`${url}/version-history`} className="card-link"> Version history </Link>
                :
                size(appLicense) > 0 ?
                  <Link to={`${url}/license`} className="card-link"> View license details </Link>
                  : isSnapshotAllowed ?
                    <span className="status-indicator completed"> Enabled </span>
                    : null
            }
            <div className={`${isSnapshotAllowed ? "u-marginTop--8" : "u-marginTop--15"}`}>
              <div className="flex flex1">
                {application ?
                  this.renderApplicationCard()
                  : versionHistory ?
                    this.renderVersionHistoryCard()
                    : license ?
                      this.renderLicenseCard()
                      : isSnapshotAllowed ?
                        <div className="flex flex-column">
                          {startSnapshotErr &&
                            <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal flex">{startSnapshotErrorMsg}</p>}
                          <span className="card-link flex" onClick={startManualSnapshot}> Start a snapshot </span>
                        </div>
                        : null
                }
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }
}
