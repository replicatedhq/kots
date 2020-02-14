import React, { Component } from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { Link, withRouter } from "react-router-dom"
import Helmet from "react-helmet";
import AppSnapshotsRow from "./AppSnapshotRow";
import ScheduleSnapshotForm from "../shared/ScheduleSnapshotForm";
import Loader from "../shared/Loader";
import Modal from "react-modal";
import { listSnapshots, snapshotSettings } from "../../queries/SnapshotQueries";
import { manualSnapshot, deleteSnapshot, restoreSnapshot } from "../../mutations/SnapshotMutations";
import "../../scss/components/snapshots/AppSnapshots.scss";
import DeleteSnapshotModal from "../modals/DeleteSnapshotModal";
import RestoreSnapshotModal from "../modals/RestoreSnapshotModal";
import AppSnapshotSettings from "./AppSnapshotSettings";

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
    restoreErrorMsg: ""
  };

  componentDidMount() {
    this.props.snapshots.startPolling(2000);
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
    this.setState({ startingSnapshot: true, startSnapshotErr: false, startSnapshotErrorMsg: "" });
    this.props.manualSnapshot(app.id)
      .then(() => {
        this.setState({ startingSnapshot: false });
        this.props.snapshots.refetch();
      })
      .catch(err => {
        err.graphQLErrors.map(({ msg }) => {
          this.setState({
            startingSnapshot: false,
            startSnapshotErr: true,
            startSnapshotErrorMsg: msg,
          });
        })
      })
      .finally(() => {
        this.setState({ startingSnapshot: false });
      });
  }

  handleScheduleSubmit = () => {
    console.log("schedule mutation to be implemented");
  }

  checkForVelero = () => {
    console.log("implement check for velero");
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
      restoreErrorMsg
    } = this.state;
    const { app, snapshots, snapshotSettings } = this.props;
    const appTitle = app.name;
    const veleroInstalled = true;

    if (snapshots?.loading ||snapshotSettings?.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    if (!veleroInstalled) {
      return (
        <div className="container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20 justifyContent--center alignItems--center">
          <div className="flex-column u-textAlign--center AppSnapshotsEmptyState--wrapper">
            <p className="u-fontSize--largest u-fontWeight--bold u-color--tundora u-marginBottom--10">Configure application snapshots</p>
            <p className="u-fontSize--normal u-fontWeight--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--30">Snapshots are enabled for {appTitle || "your application"} however you need to install Velero before you will be able to capture any snapshots. After installing Velero on your cluster click the button below so that kotsadm can pick it up and you can begin creating applicaiton snapshots.</p>
            <div className="u-textAlign--center">
              <button className="btn primary blue" onClick={this.checkForVelero}>Check for Velero</button>
            </div>
          </div>
        </div>
      )
    }

    if (!snapshotSettings.snapshotConfig?.store) {
      return (
        <AppSnapshotSettings noSnapshotsView={true} app={app} startingSnapshot={startingSnapshot} startManualSnapshot={this.startManualSnapshot} refetchSnapshotSettings={this.props.snapshotSettings?.refetch()} />
      )
    }

    if (!snapshots.loading && !snapshots?.listSnapshots?.length) {
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
              <div class="flex flex1">
                <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal">{startSnapshotErrorMsg}</p>
              </div>
              : null}
            <div className="flex">
              <Link to={`/app/${app.slug}/snapshots/settings`} className="replicated-link u-fontSize--small u-fontWeight--bold u-marginRight--20 flex alignItems--center"><span className="icon snapshotSettingsIcon u-marginRight--5" />Settings</Link>
              <Link to={`/app/${app.slug}/snapshots/schedule`} className="replicated-link u-fontSize--small u-fontWeight--bold u-marginRight--20 flex alignItems--center"><span className="icon snapshotScheduleIcon u-marginRight--5" />Schedule</Link>
              <button className="btn primary blue" disabled={startingSnapshot} onClick={this.startManualSnapshot}>{startingSnapshot ? "Starting a snapshot..." : "Start a snapshot"}</button>
            </div>
          </div>
          {snapshots?.listSnapshots && snapshots?.listSnapshots?.map((snapshot) => (
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
  graphql(listSnapshots, {
    name: "snapshots",
    options: ({ match }) => {
      const slug = match.params.slug;
      return {
        variables: { slug },
        fetchPolicy: "no-cache"
      }
    }
  }),
  graphql(snapshotSettings, {
    name: "snapshotSettings",
    options: ({ match }) => {
      const slug = match.params.slug;
      return {
        variables: { slug },
        fetchPolicy: "no-cache"
      }
    }
  }),
  graphql(manualSnapshot, {
    props: ({ mutate }) => ({
      manualSnapshot: (appId) => mutate({ variables: { appId } })
    })
  }),
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
