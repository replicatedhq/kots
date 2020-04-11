import React, { Component } from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { Link, withRouter } from "react-router-dom"
import Helmet from "react-helmet";
import AppSnapshotsRow from "./AppSnapshotRow";
import ScheduleSnapshotForm from "../shared/ScheduleSnapshotForm";
import Loader from "../shared/Loader";
import Modal from "react-modal";
import { deleteSnapshot, restoreSnapshot } from "../../mutations/SnapshotMutations";
import "../../scss/components/snapshots/AppSnapshots.scss";
import DeleteSnapshotModal from "../modals/DeleteSnapshotModal";
import RestoreSnapshotModal from "../modals/RestoreSnapshotModal";
import { Utilities } from "../../utilities/utilities";

class AppSnapshots extends Component {
  state = {
    displayScheduleSnapshotModal: false,
    deleteSnapshotModal: false,
    startingSnapshot: false,
    startSnapshotErr: false,
    startSnapshotErrorMsg: "",
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
    isStartButtonClicked: false
  };

  componentDidMount = async () => {
    await this.fetchSnapshotSettings();

    this.listSnapshots();
    this.interval = setInterval(()=> this.listSnapshots(), 2000);
  }
  
  componentWillUnmount() {
    window.clearInterval(this.interval);
  }

  listSnapshots() {
    const { app } = this.props;
    fetch(`${window.env.API_ENDPOINT}/app/${app.slug}/snapshots`, {
      method: "GET",
      headers: {
        "Authorization": `${Utilities.getToken()}`,
        "Content-Type": "application/json",
      }
    })
    .then(async (result) => {
      const body = await result.json();
      if (!result.ok) {
        console.log("failed to load snapshots", body);
        return;
      }
      this.setState({
        snapshots: body.backups,
        hasSnapshotsLoaded: true,
      });
    })
    .catch(err => {
      console.log(err);
    })
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
        "Authorization": `${Utilities.getToken()}`,
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
      console.log(err);
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

  handleDeleteSnapshot = snapshot => {
    this.setState({ deletingSnapshot: true, deleteErr: false, deleteErrorMsg: "" });
    this.props
      .deleteSnapshot(snapshot.name)
      .then(() => {
        this.setState({
          deletingSnapshot: false,
          deleteSnapshotModal: false,
          snapshotToDelete: ""
        });
      })
      .catch(err => {
        err.graphQLErrors.map(({ msg }) => {
          this.setState({
            deletingSnapshot: false,
            deleteErr: true,
            deleteErrorMsg: msg,
          });
        })
      })
      .finally(() => {
        this.setState({ deletingSnapshot: false });
      });
  };

  handleRestoreSnapshot = snapshot => {
    this.setState({ restoringSnapshot: true, restoreErr: false, restoreErrorMsg: "" });
    this.props
      .restoreSnapshot(snapshot.name)
      .then((res) => {
        this.setState({
          restoringSnapshot: false,
          restoreSnapshotModal: false,
          snapshotToRestore: ""
        });
        this.props.history.push(`/app/${this.props.app.slug}/snapshots/${res.data.restoreSnapshot.name}/restore`);
      })
      .catch(err => {
        err.graphQLErrors.map(({ msg }) => {
          this.setState({
            restoringSnapshot: false,
            restoreErr: true,
            restoreErrorMsg: msg,
          });
        })
      })
      .finally(() => {
        this.setState({ restoringSnapshot: false });
      });
  }

  startManualSnapshot = () => {
    const { app } = this.props;
    this.setState({
      startingSnapshot: true,
      startSnapshotErr: false,
      startSnapshotErrorMsg: "",
      isStartButtonClicked: true
    });

    fetch(`${window.env.API_ENDPOINT}/app/${app.slug}/snapshot/backup`, {
      method: "POST",
      headers: {
        "Authorization": `${Utilities.getToken()}`,
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
      console.log(err);
      this.setState({
        startingSnapshot: false,
        startSnapshotErr: true,
        startSnapshotErrorMsg: err,
      })
    })
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
      isStartButtonClicked
    } = this.state;
    const { app } = this.props;
    const appTitle = app.name;

    if (isLoadingSnapshotSettings || (isStartButtonClicked && snapshots.length === 0)) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    if (!snapshotSettings?.store) {
      this.props.history.replace("/snapshots");
    }

    if (hasSnapshotsLoaded && !isStartButtonClicked && snapshots.length === 0) {
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
              <div class="flex flex1 u-marginLeft--10 alignItems--center alignSelf--center u-marginBottom--10">
                <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal">{startSnapshotErrorMsg}</p>
              </div>
              : null}
            <div className="flex">
              <Link to={`/snapshots`} className="replicated-link u-fontSize--small u-fontWeight--bold u-marginRight--20 flex alignItems--center"><span className="icon snapshotSettingsIcon u-marginRight--5" />Settings</Link>
              <Link to={`/app/${app.slug}/snapshots/schedule`} className="replicated-link u-fontSize--small u-fontWeight--bold u-marginRight--20 flex alignItems--center"><span className="icon snapshotScheduleIcon u-marginRight--5" />Schedule</Link>
              <button className="btn primary blue" disabled={startingSnapshot} onClick={this.startManualSnapshot}>{startingSnapshot ? "Starting a snapshot..." : "Start a snapshot"}</button>
            </div>
          </div>
          {snapshots.map((snapshot) => (
            <AppSnapshotsRow
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
          />
        }
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  graphql(deleteSnapshot, {
    props: ({ mutate }) => ({
      deleteSnapshot: (snapshotName) => mutate({ variables: { snapshotName } })
    })
  }),
  graphql(restoreSnapshot, {
    props: ({ mutate }) => ({
      restoreSnapshot: (snapshotName) => mutate({ variables: { snapshotName } })
    })
  })
)(AppSnapshots);
