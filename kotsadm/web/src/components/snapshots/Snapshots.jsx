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
    snapshotSettingsErr: false,
    snapshotSettingsErrMsg: "",

    snapshots: [],
    hasSnapshotsLoaded: false,
    isStartButtonClicked: false,
    snapshotsListErr: false,
    snapshotsListErrMsg: "",
    listSnapshotsJob: new Repeater(),
    networkErr: false,
    displayErrorModal: false
  };

  componentDidMount() {
    this.fetchSnapshotSettings();
    this.state.listSnapshotsJob.start(this.listInstanceSnapshots, 2000);
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
      snapshotsListErr: false,
      snapshotsListErrMsg: "",
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
        this.setState({
          snapshotsListErr: true,
          snapshotsListErrMsg: `Unexpected status code: ${res.status}`,
          networkErr: false,
          displayErrorModal: true
        });
        return;
      }
      const response = await res.json();

      this.setState({
        snapshots: response.backups?.sort((a, b) => b.startedAt ? new Date(b.startedAt) - new Date(a.startedAt) : -99999999),
        hasSnapshotsLoaded: true,
        snapshotsListErr: false,
        snapshotsListErrMsg: "",
        networkErr: false,
        displayErrorModal: false
      });
    } catch (err) {
      this.setState({
        snapshotsListErr: true,
        snapshotsListErrMsg: err.message ? err.message : "There was an error while showing the snapshots. Please try again",
        networkErr: true,
        displayErrorModal: true
      })
    }
  }

  startInstanceSnapshot =  () => {
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

  fetchSnapshotSettings = () => {
    this.setState({
      isLoadingSnapshotSettings: true,
      snapshotSettingsErr: false,
      snapshotSettingsErrMsg: ""
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
      })
      .catch(err => {
        this.setState({
          isLoadingSnapshotSettings: false,
          snapshotSettingsErr: true,
          snapshotSettingsErrMsg: err,
        })
      })
  }


  render() {
    const { isLoadingSnapshotSettings, snapshotSettings, hasSnapshotsLoaded, startingSnapshot, startSnapshotErr, startSnapshotErrorMsg, snapshots, isStartButtonClicked } = this.state;
    const { isKurlEnabled } = this.props;
    const inProgressSnapshotExist = snapshots?.find(snapshot => snapshot.status === "InProgress");

    if (isLoadingSnapshotSettings || !hasSnapshotsLoaded || (isStartButtonClicked && snapshots?.length === 0) || startingSnapshot) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    return (
      <div className="container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20 alignItems--center">
        <Helmet>
          <title>Snapshots</title>
        </Helmet>
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
      </div>
    );
  }
}

export default withRouter(Snapshots);