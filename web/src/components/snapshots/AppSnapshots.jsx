import React, { Component } from "react";
import { withRouter } from "@src/utilities/react-router-utilities";
import { Link } from "react-router-dom";
import { KotsPageTitle } from "@components/Head";
import Modal from "react-modal";
import ReactTooltip from "react-tooltip";
import Select from "react-select";
import isEmpty from "lodash/isEmpty";

import SnapshotRow from "./SnapshotRow";
import GettingStartedSnapshots from "./GettingStartedSnapshots";
import ScheduleSnapshotForm from "../shared/ScheduleSnapshotForm";
import Loader from "../shared/Loader";
import DeleteSnapshotModal from "../modals/DeleteSnapshotModal";
import RestoreSnapshotModal from "../modals/RestoreSnapshotModal";
import SnapshotDifferencesModal from "../modals/SnapshotDifferencesModal";
import ErrorModal from "../modals/ErrorModal";

import "../../scss/components/snapshots/AppSnapshots.scss";
import { isVeleroCorrectVersion, Utilities } from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";
import Icon from "../Icon";

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
    displayErrorModal: false,
    selectedApp: {},
    switchingApps: false,
    snapshotDifferencesModal: false,
  };

  componentDidMount = async () => {
    if (!isEmpty(this.props.app)) {
      this.setState({ selectedApp: this.props.app });
    }

    await this.fetchSnapshotSettings();

    this.checkRestoreInProgress();
    this.state.listSnapshotsJob.start(this.listSnapshots, 2000);
  };

  componentWillUnmount() {
    this.state.listSnapshotsJob.stop();
  }

  componentDidUpdate(lastProps, lastState) {
    const { snapshots, networkErr, selectedApp } = this.state;

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

    if (selectedApp !== lastState.selectedApp && selectedApp) {
      this.setState({ switchingApps: true });
      setTimeout(() => {
        this.setState({
          switchingApps: false,
        });
      }, 3000);
      this.checkRestoreInProgress();
      this.props.history.replace(`/snapshots/partial/${selectedApp.slug}`);
    }
  }

  checkRestoreInProgress() {
    const { selectedApp } = this.state;
    fetch(
      `${process.env.API_ENDPOINT}/app/${selectedApp.slug}/snapshot/restore/status`,
      {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
      }
    )
      .then(async (result) => {
        const body = await result.json();
        if (body.error) {
          this.setState({
            restoreInProgressErr: true,
            restoreInProgressMsg: body.error,
          });
        } else if (body.status == "running") {
          this.props.history.replace(
            `/snapshots/partial/${selectedApp.slug}/${body.restore_name}/restore`
          );
        } else {
          this.state.listSnapshotsJob.start(this.listSnapshots, 2000);
        }
      })
      .catch((err) => {
        this.setState({
          restoreInProgressErr: true,
          restoreInProgressMsg: err,
        });
      });
  }

  listSnapshots = async () => {
    const { selectedApp } = this.state;
    this.setState({
      snapshotsListErr: false,
      snapshotsListErrMsg: "",
      networkErr: false,
      displayErrorModal: false,
    });
    try {
      const res = await fetch(
        `${process.env.API_ENDPOINT}/app/${selectedApp.slug}/snapshots`,
        {
          method: "GET",
          headers: {
            "Content-Type": "application/json",
          },
          credentials: "include",
        }
      );
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }
        this.setState({
          snapshotsListErr: true,
          snapshotsListErrMsg: `Unexpected status code: ${res.status}`,
          networkErr: false,
          displayErrorModal: true,
        });
        return;
      }
      const response = await res.json();

      this.setState({
        snapshots: response.backups?.sort((a, b) =>
          b.startedAt
            ? new Date(b.startedAt) - new Date(a.startedAt)
            : -99999999
        ),
        hasSnapshotsLoaded: true,
        snapshotsListErr: false,
        snapshotsListErrMsg: "",
        networkErr: false,
        displayErrorModal: false,
      });
    } catch (err) {
      this.setState({
        snapshotsListErr: true,
        snapshotsListErrMsg: err.message
          ? err.message
          : "There was an error while showing the snapshots. Please try again",
        networkErr: true,
        displayErrorModal: true,
      });
    }
  };

  fetchSnapshotSettings = async () => {
    this.setState({
      isLoadingSnapshotSettings: true,
      snapshotSettingsErr: false,
      snapshotSettingsErrMsg: "",
    });

    fetch(`${process.env.API_ENDPOINT}/snapshots/settings`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
    })
      .then(async (res) => {
        if (!res.ok && res.status === 409) {
          const result = await res.json();
          if (result.kotsadmRequiresVeleroAccess) {
            this.setState({
              isLoadingSnapshotSettings: false,
            });
            this.props.history.replace("/snapshots/settings");
            return;
          }
        }

        const result = await res.json();

        if (result?.isVeleroRunning && result?.isNodeAgentRunning) {
          if (!result?.store) {
            // velero and node-agent are running but a backup storage location is not configured yet
            this.props.history.replace("/snapshots/settings");
          } else {
            this.state.listSnapshotsJob.start(this.listInstanceSnapshots, 2000);
          }
        } else {
          this.props.history.push("/snapshots/settings?configure=true");
        }
        this.setState({
          snapshotSettings: result,
          isLoadingSnapshotSettings: false,
          snapshotSettingsErr: false,
          snapshotSettingsErrMsg: "",
        });
      })
      .catch((err) => {
        this.setState({
          isLoadingSnapshotSettings: false,
          snapshotSettingsErr: true,
          snapshotSettingsErrMsg: err,
        });
      });
  };

  toggleScheduleSnapshotModal = () => {
    this.setState({
      displayScheduleSnapshotModal: !this.state.displayScheduleSnapshotModal,
    });
  };

  toggleConfirmDeleteModal = (snapshot) => {
    if (this.state.deleteSnapshotModal) {
      this.setState({
        deleteSnapshotModal: false,
        snapshotToDelete: "",
        deleteErr: false,
        deleteErrorMsg: "",
      });
    } else {
      this.setState({
        deleteSnapshotModal: true,
        snapshotToDelete: snapshot,
        deleteErr: false,
        deleteErrorMsg: "",
      });
    }
  };

  toggleRestoreModal = (snapshot) => {
    if (this.state.restoreSnapshotModal) {
      this.setState({
        restoreSnapshotModal: false,
        snapshotToRestore: "",
        restoreErr: false,
        restoreErrorMsg: "",
      });
    } else {
      this.setState({
        restoreSnapshotModal: true,
        snapshotToRestore: snapshot,
        restoreErr: false,
        restoreErrorMsg: "",
      });
    }
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  };

  handleDeleteSnapshot = (snapshot) => {
    const fakeDeletionSnapshot = {
      name: "Preparing for snapshot deletion",
      status: "Deleting",
      trigger: "manual",
      appID: this.state.selectedApp.id,
      sequence: snapshot.sequence,
      startedAt: snapshot.startedAt,
      finishedAt: snapshot.finishedAt,
      expiresAt: snapshot.expiresAt,
      volumeCount: snapshot.volumeCount,
      volumeSuccessCount: snapshot.volumeSuccessCount,
      volumeBytes: 0,
      volumeSizeHuman: snapshot.volumeSizeHuman,
    };

    this.setState({
      deletingSnapshot: true,
      deleteErr: false,
      deleteErrorMsg: "",
      snapshots: this.state.snapshots.map((s) =>
        s === snapshot ? fakeDeletionSnapshot : s
      ),
    });

    fetch(`${process.env.API_ENDPOINT}/snapshot/${snapshot.name}/delete`, {
      method: "POST",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
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
          snapshotToDelete: "",
        });
      })
      .catch((err) => {
        this.setState({
          deletingSnapshot: false,
          deleteErr: true,
          deleteErrorMsg: err
            ? err.message
            : "Something went wrong, please try again.",
        });
      });
  };

  handleRestoreSnapshot = (snapshot) => {
    const { selectedApp } = this.state;

    if (this.state.appSlugToRestore !== selectedApp?.slug) {
      this.setState({ appSlugMismatch: true });
      return;
    }

    this.setState({
      restoringSnapshot: true,
      restoreErr: false,
      restoreErrorMsg: "",
    });

    fetch(
      `${process.env.API_ENDPOINT}/app/${selectedApp?.slug}/snapshot/restore/${snapshot.name}`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
      }
    )
      .then(async (result) => {
        if (result.ok) {
          this.setState({
            restoringSnapshot: true,
            restoreSnapshotModal: false,
            restoreErr: false,
            restoreErrorMsg: "",
          });

          this.props.history.replace(
            `/snapshots/partial/${selectedApp.slug}/${snapshot.name}/restore`
          );
        } else {
          const body = await result.json();
          this.setState({
            restoringSnapshot: false,
            restoreErr: true,
            restoreErrorMsg: body.error,
          });
        }
      })
      .catch((err) => {
        this.setState({
          restoringSnapshot: false,
          restoreErr: true,
          restoreErrorMsg: err,
        });
      });
  };

  startManualSnapshot = () => {
    const { selectedApp } = this.state;

    const fakeProgressSnapshot = {
      name: "Preparing snapshot",
      status: "InProgress",
      trigger: "manual",
      appID: selectedApp.id,
      sequence: "",
      startedAt: new Date().toISOString(),
      finishedAt: "",
      expiresAt: "",
      volumeCount: 0,
      volumeSuccessCount: 0,
      volumeBytes: 0,
      volumeSizeHuman: "",
    };

    this.setState({
      startingSnapshot: true,
      startSnapshotErr: false,
      startSnapshotErrorMsg: "",
      isStartButtonClicked: true,
      snapshots: [...this.state.snapshots, fakeProgressSnapshot].sort(
        (a, b) => new Date(b.startedAt) - new Date(a.startedAt)
      ),
    });

    fetch(
      `${process.env.API_ENDPOINT}/app/${selectedApp.slug}/snapshot/backup`,
      {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
      }
    )
      .then(async (result) => {
        if (!result.ok && result.status === 409) {
          const res = await result.json();
          if (res.kotsadmRequiresVeleroAccess) {
            this.setState({
              startingSnapshot: false,
            });
            this.props.history.replace("/snapshots/settings");
            return;
          }
        }

        if (result.ok) {
          this.setState({
            startingSnapshot: false,
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
      .catch((err) => {
        this.setState({
          startingSnapshot: false,
          startSnapshotErr: true,
          startSnapshotErrorMsg: err,
        });
      });
  };

  handleApplicationSlugChange = (e) => {
    if (this.state.appSlugMismatch) {
      this.setState({ appSlugMismatch: false });
    }
    this.setState({ appSlugToRestore: e.target.value });
  };

  onAppChange = (selectedApp) => {
    this.setState({ selectedApp });
  };

  getLabel = ({ iconUri, name }) => {
    return (
      <div style={{ alignItems: "center", display: "flex" }}>
        <span
          className="app-icon"
          style={{
            fontSize: 18,
            marginRight: "0.5em",
            backgroundImage: `url(${iconUri})`,
          }}
        ></span>
        <span style={{ fontSize: 14 }}>{name}</span>
      </div>
    );
  };

  toggleSnaphotDifferencesModal = () => {
    this.setState({
      snapshotDifferencesModal: !this.state.snapshotDifferencesModal,
    });
  };

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
      restoreInProgressErrMsg,
      selectedApp,
      switchingApps,
    } = this.state;
    const { app, appsList } = this.props;
    const appTitle = app?.name;
    const inProgressSnapshotExist = snapshots?.find(
      (snapshot) => snapshot.status === "InProgress"
    );

    if (
      isLoadingSnapshotSettings ||
      !hasSnapshotsLoaded ||
      (isStartButtonClicked && snapshots?.length === 0) ||
      startingSnapshot ||
      switchingApps
    ) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    if (!snapshotSettings?.store) {
      this.props.history.replace("/snapshots");
    }

    if (restoreInProgressErr) {
      return (
        <div className="flex1 flex-column justifyContent--center alignItems--center">
          <span className="icon redWarningIcon" />
          <p className="u-textColor--error u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-marginTop--10">
            {restoreInProgressErrMsg}
          </p>
        </div>
      );
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
          appSlug={app?.slug}
        />
      );
    }

    return (
      <div className="flex1 flex-column u-overflow--auto">
        <KotsPageTitle pageName="Partial Snapshots" showAppSlug />
        {!isVeleroCorrectVersion(snapshotSettings) ? (
          <div className="VeleroWarningBlock">
            <Icon icon={"warning"} size={24} className="warning-color" />
            <p>
              {" "}
              To use snapshots reliably, install Velero version 1.5.1 or greater
            </p>
          </div>
        ) : null}
        <div
          className="centered-container flex-column flex1 u-paddingTop--30 u-paddingBottom--20 alignItems--center"
          style={{ maxWidth: "770px" }}
        >
          <div className="InfoSnapshots--wrapper flex flex-auto u-marginBottom--20">
            <Icon icon="info" className="tw-mr-2" size={22} />
            <p className="u-fontSize--small u-fontWeight--normal u-lineHeight--normal u-textColor--accent">
              It’s recommend that you use{" "}
              <Link to="/snapshots" className="link u-fontSize--small">
                Full snapshots (Instance){" "}
              </Link>{" "}
              in lieu of Partial snapshots (Application), given Full snapshots
              offers the same restoration capabilities.
              <span
                className="link"
                onClick={this.toggleSnaphotDifferencesModal}
              >
                Learn&nbsp;more
              </span>
              .
            </p>
          </div>
          <div className="AppSnapshots--wrapper card-bg flex-column u-marginTop--20">
            <div className="flex flex-column u-marginBottom--15">
              <div className="flex justifyContent--spaceBetween">
                <p className="u-fontWeight--bold card-title u-fontSize--larger u-lineHeight--normal">
                  {" "}
                  Partial snapshots (Application){" "}
                </p>
                <div className="flex alignSelf--center">
                  <Link
                    to={`/snapshots/settings?${selectedApp.slug}`}
                    className="link u-fontSize--small u-fontWeight--bold flex alignItems--center"
                  >
                    <Icon
                      icon="settings-gear-outline"
                      size={18}
                      className="u-marginRight--5"
                    />
                    Settings
                  </Link>
                  {snapshots?.length > 0 &&
                    snapshotSettings?.veleroVersion !== "" && (
                      <span
                        data-for="startSnapshotBtn"
                        data-tip="startSnapshotBtn"
                        data-tip-disable={false}
                      >
                        <button
                          className="btn primary blue u-marginLeft--20"
                          disabled={startingSnapshot || inProgressSnapshotExist}
                          onClick={this.startManualSnapshot}
                        >
                          {startingSnapshot
                            ? "Starting a snapshot..."
                            : "Start a snapshot"}
                        </button>
                      </span>
                    )}
                  {inProgressSnapshotExist && (
                    <ReactTooltip
                      id="startSnapshotBtn"
                      effect="solid"
                      className="replicated-tooltip"
                    >
                      <span>
                        You can't start a snapshot while another one is In
                        Progress
                      </span>
                    </ReactTooltip>
                  )}
                </div>
              </div>
              <p className="u-marginTop--10 u-fontSize--normal u-lineHeight--more u-fontWeight--medium u-textColor--bodyCopy">
                {" "}
                Partial snapshots (Application) only back up application volumes
                and application manifests; they do not back up the Admin
                Console.{" "}
              </p>
            </div>
            <div className="flex flex-auto u-marginBottom--15 alignItems--flexStart justifyContent--spaceBetween">
              <div className="flex">
                <Select
                  className="replicated-select-container app"
                  classNamePrefix="replicated-select"
                  options={appsList}
                  getOptionLabel={this.getLabel}
                  getOptionValue={(app) => app.name}
                  value={selectedApp}
                  onChange={this.onAppChange}
                  isOptionSelected={(app) => {
                    app.name === selectedApp.name;
                  }}
                />
              </div>
            </div>
            {startSnapshotErr ? (
              <div className="flex alignItems--center alignSelf--center u-marginBottom--10">
                <p className="u-textColor--error u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
                  {startSnapshotErrorMsg}
                </p>
              </div>
            ) : null}
            {snapshots?.length > 0 && snapshotSettings?.veleroVersion !== "" && (
              <div className="flex flex-column">
                {snapshots?.map((snapshot) => (
                  <SnapshotRow
                    key={`snapshot-${snapshot.name}-${snapshot.started}`}
                    snapshot={snapshot}
                    appSlug={selectedApp.slug}
                    toggleConfirmDeleteModal={this.toggleConfirmDeleteModal}
                    toggleRestoreModal={this.toggleRestoreModal}
                    app={selectedApp}
                  />
                ))}
              </div>
            )}
            {!isStartButtonClicked && snapshots?.length === 0 && (
              <div className="flex flex-column justifyContent--center alignItems--center">
                <GettingStartedSnapshots
                  isApp={true}
                  app={selectedApp}
                  isVeleroInstalled={snapshotSettings?.veleroVersion !== ""}
                  history={this.props.history}
                  startManualSnapshot={this.startManualSnapshot}
                />
              </div>
            )}
          </div>
          {displayScheduleSnapshotModal && (
            <Modal
              isOpen={displayScheduleSnapshotModal}
              onRequestClose={this.toggleScheduleSnapshotModal}
              shouldReturnFocusAfterClose={false}
              contentLabel="Schedule snapshot modal"
              ariaHideApp={false}
              className="ScheduleSnapshotModal--wrapper MediumSize Modal"
            >
              <div className="Modal-body">
                <ScheduleSnapshotForm onSubmit={this.handleScheduleSubmit} />
                <div className="u-marginTop--10 flex">
                  <button
                    onClick={this.toggleScheduleSnapshotModal}
                    className="btn secondary blue u-marginRight--10"
                  >
                    Cancel
                  </button>
                  <button
                    onClick={this.scheduleSnapshot}
                    className="btn primary blue"
                  >
                    Save
                  </button>
                </div>
              </div>
            </Modal>
          )}
          {deleteSnapshotModal && (
            <DeleteSnapshotModal
              deleteSnapshotModal={deleteSnapshotModal}
              toggleConfirmDeleteModal={this.toggleConfirmDeleteModal}
              handleDeleteSnapshot={this.handleDeleteSnapshot}
              snapshotToDelete={snapshotToDelete}
              deletingSnapshot={deletingSnapshot}
              deleteErr={deleteErr}
              deleteErrorMsg={deleteErrorMsg}
            />
          )}
          {restoreSnapshotModal && (
            <RestoreSnapshotModal
              restoreSnapshotModal={restoreSnapshotModal}
              toggleRestoreModal={this.toggleRestoreModal}
              handleRestoreSnapshot={this.handleRestoreSnapshot}
              snapshotToRestore={snapshotToRestore}
              restoringSnapshot={restoringSnapshot}
              restoreErr={restoreErr}
              restoreErrorMsg={restoreErrorMsg}
              app={selectedApp}
              appSlugToRestore={this.state.appSlugToRestore}
              appSlugMismatch={this.state.appSlugMismatch}
              handleApplicationSlugChange={this.handleApplicationSlugChange}
              apps={this.props.appsList}
            />
          )}
          {this.state.snapshotDifferencesModal && (
            <SnapshotDifferencesModal
              snapshotDifferencesModal={this.state.snapshotDifferencesModal}
              toggleSnapshotDifferencesModal={
                this.toggleSnaphotDifferencesModal
              }
            />
          )}
        </div>
      </div>
    );
  }
}

export default withRouter(AppSnapshots);
