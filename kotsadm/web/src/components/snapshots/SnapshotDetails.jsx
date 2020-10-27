import React, { Component } from "react";
import { Link, withRouter } from "react-router-dom"
import Helmet from "react-helmet";
import SnapshotDetailsRow from "./SnapshotDetailsRow";
import BackupRestoreModal from "../modals/BackupRestoreModal";
import DeleteSnapshotModal from "../modals/DeleteSnapshotModal";

import "../../scss/components/snapshots/AppSnapshots.scss";
import { Utilities } from "../../utilities/utilities";


class SnapshotDetails extends Component {
  state = {
    deleteSnapshotModal: false,
    snapshotToDelete: "",
    deleteErr: false,
    deleteErrorMsg: "",
    restoreSnapshotModal: false,
    snapshotToRestore: "",
  };

  toggleConfirmDeleteModal = snapshot => {
    if (this.state.deleteSnapshotModal) {
      this.setState({ deleteSnapshotModal: false, snapshotToDelete: "", deleteErr: false, deleteErrorMsg: "" });
    } else {
      this.setState({ deleteSnapshotModal: true, snapshotToDelete: snapshot, deleteErr: false, deleteErrorMsg: "" });
    }
  };

  handleDeleteSnapshot = snapshot => {
    console.log("delete", snapshot);
  }

  toggleRestoreModal = snapshot => {
    if (this.state.restoreSnapshotModal) {
      this.setState({ restoreSnapshotModal: false, snapshotToRestore: "" });
    } else {
      this.setState({ restoreSnapshotModal: true, snapshotToRestore: snapshot });
    }
  };


  render() {
    const snapshots = [
      {
        appID: "1jR0VjB2Vm1lxrqoE3H0BTer6rd",
        expiresAt: "2020-11-25T22:56:15Z",
        finishedAt: "2020-10-26T22:56:22Z",
        name: "qakots-g4bjh",
        sequence: 0,
        startedAt: "2020-10-26T22:56:15Z",
        status: "Completed",
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
    return (
      <div className="container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20 alignItems--center">
        <Helmet>
          <title>Snapshots</title>
        </Helmet>
        <div className="AppSnapshots--wrapper flex1 flex-column u-width--full">
          <div className="flex flex-auto alignItems--flexStart justifyContent--spaceBetween u-borderBottom--gray darker">
            <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-lineHeight--normal u-marginBottom--15">Snapshots</p>
            <div className="flex u-marginBottom--15">
              <Link to={`/snapshots`} className="replicated-link u-fontSize--small u-fontWeight--bold u-marginRight--20 flex alignItems--center"><span className="icon snapshotSettingsIcon u-marginRight--5" />Settings</Link>
              <span data-for="startSnapshotBtn" data-tip="startSnapshotBtn" data-tip-disable={false}>
                <button className="btn primary blue"> Start a snapshot</button>
              </span>
            </div>
          </div>
          <div className="flex flex-column">
            {snapshots?.map((snapshot, i) => (
              <SnapshotDetailsRow
                key={`snapshot-${snapshot.name}-${snapshot.started}`}
                snapshot={snapshot}
                toggleConfirmDeleteModal={this.toggleConfirmDeleteModal}
                toggleRestoreModal={this.toggleRestoreModal}
              />
            ))}
          </div>
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

export default withRouter(SnapshotDetails);
