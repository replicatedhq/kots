import React from "react";
import { Link } from "react-router-dom";

import moment from "moment";
import dayjs from "dayjs";

import Loader from "../shared/Loader";

import {
  Utilities,
  getLicenseExpiryDate,
} from "@src/utilities/utilities";

import "../../scss/components/watches/DashboardCard.scss";

export default class DashboardCard extends React.Component {

  renderApplicationCard = () => {
    const { appStatus, url } = this.props;

    return (
      <div>
        <p className="u-fontWeight--bold u-fontSize--normal u-color--tundora"> Status </p>
        <div className="flex alignItems--center u-marginTop--5">
          <span className={`icon ${appStatus === "ready" ? "checkmark-icon" : appStatus === "degraded" ? "spinnerOrange" : "spinnerRed"}`}></span>
          <span className={`u-marginLeft--5 u-fontSize--normal u-fontWeight--medium ${appStatus === "ready" ? "u-color--dustyGray" : appStatus === "degraded" ? "u-color--orange" : "u-color--chestnut"}`}>
            {Utilities.toTitleCase(appStatus)}
          </span>
          {appStatus !== "ready" ?
            <Link to={`${url}/troubleshoot`} className="card-link u-marginLeft--10"> Troubleshoot </Link>
            : null}
        </div>
      </div>
    )
  }

  renderVersionHistoryCard = () => {
    const { app, currentVersion, downstreams, isAirgap, checkingForUpdates, checkingUpdateText, errorCheckingUpdate, onCheckForUpdates, onUploadNewVersion, deployVersion } = this.props;
    const updatesText = downstreams.pendingVersions ?.length > 0 ? "Updates are ready to be installed." : "No updates available.";
    const isUpdateAvailable = downstreams.pendingVersions ?.length > 0;

    let updateText = <p className="u-marginTop--10 u-fontSize--small u-color--dustyGray u-fontWeight--medium">Last checked {dayjs(app.lastUpdateCheck).fromNow()}</p>;
    if (errorCheckingUpdate) {
      updateText = <p className="u-marginTop--10 u-fontSize--small u-color--chestnut u-fontWeight--medium">Error checking for updates, please try again</p>
    } else if (checkingForUpdates) {
      updateText = <p className="u-marginTop--10 u-fontSize--small u-color--dustyGray u-fontWeight--medium">{checkingUpdateText}</p>
    } else if (!app.lastUpdateCheck) {
      updateText = null;
    }

    return (
      <div>
        <p className="u-fontWeight--bold u-fontSize--normal u-color--tundora"> {currentVersion?.status === "deployed" ? "Installed" : ""} </p>
        <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginTop--5"> {moment(currentVersion?.createdOn).format("lll")} </p>

        <p className="u-fontSize--small u-color--dustyGray u-marginTop--15"> {updatesText} </p>
        {checkingForUpdates
          ? <Loader size="32" className="flex justifyContent--center u-marginTop--10" />
          : <button className="btn primary lightBlue u-marginTop--10" onClick={isAirgap ? onUploadNewVersion : isUpdateAvailable ? deployVersion : onCheckForUpdates}> {isAirgap ? "Upload new version" : isUpdateAvailable ? "Install update" : "Check for update"} </button>
        }
        {updateText}
      </div>
    )
  }

  renderLicenseCard = () => {
    const { appLicense } = this.props;
    const expiresAt = getLicenseExpiryDate(appLicense);


    return (
      <div>
        <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray"> Channel: <span className="u-fontWeight--bold u-fontSize--normal u-color--tundora"> {appLicense ?.channelName} </span></p>
        <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-marginTop--15"> Expires: <span className="u-fontWeight--bold u-fontSize--normal u-color--tundora"> {expiresAt} </span></p>
        <p className="u-fontSize--small u-color--dustyGray u-marginTop--15 u-lineHeight--medium"> <a href="" target="_blank" rel="noopener noreferrer" className="card-link" > Contact your account rep </a> to update your License. </p>
      </div>
    )
  }

  render() {
    const { cardName, cardIcon, application, versionHistory, url } = this.props;


    return (
      <div className="dashboard-card flex-column flex1 flex">
        <div className="flex u-marginBottom--5">
          <span className={`icon ${cardIcon} u-marginRight--10`}></span>
          <div className="flex1 justifyContent--center">
            <div className="flex justifyContent--spaceBetween">
              <p className="flex1 u-fontWeight--bold u-fontSize--largest u-color--tundora u-paddingRight--5 u-marginBottom--5">{cardName}</p>
            </div>
            {application ?
              <Link to={""} className="card-link"> Configure </Link>
              :
              versionHistory ?
                <Link to={`${url}/version-history`} className="card-link"> Version history </Link>
                :
                <Link to={`${url}/license`} className="card-link"> View license details </Link>
            }
            <div className="u-marginTop--15">
              <div className="flex flex1">
                {application ?
                  this.renderApplicationCard()
                  : versionHistory ?
                    this.renderVersionHistoryCard()
                    : this.renderLicenseCard()
                }
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }
}
