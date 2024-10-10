import { Component } from "react";
import Select from "react-select";
import { withRouter } from "@src/utilities/react-router-utilities";
import {
  getCronFrequency,
  getCronInterval,
  getReadableCronDescriptor,
  Utilities,
} from "../../utilities/utilities";
import ErrorModal from "../modals/ErrorModal";
import Loader from "../shared/Loader";
import find from "lodash/find";
import isEmpty from "lodash/isEmpty";

import GettingStartedSnapshots from "./GettingStartedSnapshots";
import "../../scss/components/shared/SnapshotForm.scss";
import Icon from "../Icon";

const SCHEDULES = [
  {
    value: "hourly",
    label: "Hourly",
  },
  {
    value: "daily",
    label: "Daily",
  },
  {
    value: "weekly",
    label: "Weekly",
  },
  {
    value: "custom",
    label: "Custom",
  },
];

const RETENTION_UNITS = [
  {
    value: "days",
    label: "Days",
  },
  {
    value: "weeks",
    label: "Weeks",
  },
  {
    value: "months",
    label: "Months",
  },
  {
    value: "years",
    label: "Years",
  },
];

class SnapshotSchedule extends Component {
  constructor(props) {
    super();
    this.state = {
      retentionInput: "",
      autoEnabled: false,
      selectedSchedule: {},
      selectedRetentionUnit: {},
      frequency: "",
      updatingSchedule: false,
      updateConfirm: false,
      displayErrorModal: false,
      gettingConfigErrMsg: "",
      snapshotConfig: {},
      updateScheduleErrMsg: "",
      selectedApp: {},
      activeTab: !isEmpty(props.location.search) ? "partial" : "full",
    };
  }

  setFields = () => {
    const { snapshotConfig } = this.state;
    if (snapshotConfig) {
      this.setState(
        {
          autoEnabled: snapshotConfig.autoEnabled,
          retentionInput: snapshotConfig.ttl.inputValue,
          selectedRetentionUnit: find(RETENTION_UNITS, [
            "value",
            snapshotConfig.ttl.inputTimeUnit,
          ]),
          selectedSchedule: find(SCHEDULES, [
            "value",
            getCronInterval(snapshotConfig.autoSchedule.schedule),
          ]),
          frequency: snapshotConfig.autoSchedule.schedule,
        },
        () => this.getReadableCronExpression()
      );
    } else {
      this.setState(
        {
          retentionInput: "4",
          selectedRetentionUnit: find(RETENTION_UNITS, ["value", "weeks"]),
          selectedSchedule: find(SCHEDULES, ["value", "weekly"]),
          frequency: "0 0 * * MON",
        },
        () => this.getReadableCronExpression()
      );
    }
  };

  handleFormChange = (field, e) => {
    let nextState = {};
    if (field === "autoEnabled") {
      nextState[field] = e.target.checked;
    } else {
      nextState[field] = e.target.value;
    }
    this.setState(nextState);
  };

  getReadableCronExpression = () => {
    const { frequency } = this.state;
    try {
      const readable = getReadableCronDescriptor(frequency);
      if (readable.includes("undefined")) {
        this.setState({ hasValidCron: false });
      } else {
        this.setState({ humanReadableCron: readable, hasValidCron: true });
      }
    } catch {
      this.setState({ hasValidCron: false });
    }
  };

  handleScheduleChange = (selectedSchedule) => {
    this.setState(
      {
        selectedSchedule: selectedSchedule,
        frequency:
          selectedSchedule.value === "custom"
            ? this.state.frequency
            : getCronFrequency(selectedSchedule.value),
      },
      () => {
        this.getReadableCronExpression();
      }
    );
  };

  handleCronChange = (e) => {
    const schedule = find(SCHEDULES, { value: e.target.value });
    const selectedSchedule = schedule
      ? schedule
      : find(SCHEDULES, { value: "custom" });
    this.setState({ frequency: e.target.value, selectedSchedule }, () => {
      this.getReadableCronExpression();
    });
  };

  handleRetentionUnitChange = (retentionUnit) => {
    this.setState({ selectedRetentionUnit: retentionUnit });
  };

  getSnapshotConfig = async (currentApp) => {
    this.setState({
      loadingConfig: true,
      gettingConfigErrMsg: "",
      displayErrorModal: false,
    });
    const url = currentApp
      ? `${process.env.API_ENDPOINT}/app/${currentApp.slug}/snapshot/config`
      : `${process.env.API_ENDPOINT}/snapshot/config`;
    try {
      const res = await fetch(url, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
      });
      if (!res.ok) {
        this.setState({
          loadingConfig: false,
          gettingConfigErrMsg: `Unable to get snapshot config: Unexpected status code: ${res.status}`,
          displayErrorModal: true,
        });
        return;
      }
      const body = await res.json();
      this.setState({
        snapshotConfig: body,
        loadingConfig: false,
      });
    } catch (err) {
      console.log(err);
      this.setState({
        loadingConfig: false,
        gettingConfigErrMsg: err
          ? err.message
          : "Something went wrong, please try again.",
        displayErrorModal: true,
      });
    }
  };

  checkIsAppConfig = () => {
    if (!isEmpty(this.props.location.search)) {
      return true;
    } else {
      return false;
    }
  };

  settingSnapshotConfig = (currentApp) => {
    this.getSnapshotConfig(currentApp);
    this.getReadableCronExpression();
  };

  componentDidMount = () => {
    if (!isEmpty(this.props.apps) && this.props.location.search) {
      const currentApp = this.props.apps.find(
        (app) => app.slug === this.props.location.search.slice(1)
      );
      this.setState({ selectedApp: currentApp }, () => {
        this.settingSnapshotConfig(currentApp);
      });
    } else {
      this.settingSnapshotConfig();
    }
  };

  componentDidUpdate = (lastProps, lastState) => {
    if (
      this.state.snapshotConfig &&
      this.state.snapshotConfig !== lastState.snapshotConfig
    ) {
      this.setFields();
    }

    if (this.state.activeTab !== lastState.activeTab && this.state.activeTab) {
      if (this.state.activeTab === "full") {
        this.settingSnapshotConfig();
        this.props.navigate("/snapshots/settings", {
          replace: true,
        });
      } else {
        if (!isEmpty(this.props.apps) && this.props.location.search) {
          const currentApp = this.props.apps.find(
            (app) => app.slug === this.props.location.search.slice(1)
          );
          this.setState({ selectedApp: currentApp }, () => {
            this.settingSnapshotConfig(currentApp);
          });
          this.props.navigate(`/snapshots/settings?${currentApp.slug}`, {
            replace: true,
          });
        } else if (!isEmpty(this.props.apps)) {
          this.setState({ selectedApp: this.props.apps[0] }, () => {
            this.settingSnapshotConfig(this.props.apps[0]);
          });
          this.props.navigate(
            `/snapshots/settings?${this.props.apps[0].slug}`,
            {
              replace: true,
            }
          );
        }
      }
    }

    if (
      this.state.selectedApp !== lastState.selectedApp &&
      this.state.selectedApp
    ) {
      this.settingSnapshotConfig(this.state.selectedApp);
      this.props.navigate(
        `/snapshots/settings?${this.state.selectedApp.slug}`,
        {
          replace: true,
        }
      );
    }
  };

  saveSnapshotConfig = () => {
    const isAppConfig = this.checkIsAppConfig();

    this.setState({ updatingSchedule: true });
    let body;
    let url;
    if (isAppConfig) {
      body = {
        appId: this.state.selectedApp.id,
        schedule: this.state.frequency,
        autoEnabled: this.state.autoEnabled,
      };
      url = `${process.env.API_ENDPOINT}/app/${this.state.selectedApp.slug}/snapshot/schedule`;
    } else {
      body = {
        schedule: this.state.frequency,
        autoEnabled: this.state.autoEnabled,
      };
      url = `${process.env.API_ENDPOINT}/snapshot/schedule`;
    }
    fetch(url, {
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      credentials: "include",
      method: "PUT",
      body: JSON.stringify(body),
    })
      .then(async (res) => {
        if (!res.ok && res.status === 409) {
          const result = await res.json();
          if (result.kotsadmRequiresVeleroAccess) {
            this.setState({
              updatingSchedule: false,
            });
            this.props.openConfigureSnapshotsMinimalRBACModal(
              result.kotsadmRequiresVeleroAccess,
              result.kotsadmNamespace
            );
            return;
          }
        }

        const data = await res.json();
        if (!res.ok || !data.success) {
          if (res.status === 401) {
            Utilities.logoutUser();
            return;
          }
          this.setState({
            updateScheduleErrMsg:
              data.error || "Failed to save snapshot schedule",
            messageType: "error",
            updatingSchedule: false,
          });
          return;
        }

        this.setState({
          updatingSchedule: false,
          updateConfirm: true,
          updateScheduleErrMsg: " ",
        });

        if (this.confirmTimeout) {
          clearTimeout(this.confirmTimeout);
        }
        this.confirmTimeout = setTimeout(() => {
          this.setState({ updateConfirm: false });
        }, 5000);
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          updateScheduleErrMsg: err ? err.message : "Failed to connect to API",
          messageType: "error",
          updatingSchedule: false,
        });
      });
  };

  saveRetentionConfig = () => {
    const isAppConfig = this.checkIsAppConfig();

    this.setState({ updatingRetention: true });
    let body;
    let url;
    if (isAppConfig) {
      body = {
        appId: this.state.selectedApp.id,
        inputValue: this.state.retentionInput,
        inputTimeUnit: this.state.selectedRetentionUnit?.value,
      };
      url = `${process.env.API_ENDPOINT}/app/${this.state.selectedApp.slug}/snapshot/retention`;
    } else {
      body = {
        inputValue: this.state.retentionInput,
        inputTimeUnit: this.state.selectedRetentionUnit?.value,
      };
      url = `${process.env.API_ENDPOINT}/snapshot/retention`;
    }
    fetch(url, {
      headers: {
        "Content-Type": "application/json",
        Accept: "application/json",
      },
      credentials: "include",
      method: "PUT",
      body: JSON.stringify(body),
    })
      .then(async (res) => {
        if (!res.ok && res.status === 409) {
          const result = await res.json();
          if (result.kotsadmRequiresVeleroAccess) {
            this.setState({
              updatingRetention: false,
            });
            this.props.openConfigureSnapshotsMinimalRBACModal(
              result.kotsadmRequiresVeleroAccess,
              result.kotsadmNamespace
            );
            return;
          }
        }

        const data = await res.json();
        if (!res.ok || !data.success) {
          if (res.status === 401) {
            Utilities.logoutUser();
            return;
          }
          this.setState({
            updateRetentionErrMsg:
              data.error || "Failed to save snapshot retention",
            messageType: "error",
            updatingRetention: false,
          });
          return;
        }

        this.setState({
          updatingRetention: false,
          updateRetentionConfirm: true,
          updateRetentionErrMsg: " ",
        });

        if (this.confirmTimeout) {
          clearTimeout(this.confirmTimeout);
        }
        this.confirmTimeout = setTimeout(() => {
          this.setState({ updateRetentionConfirm: false });
        }, 5000);
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          updateRetentionErrMsg: err ? err.message : "Failed to connect to API",
          messageType: "error",
          updatingRetention: false,
        });
      });
  };

  toggleErrorModal = () => {
    this.setState({ displayErrorModal: !this.state.displayErrorModal });
  };

  toggleScheduleAction = (active) => {
    this.setState({
      activeTab: active,
    });
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

  render() {
    const { isVeleroInstalled, updatingSettings, isEmbeddedCluster } =
      this.props;
    const {
      hasValidCron,
      updatingSchedule,
      updatingRetention,
      updateConfirm,
      updateRetentionConfirm,
      loadingConfig,
      updateScheduleErrMsg,
      updateRetentionErrMsg,
    } = this.state;
    const selectedRetentionUnit = RETENTION_UNITS.find((ru) => {
      return ru.value === this.state.selectedRetentionUnit?.value;
    });
    const selectedSchedule = SCHEDULES.find((schedule) => {
      return schedule.value === this.state.selectedSchedule?.value;
    });
    const isAppConfig = this.checkIsAppConfig();

    if (loadingConfig) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    const isSettingsPage = window.location.pathname.includes(
      "/snapshots/settings"
    );

    let featureName = "snapshot";
    if (isEmbeddedCluster) {
      featureName = "backup";
    }

    return (
      <div className="flex-auto">
        <div className="flex flex-column">
          {!isAppConfig && !this.props.isVeleroRunning && !updatingSettings && (
            <div className="Info--wrapper card-bg flex flex1 u-marginBottom--15">
              <Icon
                icon="info"
                className={"tw-mt-2 tw-mr-2 flex-auto"}
                size={18}
              />
              <div className="flex flex-column u-marginLeft--5">
                <p className="u-fontSize--normal u-fontWeight--bold u-lineHeight--normal u-textColor--primary">
                  {" "}
                  Scheduling not active{" "}
                </p>
                <span className="u-fontSize--small u-fontWeight--normal u-lineHeight--normal u-textColor--bodyCopy">
                  {" "}
                  Schedules will not take affect until Velero is running and a
                  storage destination has been configured.
                </span>
              </div>
            </div>
          )}
          <div className="flex flex-column snapshot-form-wrapper card-bg u-padding--15">
            <p className="card-title">Scheduled {featureName}s</p>
            <div className="u-marginBottom--10">
              <p className="u-fontSize--normal u-fontWeight--normal u-lineHeight--normal u-textColor--bodyCopy u-marginTop--12 schedule">
                Configure a schedule for {featureName}s of the Admin Console and
                all application data.
              </p>
            </div>
            {!isEmbeddedCluster && (
              <div className="SnapshotScheduleTabs--wrapper flex1 flex-column">
                <div className="tab-items flex justifyContent--spaceBetween">
                  <span
                    className={`${
                      this.state.activeTab === "full" ? "is-active" : ""
                    } tab-item blue`}
                    onClick={() => this.toggleScheduleAction("full")}
                  >
                    Full snapshots (Instance)
                  </span>
                  <span
                    className={`${
                      this.state.activeTab === "partial" ? "is-active" : ""
                    } tab-item blue`}
                    onClick={() => this.toggleScheduleAction("partial")}
                  >
                    Partial snapshots (Application)
                  </span>
                </div>
              </div>
            )}
            {this.state.activeTab === "partial" && (
              <div className="flex u-marginTop--12 u-marginBottom--15">
                <Select
                  className="replicated-select-container u-width--full"
                  classNamePrefix="replicated-select"
                  options={this.props.apps}
                  getOptionLabel={this.getLabel}
                  getOptionValue={(app) => app.name}
                  value={this.state.selectedApp}
                  onChange={this.onAppChange}
                  isOptionSelected={(app) => {
                    app.name === this.state.selectedApp?.name;
                  }}
                />
              </div>
            )}
            <div
              className={`flex-column card-item u-padding--15 ${
                !isAppConfig ? "u-marginTop--12" : "u-marginBottom--20"
              }`}
            >
              <div className=" flex1 u-marginBottom--15">
                <div className="BoxedCheckbox-wrapper flex1 u-textAlign--left">
                  <div
                    className={`flex-auto flex alignItems--center ${
                      this.state.autoEnabled ? "is-active" : ""
                    }`}
                  >
                    <input
                      type="checkbox"
                      className="u-cursor--pointer"
                      id="autoEnabled"
                      checked={this.state.autoEnabled}
                      onChange={(e) => {
                        this.handleFormChange("autoEnabled", e);
                      }}
                    />
                    <label
                      htmlFor="autoEnabled"
                      className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none"
                    >
                      <div className="flex1">
                        <p className="u-textColor--primary u-fontSize--normal u-fontWeight--medium u-marginLeft--5">
                          Enable scheduled {featureName}s
                        </p>
                      </div>
                    </label>
                  </div>
                </div>
              </div>
              {this.state.autoEnabled && (
                <div className="flex-column flex1 u-position--relative u-marginBottom--40">
                  <div className="flex flex1">
                    <div className="flex1 u-paddingRight--5">
                      <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                        Schedule
                      </p>
                      <Select
                        className="replicated-select-container"
                        classNamePrefix="replicated-select"
                        placeholder="Select an interval"
                        options={SCHEDULES}
                        isSearchable={false}
                        getOptionValue={(schedule) => schedule.label}
                        value={selectedSchedule}
                        onChange={this.handleScheduleChange}
                        isOptionSelected={(option) => {
                          option.value === selectedSchedule;
                        }}
                      />
                    </div>
                    <div className="flex1 u-paddingLeft--5">
                      <p className="u-fontSize--normal card-item-title u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                        Cron expression
                      </p>
                      <input
                        type="text"
                        className="Input"
                        placeholder="0 0 * * MON"
                        value={this.state.frequency}
                        onChange={(e) => this.handleCronChange(e)}
                      />
                    </div>
                  </div>
                  {hasValidCron ? (
                    <p className="cron-expression-text">
                      {this.state.humanReadableCron}
                    </p>
                  ) : (
                    <p className="cron-expression-text">
                      Enter a valid Cron Expression{" "}
                      <a
                        className="link"
                        href=""
                        target="_blank"
                        rel="noopener noreferrer"
                      >
                        Get help
                      </a>
                    </p>
                  )}
                </div>
              )}
              <div className="flex">
                <button
                  className="btn primary blue"
                  disabled={updatingSchedule}
                  onClick={this.saveSnapshotConfig}
                >
                  {updatingSchedule ? "Updating schedule" : "Update schedule"}
                </button>
                {updateConfirm && (
                  <div className="u-marginLeft--10 flex alignItems--center">
                    <Icon
                      icon="check-circle-filled"
                      size={16}
                      className="success-color"
                    />
                    <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-textColor--success">
                      Schedule updated
                    </span>
                  </div>
                )}
                {updateScheduleErrMsg && (
                  <div className="u-marginLeft--10 flex alignItems--center">
                    <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-textColor--error">
                      {updateScheduleErrMsg}
                    </span>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
        <div> &nbsp; </div>
        {/*start of retention box*/}
        <div className="flex flex-column">
          <div className="flex flex-column snapshot-form-wrapper card-bg u-padding--15">
            <p className="card-title">Retention policy</p>
            <div className="u-marginBottom--10">
              <p className="u-fontSize--normal u-fontWeight--normal u-lineHeight--normal u-textColor--bodyCopy u-marginTop--12 retention">
                Configure the retention policy for {featureName}s of the admin
                console and all application data. This applies to both manual
                and scheduled {featureName}s.
              </p>
            </div>
            <div
              className={`flex-column card-item u-padding--15 ${
                !isAppConfig ? "u-marginTop--12" : "u-marginBottom--20"
              }`}
            >
              <div>
                <p className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--normal u-lineHeight--normal u-marginBottom--10">
                  Choose how long to retain {featureName}s before they are
                  automatically deleted.
                </p>
                <div className="flex u-marginBottom--20">
                  <div className="flex-auto u-paddingRight--5">
                    <input
                      type="text"
                      className="Input"
                      placeholder="4"
                      value={this.state.retentionInput}
                      onChange={(e) => {
                        this.handleFormChange("retentionInput", e);
                      }}
                    />
                  </div>
                  <div className="flex1 u-paddingLeft--5">
                    <Select
                      className="replicated-select-container"
                      classNamePrefix="replicated-select"
                      placeholder="Select unit"
                      options={RETENTION_UNITS}
                      isSearchable={false}
                      getOptionValue={(retentionUnit) => retentionUnit.label}
                      value={selectedRetentionUnit}
                      onChange={this.handleRetentionUnitChange}
                      isOptionSelected={(option) => {
                        option.value === selectedRetentionUnit;
                      }}
                    />
                  </div>
                </div>
              </div>
              <div className="flex">
                <button
                  className="btn primary blue"
                  disabled={updatingRetention}
                  onClick={this.saveRetentionConfig}
                >
                  {updatingRetention
                    ? "Updating retention policy"
                    : "Update retention policy"}
                </button>
                {updateRetentionConfirm && (
                  <div className="u-marginLeft--10 flex alignItems--center">
                    <Icon
                      icon="check-circle-filled"
                      size={16}
                      className="success-color"
                    />
                    <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-textColor--success">
                      Retention updated
                    </span>
                  </div>
                )}
                {updateRetentionErrMsg && (
                  <div className="u-marginLeft--10 flex alignItems--center">
                    <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-textColor--error">
                      {updateRetentionErrMsg}
                    </span>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
        <ErrorModal
          errorModal={this.state.displayErrorModal}
          toggleErrorModal={this.toggleErrorModal}
          errMsg={this.state.gettingConfigErrMsg}
          tryAgain={undefined}
          err="Failed to get snapshot schedule settings"
          loading={loadingConfig}
        />
        {!isAppConfig && !isSettingsPage && (
          <GettingStartedSnapshots isVeleroInstalled={isVeleroInstalled} />
        )}
      </div>
    );
  }
}

export default withRouter(SnapshotSchedule);
