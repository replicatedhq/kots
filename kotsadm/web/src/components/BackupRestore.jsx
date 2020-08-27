import * as React from "react";
import { withRouter, Link } from "react-router-dom";
import { Helmet } from "react-helmet";
import isEmpty from "lodash/isEmpty";

import "../scss/components/UploadLicenseFile.scss";
import { Utilities } from "../utilities/utilities";

import RestoreSnapshotRow from "./RestoreSnapshotRow";
import UploadLicenseFile from "./UploadLicenseFile";
import Loader from "./shared/Loader";
import ConfigureSnapshots from "./snapshots/ConfigureSnapshots";

class BackupRestore extends React.Component {
  state = {
    backups: [],
    isLoadingBackups: false,
    backupsErr: false,
    backupsErrMsg: "",
    selectedBackup: {},
    snapshotSettings: null,
    isLoadingSnapshotSettings: true,
    snapshotSettingsErr: false,
    snapshotSettingsErrMsg: "",
    hideCheckVeleroButton: false
  }

  useBackup = (backup) => {
    this.setState({ selectedBackup: backup })
  }

  useDifferentBackup = () => {
    this.setState({ selectedBackup: {} })
  }

  componentDidMount = () => {
    this.fetchSnapshotBackups();
    this.fetchSnapshotSettings();
  }

  fetchSnapshotBackups = () => {
    this.setState({
      isLoadingBackups: true,
      backupsErr: false,
      backupsErrMsg: ""
    });

    fetch(`${window.env.API_ENDPOINT}/snapshots`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    })
      .then(res => res.json())
      .then(result => {
        this.setState({
          backups: result.backups?.sort((a, b) => new Date(b.startedAt) - new Date(a.startedAt)),
          isLoadingBackups: false,
          backupsErr: false,
          backupsErrMsg: "",
        })
      })
      .catch(err => {
        this.setState({
          isLoadingBackups: false,
          backupsErr: true,
          backupsErrMsg: err
        })
      })
  }

  fetchSnapshotSettings = (isCheckForVelero) => {
    this.setState({
      isLoadingSnapshotSettings: true,
      snapshotSettingsErr: false,
      snapshotSettingsErrMsg: "",
      hideCheckVeleroButton: isCheckForVelero ? true : false
    });

    fetch(`${window.env.API_ENDPOINT}/snapshots/settings`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    })
      .then(res => res.json())
      .then(result => {
        this.setState({
          snapshotSettings: result,
          isLoadingSnapshotSettings: false,
          snapshotSettingsErr: false,
          snapshotSettingsErrMsg: "",
        })
        if (result.veleroVersion === "") {
          setTimeout(() => {
            this.setState({ hideCheckVeleroButton: false });
          }, 5000);
        } else {
          this.setState({ hideCheckVeleroButton: false });
        }
      })
      .catch(err => {
        this.setState({
          isLoadingSnapshotSettings: false,
          snapshotSettingsErr: true,
          snapshotSettingsErrMsg: err
        })
      })
  }

  renderSnapshotsListView = () => {
    return (
      <div className="flex flex-column">
        <div className="flex-auto">
          <Link to="/upload-license" className="u-fontSize--normal u-fontWeight--medium u-color--royalBlue u-cursor--pointer">
            <span className="icon clickable backArrow-icon u-marginRight--10" style={{ verticalAlign: "0" }} />
          Back to license upload
        </Link>
          <p className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-marginTop--10">Select a snapshot to restore from</p>
          <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal u-marginTop--5">
            Choose the snapshot backup that you want to restore your application from.
        </p>
          {!isEmpty(this.state.backups) &&
            <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal u-marginTop--5">
              Not seeing the the snapshots you want? <Link to="/snapshots?=license-upload" className="replicated-link">Pull from a different bucket</Link>.
        </p>}
        </div>
        {!isEmpty(this.state.backups) ?
          <div className="flex flex-column">
            {this.state.backups?.map((snapshot, i) => {
              return (
                <RestoreSnapshotRow
                  key={`${snapshot.name}-${i}`}
                  snapshot={snapshot}
                  useBackup={this.useBackup}
                />
              )
            })}
          </div> :
          <div className="EmptyBackup--wrapper flex1 alignItems--center u-marginTop--20">
            <p className="u-fontSize--normal u-color--tundora u-fontWeight--bold"> No backups availible </p>
            <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal u-marginTop--5">
              Not seeing the the snapshots you want? <Link to="/snapshots?=license-upload" className="replicated-link u-fontSize--normal">Check a different bucket</Link>.
        </p>
          </div>
        }
      </div>
    )
  }

  renderSelectedBackupView = (selectedBackup, applicationName, logo) => {
    return (
      <div className="flex flex-column BackupRestoreBox--wrapper">
        <div className="flex-auto">
          <p className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-marginTop--10">Selected backup</p>
          <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal u-marginTop--5"> KOTS Admin Console will be restored from this backup.</p>
        </div>
        <div className="flex flex-column">
          <RestoreSnapshotRow
            key={`${selectedBackup.name}`}
            snapshot={selectedBackup}
            isBackupSelected={true}
            useDifferentBackup={this.useDifferentBackup}
          />
        </div>
        <div className="flex-auto flex-column justifyContent--center u-marginTop--40">
          <div className="flex-auto">
            <p className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-marginTop--10">{`Provide your license file ${applicationName ? `for ${applicationName}` : ""}`}</p>
            <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal u-marginTop--5">{`In order to do a complete restore of your application you must provide the license file ${applicationName ? `for ${applicationName}` : ""}.`}</p>
            <div className="u-marginTop--15">
              <UploadLicenseFile appName={applicationName} logo={logo} isBackupRestore snapshot={selectedBackup} />
            </div>
          </div>
        </div>
      </div>
    )
  }

  renderNotVeleroMessage = () => {
    return <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginTop--12">Not able to find Velero</p>
  }

  navigateToSnapshotConfiguration = () => {
    return (
      <ConfigureSnapshots
        snapshotSettings={this.state.snapshotSettings}
        fetchSnapshotSettings={this.fetchSnapshotSettings}
        renderNotVeleroMessage={this.renderNotVeleroMessage}
        hideCheckVeleroButton={this.state.hideCheckVeleroButton}
        isLicenseUpload={true}
        history={this.props.history}
      />
    )
  }

  render() {
    const { selectedBackup, isLoadingSnapshotSettings, snapshotSettings, isLoadingBackups } = this.state;
    const {
      appName,
      logo,
      appsListLength,
    } = this.props;

    if (isLoadingBackups || isLoadingSnapshotSettings) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    let applicationName;
    if (appsListLength && appsListLength > 1) {
      applicationName = "";
    } else {
      applicationName = appName;
    }



    return (
      <div className="BackupRestore--wrapper container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20 u-marginTop--10 alignItems--center">
        <Helmet>
          <title>{`${applicationName ? `${applicationName} Admin Console` : "Admin Console"}`}</title>
        </Helmet>
        {!snapshotSettings?.isVeleroRunning || !snapshotSettings?.isResticRunning ? this.navigateToSnapshotConfiguration()
          :
          isEmpty(selectedBackup) ?
            this.renderSnapshotsListView()
            : this.renderSelectedBackupView(selectedBackup, applicationName, logo)}
      </div>
    );
  }
}

export default withRouter(BackupRestore);
