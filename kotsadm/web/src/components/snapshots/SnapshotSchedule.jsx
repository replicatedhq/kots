import React, { Component } from "react";
import Select from "react-select";
import { Link, withRouter } from "react-router-dom"
import { Utilities, getCronFrequency, getCronInterval, getReadableCronDescriptor } from "../../utilities/utilities";
import ErrorModal from "../modals/ErrorModal";
import Loader from "../shared/Loader";
import find from "lodash/find";
import isEmpty from "lodash/isEmpty";
import "../../scss/components/shared/SnapshotForm.scss";

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
  }
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
  }
];

class SnapshotSchedule extends Component {
  state = {
    retentionInput: "",
    autoEnabled: false,
    selectedSchedule: {},
    selectedRetentionUnit: {},
    frequency: "",
    updatingSchedule: false,
    updateConfirm: false,
    displayErrorModal: false,
    gettingConfigErrMsg: "",
    snapshotConfig: {}
  };

  setFields = () => {
    const { snapshotConfig } = this.state;
    if (snapshotConfig) {
      this.setState({
        autoEnabled: snapshotConfig.autoEnabled,
        retentionInput: snapshotConfig.ttl.inputValue,
        selectedRetentionUnit: find(RETENTION_UNITS, ["value", snapshotConfig.ttl.inputTimeUnit]),
        selectedSchedule: find(SCHEDULES, ["value", getCronInterval(snapshotConfig.autoSchedule.schedule)]),
        frequency: snapshotConfig.autoSchedule.schedule,
      }, () => this.getReadableCronExpression());
    } else {
      this.setState({
        retentionInput: "4",
        selectedRetentionUnit: find(RETENTION_UNITS, ["value", "weeks"]),
        selectedSchedule: find(SCHEDULES, ["value", "weekly"]),
        frequency: "0 0 * * MON",
      }, () => this.getReadableCronExpression())
    }
  }

  handleFormChange = (field, e) => {
    let nextState = {};
    if (field === "autoEnabled") {
      nextState[field] = e.target.checked;
    } else {
      nextState[field] = e.target.value;
    }
    this.setState(nextState, () => {
      if (field === "frequency") {
        this.getReadableCronExpression();
      }
    });
  }

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
  }

  handleScheduleChange = (selectedSchedule) => {
    this.setState({
      selectedSchedule: selectedSchedule,
      frequency: selectedSchedule.value === "custom" ? this.state.frequency : getCronFrequency(selectedSchedule.value),
    }, () => {
      this.getReadableCronExpression();
    });
  }

  handleRetentionUnitChange = (retentionUnit) => {
    this.setState({ selectedRetentionUnit: retentionUnit });
  }

  getSnapshotConfig = async () => {
    const isAppConfig = this.checkIsAppConfig();

    this.setState({ loadingConfig: true, gettingConfigErrMsg: "", displayErrorModal: false });
    const url = isAppConfig ? `${window.env.API_ENDPOINT}/app/${this.props.app.slug}/snapshot/config` : `${window.env.API_ENDPOINT}/snapshot/config`;
    try {
      const res = await fetch(url, {
        method: "GET",
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        }
      });
      if (!res.ok) {
        this.setState({ loadingConfig: false, gettingConfigErrMsg: `Unable to get snapshot config: Unexpected status code: ${res.status}`, displayErrorModal: true });
        return;
      }
      const body = await res.json();
      this.setState({
        snapshotConfig: body,
        loadingConfig: false
      });

    } catch (err) {
      console.log(err);
      this.setState({ loadingConfig: false, gettingConfigErrMsg: err ? err.message : "Something went wrong, please try again.", displayErrorModal: true });
    }
  }

  componentDidUpdate = (lastProps, lastState) => {
    if (this.state.snapshotConfig && this.state.snapshotConfig !== lastState.snapshotConfig) {
      this.setFields();
    }
  }

  checkIsAppConfig = () => {
    if (!isEmpty(this.props.match.params)) {
      return true;
    } else {
      return false;
    }
  }

  componentDidMount = () => {
    if (!isEmpty(this.state.snapshotConfig)) {
      this.setFields();
    } else {
      this.getSnapshotConfig();
    }
    this.getReadableCronExpression();
  }

  saveSnapshotConfig = () => {
    const isAppConfig = this.checkIsAppConfig();

    this.setState({ updatingSchedule: true });
    let body;
    let url;
    if (isAppConfig) {
      body = {
        appId: this.props.app.id,
        inputValue: this.state.retentionInput,
        inputTimeUnit: this.state.selectedRetentionUnit?.value,
        schedule: this.state.frequency,
        autoEnabled: this.state.autoEnabled,
      };
      url = `${window.env.API_ENDPOINT}/app/${this.props.app.slug}/snapshot/config`
    } else {
      body = {
        inputValue: this.state.retentionInput,
        inputTimeUnit: this.state.selectedRetentionUnit?.value,
        schedule: this.state.frequency,
        autoEnabled: this.state.autoEnabled,
      };
      url = `${window.env.API_ENDPOINT}/snapshot/config`;
    }
    fetch(url, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
        "Accept": "application/json",
      },
      method: "PUT",
      body: JSON.stringify(body),
    })
      .then(async (res) => {
        const data = await res.json();
        if (!res.ok || !data.success) {
          if (res.status === 401) {
            Utilities.logoutUser();
            return;
          }
          this.setState({
            message: data.error || "Failed to save snapshot config",
            messageType: "error",
            updatingSchedule: false,
          })
          return
        }
        this.setState({ updatingSchedule: false, updateConfirm: true });
        setTimeout(() => {
          this.setState({ updateConfirm: false })
        }, 3000);
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          message: err ? err.message : "Failed to connect to API",
          messageType: "error",
          updatingSchedule: false,
        });
      })
  }

  render() {
    const { app } = this.props;
    const { hasValidCron, updatingSchedule, updateConfirm, loadingConfig } = this.state;
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
      )
    }

    return (
      <div className={`${isAppConfig ? "container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20 alignItems--center" : "flex-auto"}`}>
        <div className="flex flex-column">
          {!isAppConfig && !this.props.isVeleroRunning &&
            <div className="Info--wrapper flex flex1 u-marginBottom--15">
              <span className="icon info-icon flex u-marginTop--5" />
              <div className="flex flex-column u-marginLeft--5">
                <p className="u-fontSize--normal u-fontWeight--bold u-lineHeight--normal u-color--tuna"> Scheduling not active </p>
                <span className="u-fontSize--small u-fontWeight--normal u-lineHeight--normal u-color--dustyGray"> Schedules will not take affect until Velero is running and a storage destination has been configured.</span>
              </div>
            </div>}
          {isAppConfig &&
            <p className="u-marginBottom--30 u-fontSize--small u-color--tundora u-fontWeight--medium">
              <Link to={`/app/${app?.slug}/snapshots`} className="replicated-link">Snapshots</Link>
              <span className="u-color--dustyGray"> &gt; </span>
            Schedule
          </p>}
          <form className="flex flex-column snapshot-form-wrapper">
            {!isAppConfig && <p className="u-fontSize--normal u-color--tundora u-fontWeight--bold"> Scheduling</p>}
            <div className={`flex-column ${!isAppConfig ? "u-marginTop--12" : "u-marginBottom--20"}`}>
              <div className="flex1 u-marginBottom--20">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Automatic snapshots</p>
                <div className="BoxedCheckbox-wrapper flex1 u-textAlign--left">
                  <div className={`BoxedCheckbox flex-auto flex alignItems--center ${this.state.autoEnabled ? "is-active" : ""}`}>
                    <input
                      type="checkbox"
                      className="u-cursor--pointer u-marginLeft--10"
                      id="autoEnabled"
                      checked={this.state.autoEnabled}
                      onChange={(e) => { this.handleFormChange("autoEnabled", e) }}
                    />
                    <label htmlFor="autoEnabled" className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none">
                      <div className="flex1">
                        <p className="u-color--tuna u-fontSize--normal u-fontWeight--medium">Enable automatic scheduled snapshots</p>
                      </div>
                    </label>
                  </div>
                </div>
              </div>
              {this.state.autoEnabled &&
                <div className="flex-column flex1 u-position--relative u-marginBottom--50">
                  <div className="flex flex1">
                    <div className="flex1 u-paddingRight--5">
                      <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Schedule</p>
                      <Select
                        className="replicated-select-container"
                        classNamePrefix="replicated-select"
                        placeholder="Select an interval"
                        options={SCHEDULES}
                        isSearchable={false}
                        getOptionValue={(schedule) => schedule.label}
                        value={selectedSchedule}
                        onChange={this.handleScheduleChange}
                        isOptionSelected={(option) => { option.value === selectedSchedule }}
                      />
                    </div>
                    {this.state.selectedSchedule.value === "custom" &&
                      <div className="flex1 u-paddingLeft--5">
                        <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Cron expression</p>
                        <input type="text" className="Input" placeholder="0 0 * * MON" value={this.state.frequency} onChange={(e) => { this.handleFormChange("frequency", e) }} />
                      </div>
                    }
                  </div>
                  {hasValidCron ?
                    <p className="cron-expression-text">{this.state.humanReadableCron}</p>
                    :
                    <p className="cron-expression-text">Enter a valid Cron Expression <a className="replicated-link" href="" target="_blank" rel="noopener noreferrer">Get help</a></p>
                  }
                </div>
              }
              <div>
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Retention policy</p>
                <p className="u-fontSize--small u-color--dustyGray u-fontWeight--normal u-lineHeight--normal u-marginBottom--10">The Admin Console can reclaim space by automatically deleting older scheduled snapshots.</p>
                <p className="u-fontSize--small u-color--dustyGray u-fontWeight--normal u-lineHeight--normal u-marginBottom--10">Snapshots older than this will be deleted.</p>
                <div className="flex u-marginBottom--20">
                  <div className="flex-auto u-paddingRight--5">
                    <input type="text" className="Input" placeholder="4" value={this.state.retentionInput} onChange={(e) => { this.handleFormChange("retentionInput", e) }} />
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
                      isOptionSelected={(option) => { option.value === selectedRetentionUnit }}
                    />
                  </div>
                </div>
              </div>
              <div className="flex">
                <button className="btn primary blue" disabled={updatingSchedule} onClick={this.saveSnapshotConfig}>{updatingSchedule ? "Updating schedule" : "Update schedule"}</button>
                {updateConfirm &&
                  <div className="u-marginLeft--10 flex alignItems--center">
                    <span className="icon checkmark-icon" />
                    <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-color--chateauGreen">Schedule updated</span>
                  </div>
                }
              </div>
            </div>
          </form>
        </div>
        <ErrorModal
          errorModal={this.state.displayErrorModal}
          toggleErrorModal={this.toggleErrorModal}
          errMsg={this.state.gettingConfigErrMsg}
          tryAgain={undefined}
          err="Failed to get snapshot schedule settings"
          loading={loadingConfig}
        />
      </div>
    );
  }
}

export default withRouter(SnapshotSchedule);
