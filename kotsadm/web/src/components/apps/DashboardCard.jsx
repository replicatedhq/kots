import React from "react";
import { Link } from "react-router-dom";
import ReactTooltip from "react-tooltip"

import Select from "react-select";
import isEmpty from "lodash/isEmpty";
import moment from "moment";
import dayjs from "dayjs";
import size from "lodash/size";
import url from "url";

import AirgapUploadProgress from "../AirgapUploadProgress";
import Loader from "../shared/Loader";
import MountAware from "../shared/MountAware";
import ShowLogsModal from "@src/components/modals/ShowLogsModal";

import {
  dynamicallyResizeText,
  Utilities,
  getLicenseExpiryDate,
} from "@src/utilities/utilities";

import "../../scss/components/watches/DashboardCard.scss";

export default class DashboardCard extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      selectedAction: "",
      logsLoading: false,
      logs: null,
      selectedTab: null,
    }
    this.cardTitleText = React.createRef();
  }

  resizeCardTitleFont = () => {
    const newFontSize = dynamicallyResizeText(this.cardTitleText.current.innerHTML, this.cardTitleText.current.clientWidth, "20px", 14);
    this.cardTitleText.current.style.fontSize = newFontSize;
  }

  componentDidMount() {
    if (this.props.links && this.props.links.length > 0) {
      this.setState({ selectedAction: this.props.links[0] })
    }
  }

  componentDidUpdate(lastProps) {
    const { cardName } = this.props;
    if (this.props.links !== lastProps.links && this.props.links && this.props.links.length > 0) {
      this.setState({ selectedAction: this.props.links[0] })
    }
    if (cardName && cardName !== lastProps.cardName) {
      if (this.cardTitleText) {
        this.resizeCardTitleFont();
      }
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

  renderVersionAvailable = (downstream) => {
    if (downstream?.pendingVersions?.length > 0) {
      return (
        <div className="flex flex-column u-marginTop--12">
          <p className="u-fontSize--small u-lineHeight--normal u-color--chateauGreen u-fontWeight--bold">New version available</p>
          <div className="flex flex1 alignItems--center u-marginTop--5">
            <span className="u-fontSize--normal u-fontWeight--bold u-color--tundora"> {downstream?.pendingVersions[0].versionLabel} </span>
            <Link to={`${this.props.url}/version-history`} className="card-link u-marginLeft--5"> View </Link>
          </div>
        </div>
      )
    } else {
      return (
        <p className="u-fontSize--small u-lineHeight--normal u-fontWeight--medium u-marginTop--12" style={{ color: "#DFDFDF" }}> No new version available </p>
      )
    }
  }

  hideLogsModal = () => {
    this.setState({
      showLogsModal: false
    });
  }

  renderLogsTabs = () => {
    const { logs, selectedTab } = this.state;
    if (!logs) {
      return null;
    }
    const tabs = Object.keys(logs);

    return (
      <div className="flex action-tab-bar u-marginTop--10">
        {tabs.filter(tab => tab !== "renderError").map(tab => (
          <div className={`tab-item blue ${tab === selectedTab && "is-active"}`} key={tab} onClick={() => this.setState({ selectedTab: tab })}>
            {tab}
          </div>
        ))}
      </div>
    );
  }

  handleViewLogs = async (version, isFailing) => {
    try {
      const { app } = this.props;
      const clusterId = app.downstreams?.length && app.downstreams[0].cluster?.id;

      this.setState({ logsLoading: true, showLogsModal: true, viewLogsErrMsg: "" });

      const res = await fetch(`${window.env.API_ENDPOINT}/app/${app?.slug}/cluster/${clusterId}/sequence/${version?.sequence}/downstreamoutput`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      });
      if (res.ok && res.status === 200) {
        const response = await res.json();
        let selectedTab;
        if (isFailing) {
          selectedTab = Utilities.getDeployErrorTab(response.logs);
        } else {
          selectedTab = Object.keys(response.logs)[0];
        }
        this.setState({ logs: response.logs, selectedTab, logsLoading: false, viewLogsErrMsg: "" });
      } else {
        this.setState({ logsLoading: false, viewLogsErrMsg: `Failed to view logs, unexpected status code, ${res.status}` });
      }
    } catch (err) {
      console.log(err)
      this.setState({ logsLoading: false, viewLogsErrMsg: err ? `Failed to view logs: ${err.message}` : "Something went wrong, please try again." });
    }
  }

  getCurrentVersionStatus = (version) => {
    if (version?.status === "deployed" || version?.status === "merged" || version?.status === "pending") {
      return <span className="u-fontSize--small u-lineHeight--normal u-color--dustyGray u-fontWeight--medium flex alignItems--center"> <span className="icon checkmark-icon u-marginRight--5" /> {Utilities.toTitleCase(version?.status).replace("_", " ")} </span>
    } else if (version?.status === "failed") {
      return <span className="u-fontSize--small u-lineHeight--normal u-color--red u-fontWeight--medium flex alignItems--center"> <span className="icon error-small u-marginRight--5" /> Failed <span className="u-marginLeft--5 replicated-link u-fontSize--small" onClick={() => this.handleViewLogs(version, true)}> See details </span></span>
    } else if (version?.status === "deploying") {
      return (
        <span className="flex alignItems--center u-fontSize--small u-lineHeight--normal u-color--dustyGray u-fontWeight--medium">
          <Loader className="flex alignItems--center u-marginRight--5" size="16" />
            Deploying
        </span>);
    } else {
      return <span className="u-fontSize--small u-lineHeight--normal u-color--dustyGray u-fontWeight--medium flex alignItems--center"> {Utilities.toTitleCase(version?.status).replace("_", " ")} </span>
    }
  }

  renderVersionHistoryCard = () => {
    const { app, currentVersion, downstream, checkingForUpdates, checkingForUpdateError, checkingUpdateText, errorCheckingUpdate, onCheckForUpdates, isBundleUploading } = this.props;

    let checkingUpdateTextShort = checkingUpdateText;
    if (checkingUpdateTextShort && checkingUpdateTextShort.length > 30) {
      checkingUpdateTextShort = checkingUpdateTextShort.slice(0, 30) + "...";
    }

    const showOnlineUI = !app.isAirgap && !checkingForUpdates;
    const showAirgapUI = app.isAirgap && !isBundleUploading;


    let updateText;
    if (showOnlineUI && app.lastUpdateCheckAt) {
      updateText = <p className="u-marginTop--8 u-fontSize--smaller u-color--silverSand u-marginTop--8">Last checked <span className="u-fontWeight--bold">{dayjs(app.lastUpdateCheckAt).fromNow()}</span></p>;
    } else if (this.props.airgapUploadError) {
      updateText = <p className="u-marginTop--10 u-fontSize--small u-color--chestnut u-fontWeight--medium">Error uploading bundle <span className="u-color--royalBlue u-textDecoration--underlineOnHover" onClick={this.props.viewAirgapUploadError}>See details</span></p>
    } else if (this.props.uploadingAirgapFile) {
      updateText = (
        <AirgapUploadProgress
          total={this.props.uploadSize}
          progress={this.props.uploadProgress}
          resuming={this.props.uploadResuming}
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
    } else if (!app.lastUpdateCheckAt) {
      updateText = null;
    }


    return (
      <div className="flex1 flex-column">
        {currentVersion?.deployedAt ?
          <div className="flex flex-column" style={{ minHeight: "35px" }}>
            {this.getCurrentVersionStatus(currentVersion)}
            <p className="u-fontSize--small u-fontWeight--medium u-color--dustyGray u-marginTop--5"> {moment(currentVersion?.deployedAt).format("lll")} </p>
          </div>
          :
          <p className="u-fontWeight--bold u-fontSize--normal u-color--dustyGray" style={{ minHeight: "35px" }}> No version deployed </p>}
        {checkingForUpdates && !isBundleUploading
          ? <Loader className="flex justifyContent--center u-marginTop--10" size="32" />
          : showAirgapUI
            ?
            <MountAware className="u-marginTop--30" onMount={el => this.props.airgapUploader?.assignElement(el)}>
              <button className="btn primary blue">Upload a new version</button>
            </MountAware>
            : showOnlineUI ?
              <div className="flex1 flex-column" style={{ flexGrow: 1 }}>
                {this.renderVersionAvailable(downstream)}
                <div className="flex alignItems--center">
                  <button className="btn primary blue u-marginTop--10" onClick={onCheckForUpdates}>Check for update</button>
                  <span className="icon settings-small-icon u-marginLeft--10 u-cursor--pointer u-marginTop--10" onClick={this.props.showUpdateCheckerModal} data-tip="Configure automatic update checks"></span>
                  <ReactTooltip effect="solid" className="replicated-tooltip" />
                </div>
                {updateText}
              </div>
              : null
        }
        {!showOnlineUI && updateText}
        {checkingForUpdateError &&
          <div className="flex-column flex-auto u-marginTop--5">
            <p className="u-marginTop--10 u-fontSize--small u-color--chestnut u-fontWeight--medium">Error updating version <span className="u-color--royalBlue u-textDecoration--underlineOnHover" onClick={() => this.props.viewAirgapUpdateError(checkingUpdateText)}>View details</span></p>
          </div>}
      </div>
    )
  }

  renderLicenseCard = () => {
    const { appLicense, isSnapshotAllowed, getingAppLicenseErrMsg } = this.props;
    const expiresAt = getLicenseExpiryDate(appLicense);

    return (
      <div>
        {isSnapshotAllowed ?
          getingAppLicenseErrMsg && <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal flex">{getingAppLicenseErrMsg}</p>
          :
          size(appLicense) > 0 ?
            <div>
              {appLicense?.licenseType === "community" && <p className="u-fontSize--normal u-fontWeight--medium u-color--selectiveYellow u-marginBottom--15"> Community Edition </p>}
              <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray"> Channel: <span className="u-fontWeight--bold u-fontSize--normal u-color--tundora"> {appLicense?.channelName} </span></p>
              <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-marginTop--15"> Expires: <span className="u-fontWeight--bold u-fontSize--normal u-color--tundora"> {expiresAt} </span></p>
              <p className="u-fontSize--small u-color--dustyGray u-marginTop--15 u-lineHeight--medium"> Contact your account rep to change your License. </p>
            </div>
            :
            getingAppLicenseErrMsg ?
              <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal flex">{getingAppLicenseErrMsg}</p>
              :
              <p className="u-fontSize--normal u-color--dustyGray u-marginTop--15 u-lineHeight--more"> License data is not available on this application because it was installed via Helm </p>
        }
      </div>
    )
  }

  render() {
    const { cardName, cardIcon, application, versionHistory, url, app, appLicense, license, isSnapshotAllowed, startManualSnapshot, startSnapshotErr, startSnapshotErrorMsg, snapshotInProgressApps, getingAppLicenseErrMsg } = this.props;
    const isSnapshotInProgress = !!snapshotInProgressApps?.find(a => a === app?.slug);

    return (
      <div className={`${isSnapshotAllowed ? "small-dashboard-card" : appLicense?.licenseType === "community" ? "community-dashboard-card" : appLicense && size(appLicense) === 0 ? "grayed-dashboard-card" : "dashboard-card"} flex flex1`}>
        <div className="flex flex1 u-marginBottom--5">
          <span className={`icon ${cardIcon} u-marginRight--10`}></span>
          <div className="flex1 flex-column">
            <div className={`flex justifyContent--spaceBetween ${appLicense && size(appLicense) === 0 ? "u-marginTop--10" : ""}`}>
              <p ref={this.cardTitleText} style={{ fontSize: "20px" }} className={`flex1 u-fontWeight--bold u-fontSize--largest u-paddingRight--5 u-marginBottom--5 ${appLicense && size(appLicense) === 0 ? "u-color--doveGray" : "u-color--tundora"}`}>{cardName}</p>
            </div>
            {application ?
              app.isConfigurable && <Link to={`${url}/config`} className="card-link"> Configure </Link>
              :
              versionHistory ?
                <Link to={`${url}/version-history`} className="card-link"> Version history </Link>
                :
                size(appLicense) > 0 ?
                  <Link to={`${url}/license`} className="card-link"> View license details </Link>
                  : isSnapshotAllowed ?
                    isSnapshotInProgress ?
                      <Loader size="16" />
                      :
                      !getingAppLicenseErrMsg && <span className="status-indicator completed"> Enabled </span>
                    : null
            }
            <div className={`${isSnapshotAllowed || versionHistory ? "flex-auto flex-column u-marginTop--8" : "u-marginTop--15"}`}>
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
        {this.state.showLogsModal &&
          <ShowLogsModal
            showLogsModal={this.state.showLogsModal}
            hideLogsModal={this.hideLogsModal}
            viewLogsErrMsg={this.state.viewLogsErrMsg}
            logs={this.state.logs}
            selectedTab={this.state.selectedTab}
            logsLoading={this.state.logsLoading}
            renderLogsTabs={this.renderLogsTabs()}
          />}
      </div>
    );
  }
}
