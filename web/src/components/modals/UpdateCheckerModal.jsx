import React from "react";
import Modal from "react-modal";
import Select from "react-select";
import find from "lodash/find";
import { Utilities, getReadableCronDescriptor } from "../../utilities/utilities";

const SCHEDULES = [
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

export default class UpdateCheckerModal extends React.Component {
  constructor(props) {
    super(props);

    let selectedSchedule = find(SCHEDULES, { value: props.updateCheckerSpec });
    if (!selectedSchedule) {
      selectedSchedule = find(SCHEDULES, { value: "custom" });
    }

    let selectedSemverAutoDeploy = find(SEMVER_AUTO_DEPLOY_OPTIONS, ["value", props.semverAutoDeploy]);
    if (!selectedSemverAutoDeploy) {
      selectedSemverAutoDeploy = find(SEMVER_AUTO_DEPLOY_OPTIONS, ["value", "disabled"])
    }

    this.state = {
      updateCheckerSpec: props.updateCheckerSpec,
      submitUpdateCheckerSpecErr: "",
      selectedSchedule,
      selectedSemverAutoDeploy,
    };
  }

  onSubmitUpdateCheckerSpec = () => {
    const { updateCheckerSpec, selectedSemverAutoDeploy } = this.state;
    const { appSlug } = this.props;

    this.setState({
      submitUpdateCheckerSpecErr: ""
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
        semverAutoDeploySchedule: "" // TODO: @Grayson
      })
    })
      .then(async (res) => {
        if (!res.ok) {
          const response = await res.json();
          this.setState({
            submitUpdateCheckerSpecErr: response?.error
          });
          return;
        }

        this.setState({
          submitUpdateCheckerSpecErr: ""
        });
        
        if (this.props.onUpdateCheckerSpecSubmitted) {
          this.props.onUpdateCheckerSpecSubmitted();
        }
      })
      .catch((err) => {
        this.setState({
          submitUpdateCheckerSpecErr: String(err)
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

  handleScheduleChange = selectedSchedule => {
    let updateCheckerSpec;
    if (selectedSchedule.value !== "custom") {
      updateCheckerSpec = selectedSchedule.value;
    } else {
      updateCheckerSpec = "0 2 * * WED,SAT"; // arbitrary choice
    }
    this.setState({
      selectedSchedule,
      updateCheckerSpec,
    });
  }

  handleSemverAutoDeployOptionChange = selectedSemverAutoDeploy => {
    this.setState({
      selectedSemverAutoDeploy: { ...selectedSemverAutoDeploy },
    });
  }

  render() {
    const { isOpen, onRequestClose, gitopsEnabled } = this.props;
    const { updateCheckerSpec, selectedSchedule, selectedSemverAutoDeploy, submitUpdateCheckerSpecErr } = this.state;

    const humanReadableCron = this.getReadableCronExpression(updateCheckerSpec);

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
          <span className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-marginBottom--15">Configure automatic update checks</span>
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
                options={SCHEDULES}
                isSearchable={false}
                getOptionValue={(schedule) => schedule.label}
                value={selectedSchedule}
                onChange={this.handleScheduleChange}
                isOptionSelected={(option) => { option.value === selectedSchedule }}
              />
              <div className="flex-column flex2 u-marginLeft--10">
                <input
                  type="text"
                  className="Input u-marginBottom--5"
                  placeholder="0 0 * * MON"
                  value={updateCheckerSpec}
                  onChange={(e) => {
                    const schedule = find(SCHEDULES, { value: e.target.value });
                    const selectedSchedule = schedule ? schedule : find(SCHEDULES, { value: "custom" });
                    this.setState({ updateCheckerSpec: e.target.value, selectedSchedule });
                  }}
                />
                {selectedSchedule.value === "@default" ?
                  <span className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy">Every 4 hours</span>
                  :
                  humanReadableCron ?
                    <span className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy">{humanReadableCron}</span>
                    :
                    null
                }
              </div>
            </div>
            {submitUpdateCheckerSpecErr && <span className="u-textColor--error u-fontSize--small u-fontWeight--bold u-marginTop--15">Error: {submitUpdateCheckerSpecErr}</span>}
          </div>
          <div className="flex-column flex1 u-marginTop--15">
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
          <div className="flex u-marginTop--20">
            <button className="btn primary blue" onClick={this.onSubmitUpdateCheckerSpec}>Update</button>
            <button className="btn secondary u-marginLeft--10" onClick={onRequestClose}>Cancel</button>
          </div>
        </div>
      </Modal>
    );
  }
}