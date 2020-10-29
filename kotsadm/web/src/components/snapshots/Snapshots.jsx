import React, { Component } from "react";
import { Link, withRouter } from "react-router-dom";
import Helmet from "react-helmet";
import Loader from "../shared/Loader";
import SnapshotRow from "./SnapshotRow";
import BackupRestoreModal from "../modals/BackupRestoreModal";
import DeleteSnapshotModal from "../modals/DeleteSnapshotModal";
import DummySnapshotRow from "./DummySnapshotRow";
import GettingStartedSnapshots from "./GettingStartedSnapshots";

import "../../scss/components/snapshots/AppSnapshots.scss";
import { Utilities } from "../../utilities/utilities";


class Snapshots extends Component {
  state = {
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
    //dummy snapshots
    snapshots: [
      {
        appID: "1jR0VjB2Vm1lxrqoE3H0BTer6rd",
        expiresAt: "2020-11-25T22:56:15Z",
        finishedAt: "2020-10-26T22:56:22Z",
        name: "qakots-g4bjh",
        sequence: 0,
        startedAt: "2020-10-26T22:56:15Z",
        status: "PartiallyFailed",
        supportBundleId: "backup-qakots-g4bjh",
        trigger: "manual",
        volumeBytes: 0,
        volumeCount: 0,
        volumeSizeHuman: "0B",
        volumeSuccessCount: 0
      },
      {
        appID: "1jR0VjB2Vm1lxrqoE3H0BTer6rk",
        expiresAt: "2020-11-27T20:56:15Z",
        finishedAt: "2020-10-26T20:56:22Z",
        name: "qakots-g4bjk",
        sequence: 0,
        startedAt: "2020-10-25T20:56:15Z",
        status: "Completed",
        supportBundleId: "backup-qakots-g4bjh",
        trigger: "manual",
        volumeBytes: 4,
        volumeCount: 0,
        volumeSizeHuman: "4B",
        volumeSuccessCount: 0
      }
    ]
  };

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

  componentDidMount() {
    this.fetchSnapshotSettings();
  }


  render() {
    const { isLoadingSnapshotSettings, snapshotSettings, snapshots } = this.state;

    if (isLoadingSnapshotSettings) {
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
          <div className={`flex flex-auto alignItems--flexStart justifyContent--spaceBetween ${(snapshots?.length > 0 && snapshotSettings?.veleroVersion !== "") && "u-borderBottom--gray darker"}`}>
            <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-lineHeight--normal u-marginBottom--15">Snapshots</p>
            <div className="flex u-marginBottom--15">
              <Link to={`/snapshots/settings`} className="replicated-link u-fontSize--small u-fontWeight--bold u-marginRight--20 flex alignItems--center"><span className="icon snapshotSettingsIcon u-marginRight--5" />Settings</Link>
              <span data-for="startSnapshotBtn" data-tip="startSnapshotBtn" data-tip-disable={false}>
                <button className="btn primary blue"> Start a snapshot</button>
              </span>
            </div>
          </div>
          {snapshots?.length > 0  && snapshotSettings?.veleroVersion !== "" ?
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
            <div className="flex flex-column u-position--relative">
              {[0, 1, 2, 3, 4, 5].map((el) => (<DummySnapshotRow key={el}/>
              ))}
              <GettingStartedSnapshots isVeleroInstalled={snapshotSettings?.veleroVersion !== ""} history={this.props.history} />
            </div>}
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
