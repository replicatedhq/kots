import React, { Component } from "react";
import { Link, withRouter } from "react-router-dom"
import Helmet from "react-helmet";
import Modal from "react-modal";
import ReactTooltip from "react-tooltip"
import moment from "moment";

import AppSnapshotRow from "./AppSnapshotRow";
import ScheduleSnapshotForm from "../shared/ScheduleSnapshotForm";
import Loader from "../shared/Loader";
import DeleteSnapshotModal from "../modals/DeleteSnapshotModal";
import RestoreSnapshotModal from "../modals/RestoreSnapshotModal";
import ErrorModal from "../modals/ErrorModal";

import "../../scss/components/snapshots/AppSnapshots.scss";
import { Utilities } from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";


class AppSnapshots extends Component {
  state = {
    displayScheduleSnapshotModal: false,
    deleteSnapshotModal: false,
    startingSnapshot: false,
    startSnapshotErr: false,
    startSnapshotErrorMsg: "",
    snapshotsListErr: false,
    snapshotsListErrMsg: "",
    snapshotToDelete: "",
    deletingSnapshot: false,
    deleteErr: false,
    deleteErrorMsg: "",
    restoreSnapshotModal: false,
    restoringSnapshot: false,
    snapshotToRestore: "",
    restoreErr: false,
    restoreErrorMsg: "",
    hideCheckVeleroButton: false,
    snapshots: [],
    hasSnapshotsLoaded: false,
    snapshotSettings: null,
    isLoadingSnapshotSettings: true,
    isStartButtonClicked: false,
    restoreInProgressErr: false,
    restoreInProgressMsg: "",
    appSlugToRestore: "",
    appSlugMismatch: false,
    listSnapshotsJob: new Repeater(),
    networkErr: false,
    displayErrorModal: false
  };

  componentDidMount = async () => {
    await this.fetchSnapshotSettings();

    this.checkRestoreInProgress();
    this.state.listSnapshotsJob.start(this.listSnapshots, 2000);
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

  checkRestoreInProgress() {
    const { app } = this.props;
    fetch(`${window.env.API_ENDPOINT}/app/${app.slug}/snapshot/restore/status`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    })
      .then(async (result) => {
        const body = await result.json();
        if (body.error) {
          this.setState({
            restoreInProgressErr: true,
            restoreInProgressMsg: body.error
          });
        } else if (body.status == "running") {
          this.props.history.replace(`/app/${this.props.app.slug}/snapshots/${body.restore_name}/restore`);
        } else {
          this.state.listSnapshotsJob.start(this.listSnapshots, 2000);
        }
      })
      .catch(err => {
        this.setState({
          restoreInProgressErr: true,
          restoreInProgressMsg: err
        });
      })
  }

  listSnapshots = async () => {
    const { app } = this.props;
    this.setState({
      snapshotsListErr: false,
      snapshotsListErrMsg: "",
      networkErr: false,
      displayErrorModal: false
    })
    try {
      const res = await fetch(`${window.env.API_ENDPOINT}/app/${app.slug}/snapshots`, {
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

  fetchSnapshotSettings = async () => {
    this.setState({
      isLoadingSnapshotSettings: true,
      snapshotSettingsErr: false,
      snapshotSettingsErrMsg: "",
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

  toggleScheduleSnapshotModal = () => {
    this.setState({ displayScheduleSnapshotModal: !this.state.displayScheduleSnapshotModal });
  }

  toggleConfirmDeleteModal = snapshot => {
    if (this.state.deleteSnapshotModal) {
      this.setState({ deleteSnapshotModal: false, snapshotToDelete: "", deleteErr: false, deleteErrorMsg: "" });
    } else {
      this.setState({ deleteSnapshotModal: true, snapshotToDelete: snapshot, deleteErr: false, deleteErrorMsg: "" });
    }
  };

  toggleRestoreModal = snapshot => {
    if (this.state.restoreSnapshotModal) {
      this.setState({ restoreSnapshotModal: false, snapshotToRestore: "", restoreErr: false, restoreErrorMsg: "" });
    } else {
      this.setState({ restoreSnapshotModal: true, snapshotToRestore: snapshot, restoreErr: false, restoreErrorMsg: "" });
    }
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  }

  handleDeleteSnapshot = snapshot => {
    const fakeDeletionSnapshot = {
      name: "Preparing for snapshot deletion",
      status: "Deleting",
      trigger: "manual",
      appID: this.props.app.id,
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
  };

  handleRestoreSnapshot = snapshot => {
    const { app } = this.props;

    if (this.state.appSlugToRestore !== app?.slug) {
      this.setState({ appSlugMismatch: true });
      return;
    }

    this.setState({
      restoringSnapshot: true,
      restoreErr: false,
      restoreErrorMsg: "",
    });

    fetch(`${window.env.API_ENDPOINT}/app/${app?.slug}/snapshot/restore/${snapshot.name}`, {
      method: "POST",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    })
      .then(async (result) => {
        if (result.ok) {
          this.setState({
            restoringSnapshot: true,
            restoreSnapshotModal: false,
            restoreErr: false,
            restoreErrorMsg: "",
          });

          this.props.history.replace(`/app/${this.props.app.slug}/snapshots/${snapshot.name}/restore`);
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

  startManualSnapshot = () => {
    const { app } = this.props;

    const fakeProgressSnapshot = {
      name: "Preparing snapshot",
      status: "InProgress",
      trigger: "manual",
      appID: app.id,
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
    });

    fetch(`${window.env.API_ENDPOINT}/app/${app.slug}/snapshot/backup`, {
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
            startSnapshotErrorMsg: body.error,
          });
        }
      })
      .catch(err => {
        this.setState({
          startingSnapshot: false,
          startSnapshotErr: true,
          startSnapshotErrorMsg: err,
        })
      })
  }

  handleApplicationSlugChange = (e) => {
    if (this.state.appSlugMismatch) {
      this.setState({ appSlugMismatch: false });
    }
    this.setState({ appSlugToRestore: e.target.value });
  }


  render() {
    const {
      displayScheduleSnapshotModal,
      startingSnapshot,
      startSnapshotErr,
      startSnapshotErrorMsg,
      deleteSnapshotModal,
      snapshotToDelete,
      deletingSnapshot,
      deleteErr,
      deleteErrorMsg,
      restoreSnapshotModal,
      restoringSnapshot,
      snapshotToRestore,
      restoreErr,
      restoreErrorMsg,
      snapshots,
      hasSnapshotsLoaded,
      snapshotSettings,
      isLoadingSnapshotSettings,
      isStartButtonClicked,
      snapshotsListErr,
      snapshotsListErrMsg,
      restoreInProgressErr,
      restoreInProgressErrMsg
    } = this.state;
    const { app } = this.props;
    const appTitle = app?.name;
    const inProgressSnapshotExist = snapshots?.find(snapshot => snapshot.status === "InProgress");

    if (isLoadingSnapshotSettings || !hasSnapshotsLoaded || (isStartButtonClicked && snapshots?.length === 0) || startingSnapshot) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    if (!snapshotSettings?.store) {
      this.props.history.replace("/snapshots");
    }

    if (restoreInProgressErr) {
      return (
        <div class="flex1 flex-column justifyContent--center alignItems--center">
          <span className="icon redWarningIcon" />
          <p className="u-color--chestnut u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-marginTop--10">{restoreInProgressErrMsg}</p>
        </div>
      )
    }

    if (snapshotsListErr || !snapshots) {
      return (
        <ErrorModal
          errorModal={this.state.displayErrorModal}
          toggleErrorModal={this.toggleErrorModal}
          errMsg={snapshotsListErrMsg}
          tryAgain={this.listSnapshots}
          err="Failed to get snapshots"
          loading={false}
        />
      )
    }

    if (hasSnapshotsLoaded && !isStartButtonClicked && snapshots?.length === 0) {
      return (
        <div className="container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20 justifyContent--center alignItems--center">
          <div className="flex-column u-textAlign--center AppSnapshotsEmptyState--wrapper">
            <p className="u-fontSize--largest u-fontWeight--bold u-color--tundora u-marginBottom--10">No snapshots have been made</p>
            <p className="u-fontSize--normal u-fontWeight--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--30">There have been no snapshots made for {appTitle || "your application"} yet. You can manually trigger snapshots or you can set up automatic snapshots to be made on a custom schedule.</p>
            <div className="flex justifyContent--center">
              <div className="flex-auto u-marginRight--20">
                <button className="btn secondary blue" disabled={startingSnapshot} onClick={this.startManualSnapshot}>{startingSnapshot ? "Starting a snapshot..." : "Start a snapshot"}</button>
              </div>
              <div className="flex-auto">
                <Link to={`/app/${app.slug}/snapshots/schedule`} className="btn primary blue">Schedule snapshots</Link>
              </div>
            </div>
          </div>
        </div>
      )
    }

    return (
      <div className="container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20 alignItems--center">
        <Helmet>
          <title>{`${appTitle} Snapshots`}</title>
        </Helmet>
        <div className="AppSnapshots--wrapper flex1 flex-column u-width--full">
          <div className="flex flex-auto alignItems--flexStart justifyContent--spaceBetween">
            <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-lineHeight--normal u-marginBottom--10">Snapshots</p>
            {startSnapshotErr ?
              <div className="flex flex1 u-marginLeft--10 alignItems--center alignSelf--center u-marginBottom--10">
                <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal">{startSnapshotErrorMsg}</p>
              </div>
              : null}
            <div className="flex">
              <Link to={`/snapshots/settings`} className="replicated-link u-fontSize--small u-fontWeight--bold u-marginRight--20 flex alignItems--center"><span className="icon snapshotSettingsIcon u-marginRight--5" />Settings</Link>
              <Link to={`/app/${app.slug}/snapshots/schedule`} className="replicated-link u-fontSize--small u-fontWeight--bold u-marginRight--20 flex alignItems--center"><span className="icon snapshotScheduleIcon u-marginRight--5" />Schedule</Link>
              <span data-for="startSnapshotBtn" data-tip="startSnapshotBtn" data-tip-disable={false}>
                <button className="btn primary blue" disabled={startingSnapshot || inProgressSnapshotExist} onClick={this.startManualSnapshot}>{startingSnapshot ? "Starting a snapshot..." : "Start a snapshot"}</button>
              </span>
              {inProgressSnapshotExist &&
                <ReactTooltip id="startSnapshotBtn" effect="solid" className="replicated-tooltip">
                  <span>You can't start a snapshot while another one is In Progress</span>
                </ReactTooltip>}
            </div>
          </div>
          {snapshots?.map((snapshot) => (
            <AppSnapshotRow
              key={`snapshot-${snapshot.name}-${snapshot.started}`}
              snapshot={snapshot}
              appSlug={app.slug}
              toggleConfirmDeleteModal={this.toggleConfirmDeleteModal}
              toggleRestoreModal={this.toggleRestoreModal}
            />
          ))
          }
        </div>
        {displayScheduleSnapshotModal &&
          <Modal
            isOpen={displayScheduleSnapshotModal}
            onRequestClose={this.toggleScheduleSnapshotModal}
            shouldReturnFocusAfterClose={false}
            contentLabel="Schedule snapshot modal"
            ariaHideApp={false}
            className="ScheduleSnapshotModal--wrapper MediumSize Modal"
          >
            <div className="Modal-body">
              <ScheduleSnapshotForm
                onSubmit={this.handleScheduleSubmit}
              />
              <div className="u-marginTop--10 flex">
                <button onClick={this.toggleScheduleSnapshotModal} className="btn secondary blue u-marginRight--10">Cancel</button>
                <button onClick={this.scheduleSnapshot} className="btn primary blue">Save</button>
              </div>
            </div>
          </Modal>
        }
        {deleteSnapshotModal &&
          <DeleteSnapshotModal
            deleteSnapshotModal={deleteSnapshotModal}
            toggleConfirmDeleteModal={this.toggleConfirmDeleteModal}
            handleDeleteSnapshot={this.handleDeleteSnapshot}
            snapshotToDelete={snapshotToDelete}
            deletingSnapshot={deletingSnapshot}
            deleteErr={deleteErr}
            deleteErrorMsg={deleteErrorMsg}
          />
        }
        {restoreSnapshotModal &&
          <RestoreSnapshotModal
            restoreSnapshotModal={restoreSnapshotModal}
            toggleRestoreModal={this.toggleRestoreModal}
            handleRestoreSnapshot={this.handleRestoreSnapshot}
            snapshotToRestore={snapshotToRestore}
            restoringSnapshot={restoringSnapshot}
            restoreErr={restoreErr}
            restoreErrorMsg={restoreErrorMsg}
            app={this.props.app}
            appSlugToRestore={this.state.appSlugToRestore}
            appSlugMismatch={this.state.appSlugMismatch}
            handleApplicationSlugChange={this.handleApplicationSlugChange}
          />
        }
      </div>
    );
  }
}

export default withRouter(AppSnapshots);
