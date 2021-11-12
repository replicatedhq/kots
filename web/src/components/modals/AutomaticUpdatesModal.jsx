import React from "react";
import Modal from "react-modal";
import Select from "react-select";
import find from "lodash/find";
import { Utilities, getReadableCronDescriptor } from "../../utilities/utilities";

const UPDATE_CHECK_SCHEDULES = [
  {
    value: "@hourly",
    label: "Hourly",
  },
  {
    value: "@daily",
    label: "Daily",
  },
  {
    value: "@weekly",
    label: "Weekly",
  },
  {
    value: "@default",
    label: "Default",
  },
  {
    value: "custom",
    label: "Custom",
  },
];

const SEMVER_AUTO_DEPLOY_OPTIONS = [
  {
    value: "disabled",
    label: "Do not automatically deploy new versions",
  },
  {
    value: "patch",
    label: "Automatically deploy new patch versions",
  },
  {
    value: "minor-patch",
    label: "Automatically deploy new patch and minor versions",
  },
  {
    value: "major-minor-patch",
    label: "Automatically deploy new patch, minor, and major versions",
  }
];

const SEMVER_AUTO_DEPLOY_SCHEDULES = [
  {
    value: "@daily",
    label: "Daily",
  },
  {
    value: "@weekly",
    label: "Weekly",
  },
  {
    value: "@monthly",
    label: "Monthly",
  },
  {
    value: "@default",
    label: "Default",
  },
  {
    value: "custom",
    label: "Custom",
  },
];

export default class AutomaticUpdatesModal extends React.Component {
  constructor(props) {
    super(props);

    let selectedUpdateCheckSchedule = find(UPDATE_CHECK_SCHEDULES, { value: props.updateCheckerSpec });
    if (!selectedUpdateCheckSchedule) {
      selectedUpdateCheckSchedule = find(UPDATE_CHECK_SCHEDULES, { value: "custom" });
    }

    let selectedSemverAutoDeploy = find(SEMVER_AUTO_DEPLOY_OPTIONS, ["value", props.semverAutoDeploy]);
    if (!selectedSemverAutoDeploy) {
      selectedSemverAutoDeploy = find(SEMVER_AUTO_DEPLOY_OPTIONS, ["value", "disabled"])
    }

    let selectedSemverAutoDeploySchedule = find(SEMVER_AUTO_DEPLOY_SCHEDULES, { value: props.semverAutoDeploySchedule });
    if (!selectedSemverAutoDeploySchedule) {
      selectedSemverAutoDeploySchedule = find(SEMVER_AUTO_DEPLOY_SCHEDULES, { value: "custom" });
    }

    this.state = {
      updateCheckerSpec: props.updateCheckerSpec,
      selectedUpdateCheckSchedule,
      semverAutoDeployCronSpec: props.semverAutoDeploySchedule,
      selectedSemverAutoDeploy,
      selectedSemverAutoDeploySchedule,
      configureAutomaticUpdatesErr: ""
    };
  }

  onSubmitUpdateCheckerSpec = () => {
    const { updateCheckerSpec, selectedSemverAutoDeploy, semverAutoDeployCronSpec } = this.state;
    const { appSlug } = this.props;

    this.setState({
      configureAutomaticUpdatesErr: ""
    });

    fetch(`${window.env.API_ENDPOINT}/app/${appSlug}/automaticupdates`, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "PUT",
      body: JSON.stringify({
        updateCheckerSpec: updateCheckerSpec,
        semverAutoDeploy: selectedSemverAutoDeploy.value,
        semverAutoDeploySchedule: semverAutoDeployCronSpec,
      })
    })
      .then(async (res) => {
        if (!res.ok) {
          const response = await res.json();
          this.setState({
            configureAutomaticUpdatesErr: response?.error
          });
          return;
        }

        this.setState({
          configureAutomaticUpdatesErr: ""
        });
        
        if (this.props.onUpdateCheckerSpecSubmitted) {
          this.props.onUpdateCheckerSpecSubmitted();
        }
      })
      .catch((err) => {
        this.setState({
          configureAutomaticUpdatesErr: String(err)
        });
      });
  }

  getReadableCronExpression = () => {
    const { updateCheckerSpec } = this.state;
    try {
      const readable = getReadableCronDescriptor(updateCheckerSpec);
      if (readable.includes("undefined")) {
        return "";
      } else {
        return readable;
      }
    } catch(error) {
      return "";
    }
  }

  handleUpdateCheckScheduleChange = selectedUpdateCheckSchedule => {
    let updateCheckerSpec;
    if (selectedUpdateCheckSchedule.value !== "custom") {
      updateCheckerSpec = selectedUpdateCheckSchedule.value;
    } else {
      updateCheckerSpec = "0 2 * * WED,SAT"; // arbitrary choice
    }
    this.setState({
      selectedUpdateCheckSchedule,
      updateCheckerSpec,
    });
  }

  handleSemverAutoDeployOptionChange = selectedSemverAutoDeploy => {
    this.setState({
      selectedSemverAutoDeploy: { ...selectedSemverAutoDeploy },
    });
  }

  handleSemverAutoDeployScheduleChange = selectedSemverAutoDeploySchedule => {
    let semverAutoDeployCronSpec;
    if (selectedSemverAutoDeploySchedule.value !== "custom") {
      semverAutoDeployCronSpec = selectedSemverAutoDeploySchedule.value;
    } else {
      semverAutoDeployCronSpec = "0 2 * * WED,SAT"; // arbitrary choice
    }
    this.setState({
      selectedSemverAutoDeploySchedule,
      semverAutoDeployCronSpec,
    });
  }

  render() {
    const { isOpen, onRequestClose, gitopsEnabled } = this.props;
    const { updateCheckerSpec, selectedUpdateCheckSchedule, selectedSemverAutoDeploy, selectedSemverAutoDeploySchedule, semverAutoDeployCronSpec, configureAutomaticUpdatesErr } = this.state;

    const updateCheckHumanReadableCron = this.getReadableCronExpression(updateCheckerSpec);
    const semverAutoDeployHumanReadableCron = this.getReadableCronExpression(semverAutoDeployCronSpec);

    return (
      <Modal
        isOpen={isOpen}
        onRequestClose={onRequestClose}
        shouldReturnFocusAfterClose={false}
        contentLabel="Update Checker"
        ariaHideApp={false}
        className="Modal SmallSize ConfigureUpdatesModal"
      >
        <div className="u-position--relative flex-column u-padding--20">
          <span className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-marginBottom--15">Configure automatic updates</span>
          {gitopsEnabled ? 
            <p className="u-fontSize--normal u-lineHeight--normal u-textColor--bodyCopy u-marginBottom--20">
              Configure how often you would like to automatically check for updates.<br/>A commit will be made if an update was found.
            </p>
            :
            <p className="u-fontSize--normal u-lineHeight--normal u-textColor--bodyCopy u-marginBottom--20">
              Configure how often you would like to automatically check for updates.<br/>This will only download updates, not deploy them.
            </p>
          }
          <div className="flex-column flex1">
            <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal">Cron expression</p>
            <span className="u-fontSize--small u-marginTop--5 u-textColor--info u-marginBottom--15">You can enter <span className="u-fontWeight--bold u-textColor--primary">@never</span> to disable scheduled update checks</span>
            <div className="flex flex1">
              <Select
                className="replicated-select-container flex1"
                classNamePrefix="replicated-select"
                placeholder="Select an interval"
                options={UPDATE_CHECK_SCHEDULES}
                isSearchable={false}
                getOptionValue={(schedule) => schedule.label}
                value={selectedUpdateCheckSchedule}
                onChange={this.handleUpdateCheckScheduleChange}
                isOptionSelected={(option) => { option.value === selectedUpdateCheckSchedule }}
              />
              <div className="flex-column flex2 u-marginLeft--10">
                <input
                  type="text"
                  className="Input u-marginBottom--5"
                  placeholder="0 0 * * MON"
                  value={updateCheckerSpec}
                  onChange={(e) => {
                    const schedule = find(UPDATE_CHECK_SCHEDULES, { value: e.target.value });
                    const selectedUpdateCheckSchedule = schedule ? schedule : find(UPDATE_CHECK_SCHEDULES, { value: "custom" });
                    this.setState({ updateCheckerSpec: e.target.value, selectedUpdateCheckSchedule });
                  }}
                />
                {selectedUpdateCheckSchedule.value === "@default" ?
                  <span className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy">Every 4 hours</span>
                  :
                  updateCheckHumanReadableCron ?
                    <span className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy">{updateCheckHumanReadableCron}</span>
                    :
                    null
                }
              </div>
            </div>
          </div>
          <div className="flex-column flex1 u-marginTop--15 u-marginBottom--15">
            <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal">Automatically deploy new versions</p>
            <span className="u-marginTop--5 u-marginBottom--15 u-fontSize--small u-textColor--info u-fontWeight--medium">Releases without a valid <a href="https://semver.org/" className="replicated-link" target="_blank" rel="noopener noreferrer">semantic version</a> will <span className="u-fontWeight--bold">not</span> be automatically deployed.</span>
            <Select
              className="replicated-select-container flex1"
              classNamePrefix="replicated-select"
              placeholder="Automatically deploy new versions"
              options={SEMVER_AUTO_DEPLOY_OPTIONS}
              isSearchable={false}
              getOptionValue={(option) => option.label}
              value={selectedSemverAutoDeploy}
              onChange={this.handleSemverAutoDeployOptionChange}
              isOptionSelected={(option) => { option.value === selectedSemverAutoDeploy }}
            />
          </div>
          <div className="flex-column flex1">
            <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal">Cron expression</p>
            <div className="flex flex1">
              <Select
                className="replicated-select-container flex1"
                classNamePrefix="replicated-select"
                placeholder="Select an interval"
                options={SEMVER_AUTO_DEPLOY_SCHEDULES}
                isSearchable={false}
                getOptionValue={(schedule) => schedule.label}
                value={selectedSemverAutoDeploySchedule}
                onChange={this.handleSemverAutoDeployScheduleChange}
                isOptionSelected={(option) => { option.value === selectedSemverAutoDeploySchedule }}
              />
              <div className="flex-column flex2 u-marginLeft--10">
                <input
                  type="text"
                  className="Input u-marginBottom--5"
                  placeholder="0 0 * * MON"
                  value={semverAutoDeployCronSpec}
                  onChange={(e) => {
                    const schedule = find(SEMVER_AUTO_DEPLOY_SCHEDULES, { value: e.target.value });
                    const selectedSemverAutoDeploySchedule = schedule ? schedule : find(SEMVER_AUTO_DEPLOY_SCHEDULES, { value: "custom" });
                    this.setState({ updateCheckerSpec: e.target.value, selectedSemverAutoDeploySchedule });
                  }}
                />
                {selectedSemverAutoDeploySchedule.value === "@default" ?
                  <span className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy">As soon as they're available</span>
                  :
                  semverAutoDeployHumanReadableCron ?
                    <span className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy">{semverAutoDeployHumanReadableCron}</span>
                    :
                    null
                }
              </div>
            </div>
          </div>
          {configureAutomaticUpdatesErr && <span className="u-textColor--error u-fontSize--small u-fontWeight--bold u-marginTop--15">Error: {configureAutomaticUpdatesErr}</span>}
          <div className="flex u-marginTop--20">
            <button className="btn primary blue" onClick={this.onSubmitUpdateCheckerSpec}>Update</button>
            <button className="btn secondary u-marginLeft--10" onClick={onRequestClose}>Cancel</button>
          </div>
        </div>
      </Modal>
    );
  }
}