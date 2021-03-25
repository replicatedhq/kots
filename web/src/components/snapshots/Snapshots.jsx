import React, { Component } from "react";
import { Link, withRouter } from "react-router-dom";
import Helmet from "react-helmet";
import ReactTooltip from "react-tooltip";

import Loader from "../shared/Loader";
import SnapshotRow from "./SnapshotRow";
import BackupRestoreModal from "../modals/BackupRestoreModal";
import DeleteSnapshotModal from "../modals/DeleteSnapshotModal";
import DummySnapshotRow from "./DummySnapshotRow";
import GettingStartedSnapshots from "./GettingStartedSnapshots";
import ErrorModal from "../modals/ErrorModal";
import SnapshotDifferencesModal from "../modals/SnapshotDifferencesModal";

import "../../scss/components/snapshots/AppSnapshots.scss";
import { Utilities } from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";
import dayjs from "dayjs";


class Snapshots extends Component {
  state = {
    startingSnapshot: false,
    startSnapshotErr: false,
    startSnapshotErrorMsg: "",
    deleteSnapshotModal: false,
    snapshotToDelete: "",
    deleteErr: false,
    deleteErrorMsg: "",
    restoreSnapshotModal: false,
    snapshotToRestore: "",

    snapshotSettings: null,
    isLoadingSnapshotSettings: true,

    errorMsg: "",
    errorTitle: "",

    snapshots: [],
    hasSnapshotsLoaded: false,
    isStartButtonClicked: false,
    listSnapshotsJob: new Repeater(),
    networkErr: false,
    displayErrorModal: false,

    selectedRestore: "full",
    selectedRestoreApp: {},
    appSlugToRestore: "",
    appSlugMismatch: false,
    restoringSnapshot: false,
    snapshotDifferencesModal: false
  };

  componentDidMount() {
    this.fetchSnapshotSettings();
  }

  componentWillUnmount() {
    this.state.listSnapshotsJob.stop();
  }

  componentDidUpdate(lastProps, lastState) {
    const { snapshots, networkErr } = this.state;

    if (snapshots?.length !== lastState.snapshots?.length && snapshots) {
      if (snapshots?.length === 0 && lastState.snapshots?.length > 0) {
        this.setState({ isStartButtonClicked: false });
      }
    }

    if (networkErr !== lastState.networkErr) {
      if (networkErr) {
        this.state.listSnapshotsJob.stop();
      } else {
        this.state.listSnapshotsJob.start(this.listSnapshots, 2000);
        return;
      }
    }
  }

  listInstanceSnapshots = async () => {
    this.setState({
      errorMsg: "",
      errorTitle: "",
      networkErr: false,
      displayErrorModal: false
    })
    try {
      const res = await fetch(`${window.env.API_ENDPOINT}/snapshots`, {
        method: "GET",
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        }
      })
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }
        const err = await res.json();
        this.setState({
          errorTitle: "Failed to get snapshots",
          errorMsg: err ? err.error : `Unexpected status code: ${res.status}`,
          networkErr: true,
          displayErrorModal: true
        });
        return;
      }
      const response = await res.json();

      this.setState({
        snapshots: response.backups?.sort((a, b) => b.startedAt ? new Date(b.startedAt) - new Date(a.startedAt) : -99999999),
        hasSnapshotsLoaded: true,
        errorMsg: "",
        errorTitle: "",
        networkErr: false,
        displayErrorModal: false
      });
    } catch (err) {
      this.setState({
        errorTitle: "Failed to get snapshots",
        errorMsg: err.message ? err.message : "There was an error while showing the snapshots. Please try again",
        networkErr: true,
        displayErrorModal: true
      })
    }
  }

  startInstanceSnapshot = () => {
    const fakeProgressSnapshot = {
      name: "Preparing snapshot",
      status: "InProgress",
      trigger: "manual",
      sequence: "",
      startedAt: dayjs().format("MM/DD/YY @ hh:mm a z"),
      finishedAt: "",
      expiresAt: "",
      volumeCount: 0,
      volumeSuccessCount: 0,
      volumeBytes: 0,
      volumeSizeHuman: ""
    }

    this.setState({
      startingSnapshot: true,
      startSnapshotErr: false,
      startSnapshotErrorMsg: "",
      isStartButtonClicked: true,
      snapshots: [...this.state.snapshots, fakeProgressSnapshot].sort((a, b) => new Date(b.startedAt) - new Date(a.startedAt))
    }, () => {
      fetch(`${window.env.API_ENDPOINT}/snapshot/backup`, {
        method: "POST",
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        }
      })
        .then(async (result) => {
          if (!result.ok && result.status === 409) {
            const res = await result.json();
            if (res.kotsadmRequiresVeleroAccess) {
              this.setState({
                startingSnapshot: false
              });
              this.props.history.replace("/snapshots/settings");
              return;
            }
          }

          if (result.ok) {
            this.setState({
              startingSnapshot: false
            });
          } else {
            const body = await result.json();
            this.setState({
              startingSnapshot: false,
              startSnapshotErr: true,
              startSnapshotErrorMsg: body.error
            });
          }
        })
        .catch(err => {
          this.setState({
            startingSnapshot: false,
            startSnapshotErr: true,
            startSnapshotErrorMsg: err
          })
        })
    });
  }

  toggleConfirmDeleteModal = snapshot => {
    if (this.state.deleteSnapshotModal) {
      this.setState({ deleteSnapshotModal: false, snapshotToDelete: "", deleteErr: false, deleteErrorMsg: "" });
    } else {
      this.setState({ deleteSnapshotModal: true, snapshotToDelete: snapshot, deleteErr: false, deleteErrorMsg: "" });
    }
  };

  handleDeleteSnapshot = snapshot => {
    const fakeDeletionSnapshot = {
      name: "Preparing for snapshot deletion",
      status: "Deleting",
      trigger: "manual",
      sequence: snapshot.sequence,
      startedAt: Utilities.dateFormat(snapshot.startedAt, "MM/DD/YY @ hh:mm a z"),
      finishedAt: Utilities.dateFormat(snapshot.finishedAt, "MM/DD/YY @ hh:mm a z"),
      expiresAt: Utilities.dateFormat(snapshot.expiresAt, "MM/DD/YY @ hh:mm a z"),
      volumeCount: snapshot.volumeCount,
      volumeSuccessCount: snapshot.volumeSuccessCount,
      volumeBytes: 0,
      volumeSizeHuman: snapshot.volumeSizeHuman
    }

    this.setState({ deletingSnapshot: true, deleteErr: false, deleteErrorMsg: "", snapshots: this.state.snapshots.map(s => s === snapshot ? fakeDeletionSnapshot : s) });

    fetch(`${window.env.API_ENDPOINT}/snapshot/${snapshot.name}/delete`, {
      method: "POST",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    })
      .then(async (res) => {
        if (!res.ok && res.status === 401) {
          Utilities.logoutUser();
          return;
        }

        const response = await res.json();
        if (response.error) {
          this.setState({
            deletingSnapshot: false,
            deleteErr: true,
            deleteErrorMsg: response.error,
          });
          return;
        }

        this.setState({
          deletingSnapshot: false,
          deleteSnapshotModal: false,
          snapshotToDelete: ""
        });
      })
      .catch(err => {
        this.setState({
          deletingSnapshot: false,
          deleteErr: true,
          deleteErrorMsg: err ? err.message : "Something went wrong, please try again.",
        });
      });
  }

  toggleRestoreModal = snapshot => {
    if (this.state.restoreSnapshotModal) {
      this.setState({ restoreSnapshotModal: false, snapshotToRestore: "", selectedRestoreApp: {} });
    } else {
      this.setState({ restoreSnapshotModal: true, snapshotToRestore: snapshot, selectedRestoreApp: snapshot.includedApps[0] });
    }
  };

  fetchSnapshotSettings = async () => {
    this.setState({
      isLoadingSnapshotSettings: true,
      errorMsg: "",
      errorTitle: "",
      displayErrorModal: false
    });
    try {
      const res = await fetch(`${window.env.API_ENDPOINT}/snapshots/settings`, {
        method: "GET",
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        }
      });
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }
        if (res.status === 409) {
          const response = await res.json();
          if (response.kotsadmRequiresVeleroAccess) {
            this.setState({
              isLoadingSnapshotSettings: false
            });
            this.props.history.replace("/snapshots/settings");
            return;
          }
        }
        const err = await res.json();
        this.setState({
          isLoadingSnapshotSettings: false,
          errorTitle: "Failed to get snapshot settings",
          errorMsg: err ? err.error : `Unexpected status code: ${res.status}`,
          networkErr: false,
          displayErrorModal: true
        });
        return;
      }
      const result = await res.json();
      this.setState({
        snapshotSettings: result,
        isLoadingSnapshotSettings: false,
        errorMsg: "",
        errorTitle: "",
        displayErrorModal: false
      })
      if (result?.veleroVersion) {
        this.state.listSnapshotsJob.start(this.listInstanceSnapshots, 2000);
      } else {
        this.props.history.push("/snapshots/settings?configure=true")
      }
    } catch (err) {
      this.setState({
        isLoadingSnapshotSettings: false,
        errorMsg: err,
        errorTitle: "Failed to get snapshot settings",
        displayErrorModal: true
      })
    }
  }

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  }

  onChangeRestoreOption = (selectedRestore) => {
    this.setState({ selectedRestore });
  }

  onChangeRestoreApp = (selectedRestoreApp) => {
    this.setState({ selectedRestoreApp });
  }

  handleApplicationSlugChange = (e) => {
    if (this.state.appSlugMismatch) {
      this.setState({ appSlugMismatch: false });
    }
    this.setState({ appSlugToRestore: e.target.value });
  }

  handlePartialRestoreSnapshot = (snapshot, isOneApp) => {
    const { selectedRestoreApp } = this.state;

    if (isOneApp) {
      if (this.state.appSlugToRestore !== selectedRestoreApp?.slug) {
        this.setState({ appSlugMismatch: true });
        return;
      }
    }

    this.setState({
      restoringSnapshot: true,
      restoreErr: false,
      restoreErrorMsg: "",
    });

    fetch(`${window.env.API_ENDPOINT}/snapshot/${snapshot.name}/restore-apps`, {
      method: "POST",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        appSlugs: [selectedRestoreApp?.slug]
      }),
    })
      .then(async (result) => {
        if (result.ok) {
          this.setState({
            restoringSnapshot: true,
            restoreSnapshotModal: false,
            restoreErr: false,
            restoreErrorMsg: "",
          });

          this.props.history.replace(`/snapshots/${selectedRestoreApp?.slug}/${snapshot.name}/restore`);
        } else {
          const body = await result.json();
          this.setState({
            restoringSnapshot: false,
            restoreErr: true,
            restoreErrorMsg: body.error,
          });
        }
      })
      .catch(err => {
        this.setState({
          restoringSnapshot: false,
          restoreErr: true,
          restoreErrorMsg: err,
        })
      })
  }

  getLabel = ({ iconUri, name, sequence }) => {
    return (
      <div style={{ alignItems: "center", display: "flex", flex: 1 }}>
        <div style={{ display: "flex", flex: 1 }}>
          <span className="app-icon" style={{ fontSize: 18, marginRight: "0.5em", backgroundImage: `url(${iconUri})` }}></span>
          <span style={{ fontSize: 14 }}>{name}</span>
        </div>
        <div style={{ display: "flex" }}>
          <span style={{ fontSize: 14, color: "#9B9B9B", marginLeft: "10px" }}>Sequence {sequence}</span>
        </div>
      </div>
    );
  }

  toggleSnaphotDifferencesModal = () => {
    this.setState({ snapshotDifferencesModal: !this.state.snapshotDifferencesModal });
  }


  render() {
    const { isLoadingSnapshotSettings, snapshotSettings, hasSnapshotsLoaded, startingSnapshot, startSnapshotErr, startSnapshotErrorMsg, snapshots, isStartButtonClicked, displayErrorModal } = this.state;
    const inProgressSnapshotExist = snapshots?.find(snapshot => snapshot.status === "InProgress");


    if (isLoadingSnapshotSettings || (!hasSnapshotsLoaded && !displayErrorModal) || (isStartButtonClicked && snapshots?.length === 0) || startingSnapshot) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    const isVeleroCorrectVersion = snapshotSettings?.isVeleroRunning && snapshotSettings?.veleroVersion.includes("v1.5");

    return (
      <div className="flex1 flex-column u-overflow--auto">
        <Helmet>
          <title>Snapshots</title>
        </Helmet>
        {!isVeleroCorrectVersion ?
          <div className="VeleroWarningBlock">
            <span className="icon small-warning-icon" />
            <p> To use snapshots reliably you have to install velero version 1.5.1 </p>
          </div>
          : null}
        <div className="container flex-column flex1 u-paddingTop--30 u-paddingBottom--20 alignItems--center">
          <div className="AppSnapshots--wrapper flex1 flex-column u-width--full u-marginTop--20">
            <div className="flex flex-auto u-marginBottom--15 alignItems--center justifyContent--spaceBetween">
              <div className="flex1 flex-column" style={{ marginRight: "60px" }}>
                <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-lineHeight--normal">Full Snapshots (Instance) </p>
                <p className="u-marginTop--10 u-fontSize--normal u-lineHeight--more u-fontWeight--medium u-color--dustyGray"> Full snapshots (Instance) back up the Admin Console and all application data. They can be used for full Disaster Recovery; by restoring over top of this instance, or into a new cluster.
                <span className="replicated-link" onClick={this.toggleSnaphotDifferencesModal}> Learn more</span>.</p>
              </div>
              <div className="flex alignSelf--flexEnd">
                <Link to={`/snapshots/settings`} className="replicated-link u-fontSize--small u-fontWeight--bold u-marginRight--20 flex alignItems--center"><span className="icon snapshotSettingsIcon u-marginRight--5" />Settings</Link>
                {snapshots?.length > 0 && snapshotSettings?.veleroVersion &&
                  <span data-for="startSnapshotBtn" data-tip="startSnapshotBtn" data-tip-disable={false}>
                    <button className="btn primary blue" disabled={startingSnapshot || (inProgressSnapshotExist && !startSnapshotErr)} onClick={this.startInstanceSnapshot}>{startingSnapshot ? "Starting a snapshot..." : "Start a snapshot"}</button>
                  </span>}
                {(inProgressSnapshotExist && !startSnapshotErr) &&
                  <ReactTooltip id="startSnapshotBtn" effect="solid" className="replicated-tooltip">
                    <span>You can't start a snapshot while another one is In Progress</span>
                  </ReactTooltip>}
              </div>
            </div>
            {startSnapshotErr ?
              <div className="flex alignItems--center alignSelf--center justifyContent--center u-marginBottom--10">
                <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal">{startSnapshotErrorMsg}</p>
              </div>
              : null}
            {snapshots?.length > 0 && snapshotSettings?.veleroVersion ?
              <div className="flex flex-column">
                {snapshots?.map((snapshot) => (
                  <SnapshotRow
                    key={`snapshot-${snapshot.name}-${snapshot.started}`}
                    snapshot={snapshot}
                    toggleConfirmDeleteModal={this.toggleConfirmDeleteModal}
                    toggleRestoreModal={this.toggleRestoreModal}
                  />
                ))}
              </div> :
              !isStartButtonClicked ?
                <div className="flex flex-column u-position--relative">
                  {[0, 1, 2, 3, 4, 5].map((el) => (<DummySnapshotRow key={el} />
                  ))}
                  <GettingStartedSnapshots isVeleroInstalled={!!snapshotSettings?.veleroVersion} history={this.props.history} startInstanceSnapshot={this.startInstanceSnapshot} />
                </div> : null}
          </div>
          {this.state.deleteSnapshotModal &&
            <DeleteSnapshotModal
              deleteSnapshotModal={this.state.deleteSnapshotModal}
              toggleConfirmDeleteModal={this.toggleConfirmDeleteModal}
              handleDeleteSnapshot={this.handleDeleteSnapshot}
              snapshotToDelete={this.state.snapshotToDelete}
              deleteErr={this.state.deleteErr}
              deleteErrorMsg={this.state.deleteErrorMsg}
            />
          }
          {this.state.restoreSnapshotModal &&
            <BackupRestoreModal
              veleroNamespace={snapshotSettings?.veleroNamespace}
              isMinimalRBACEnabled={snapshotSettings?.isMinimalRBACEnabled}
              restoreSnapshotModal={this.state.restoreSnapshotModal}
              toggleRestoreModal={this.toggleRestoreModal}
              snapshotToRestore={this.state.snapshotToRestore}
              includedApps={this.state.snapshotToRestore?.includedApps}
              selectedRestore={this.state.selectedRestore}
              onChangeRestoreOption={this.onChangeRestoreOption}
              selectedRestoreApp={this.state.selectedRestoreApp}
              onChangeRestoreApp={this.onChangeRestoreApp}
              getLabel={this.getLabel}
              handleApplicationSlugChange={this.handleApplicationSlugChange}
              appSlugToRestore={this.state.appSlugToRestore}
              appSlugMismatch={this.state.appSlugMismatch}
              handlePartialRestoreSnapshot={this.handlePartialRestoreSnapshot}
            />}
          {displayErrorModal &&
            <ErrorModal
              errorModal={displayErrorModal}
              toggleErrorModal={this.toggleErrorModal}
              errMsg={this.state.errorMsg}
              err={this.state.errorTitle}
              tryAgain={this.fetchSnapshotSettings}
              loading={isLoadingSnapshotSettings}
            />}
          {this.state.snapshotDifferencesModal &&
            <SnapshotDifferencesModal
              snapshotDifferencesModal={this.state.snapshotDifferencesModal}
              toggleSnapshotDifferencesModal={this.toggleSnaphotDifferencesModal}
            />}
        </div>
      </div>
    );
  }
}

export default withRouter(Snapshots);