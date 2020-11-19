import React, { Component } from "react";
import { Link, withRouter } from "react-router-dom";
import Helmet from "react-helmet";
import ReactTooltip from "react-tooltip";
import moment from "moment";

import Loader from "../shared/Loader";
import SnapshotRow from "./SnapshotRow";
import BackupRestoreModal from "../modals/BackupRestoreModal";
import DeleteSnapshotModal from "../modals/DeleteSnapshotModal";
import DummySnapshotRow from "./DummySnapshotRow";
import GettingStartedSnapshots from "./GettingStartedSnapshots";
import ErrorModal from "../modals/ErrorModal";

import "../../scss/components/snapshots/AppSnapshots.scss";
import { Utilities } from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";


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
    displayErrorModal: false
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
      startedAt: moment().format("MM/DD/YY @ hh:mm a"),
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
      startedAt: Utilities.dateFormat(snapshot.startedAt, "MM/DD/YY @ hh:mm a"),
      finishedAt: Utilities.dateFormat(snapshot.finishedAt, "MM/DD/YY @ hh:mm a"),
      expiresAt: Utilities.dateFormat(snapshot.expiresAt, "MM/DD/YY @ hh:mm a"),
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
      this.setState({ restoreSnapshotModal: false, snapshotToRestore: "" });
    } else {
      this.setState({ restoreSnapshotModal: true, snapshotToRestore: snapshot });
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
      })
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
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


  render() {
    const { isLoadingSnapshotSettings, snapshotSettings, hasSnapshotsLoaded, startingSnapshot, startSnapshotErr, startSnapshotErrorMsg, snapshots, isStartButtonClicked } = this.state;
    const { isKurlEnabled } = this.props;
    const inProgressSnapshotExist = snapshots?.find(snapshot => snapshot.status === "InProgress");

    if (isLoadingSnapshotSettings && !hasSnapshotsLoaded) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    const isVeleroCorrectVersion = snapshotSettings?.isVeleroRunning && snapshotSettings?.veleroVersion.includes("v1.5");
    const snapshotApp = this.props.appsList?.find(app => app.allowSnapshots);


    return (
      <div className="flex1 flex-column u-overflow--auto">
        <Helmet>
          <title>Snapshots</title>
        </Helmet>
        {!isVeleroCorrectVersion ?
          <div className="VeleroWarningBlock">
            <span className="icon snapshot-warning-icon" />
            <p> To use snapshots reliably you have to install velero version 1.5.1 </p>
          </div>
          : null}
        <div className="container flex-column flex1 u-paddingTop--30 u-paddingBottom--20 alignItems--center">
          <div className="InfoSnapshots--wrapper flex flex-auto u-marginBottom--20">
            <span className="icon snapshot-getstarted-icon flex-auto u-marginRight--20 u-marginTop--5" />
            <div className="flex-column">
              <p className="u-fontSize--large u-fontWeight--bold u-lineHeight--normal u-color--tundora flex alignItems--center"> Instance Snapshots <span className="beta-tag u-marginLeft--5"> beta </span> </p>
              <p className="u-fontSize--small u-fontWeight--normal u-lineHeight--normal u-color--doveGray u-marginTop--5">
                Instance snapshots back up the Admin Console and all application data. They can be used for full Disaster Recovery; by restoring over top of this instance, or into a new cluster.
              </p>
              <p className="u-fontSize--small u-fontWeight--normal u-lineHeight--normal u-color--doveGray u-marginTop--5">
                If you only need a partial backup of just application volumes and manifests for rollbacks, <Link to={`/app/${snapshotApp?.slug}/snapshots`} className="replicated-link u-fontSize--small">use Application Snapshots</Link>.
              </p>
            </div>
          </div>
          <div className="AppSnapshots--wrapper flex1 flex-column u-width--full">
            <div className={`flex flex-auto alignItems--center justifyContent--spaceBetween ${(snapshots?.length > 0 && snapshotSettings?.veleroVersion !== "") && "u-borderBottom--gray darker"}`}>
              <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-lineHeight--normal u-marginBottom--15">Snapshots</p>
              {startSnapshotErr ?
                <div className="flex flex1 alignItems--center alignSelf--center justifyContent--center u-marginBottom--10">
                  <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal">{startSnapshotErrorMsg}</p>
                </div>
                : null}
              <div className="flex u-marginBottom--15">
                <Link to={`/snapshots/settings`} className="replicated-link u-fontSize--small u-fontWeight--bold u-marginRight--20 flex alignItems--center"><span className="icon snapshotSettingsIcon u-marginRight--5" />Settings</Link>
                {snapshotSettings?.veleroVersion !== "" && !isKurlEnabled &&
                  <span data-for="startSnapshotBtn" data-tip="startSnapshotBtn" data-tip-disable={false}>
                    <button className="btn primary blue" disabled={startingSnapshot || (inProgressSnapshotExist && !startSnapshotErr)} onClick={this.startInstanceSnapshot}>{startingSnapshot ? "Starting a snapshot..." : "Start a snapshot"}</button>
                  </span>}
                {(inProgressSnapshotExist && !startSnapshotErr) &&
                  <ReactTooltip id="startSnapshotBtn" effect="solid" className="replicated-tooltip">
                    <span>You can't start a snapshot while another one is In Progress</span>
                  </ReactTooltip>}
              </div>
            </div>
            {snapshots?.length > 0 && snapshotSettings?.veleroVersion !== "" && !isKurlEnabled ?
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
                  <GettingStartedSnapshots isVeleroInstalled={snapshotSettings?.veleroVersion !== ""} history={this.props.history} startInstanceSnapshot={this.startInstanceSnapshot} isKurlEnabled={isKurlEnabled} />
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
              restoreSnapshotModal={this.state.restoreSnapshotModal}
              toggleRestoreModal={this.toggleRestoreModal}
              snapshotToRestore={this.state.snapshotToRestore}
            />}
          {this.state.displayErrorModal &&
            <ErrorModal
              errorModal={this.state.displayErrorModal}
              toggleErrorModal={this.toggleErrorModal}
              errMsg={this.state.errorMsg}
              err={this.state.errorTitle}
              tryAgain={this.fetchSnapshotSettings}
              loading={isLoadingSnapshotSettings}
            />}
        </div>
      </div>
    );
  }
}

export default withRouter(Snapshots);