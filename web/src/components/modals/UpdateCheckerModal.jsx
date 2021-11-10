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
const AUTO_INSTALL_OPTIONS = [
  {
    value: "none",
    label: "Do not automatically install new versions",
  },
  {
    value: "patch",
    label: "Automatically install new patch versions",
  },
  {
    value: "minor-patch",
    label: "Automatically install new patch and minor versions",
  },
  {
    value: "minor-patch-major",
    label: "Automatically install new path, minor, and major versions",
  }
];

export default class UpdateCheckerModal extends React.Component {
  constructor(props) {
    super(props);

    let selectedSchedule = find(SCHEDULES, { value: props.updateCheckerSpec });
    if (!selectedSchedule) {
      selectedSchedule = find(SCHEDULES, { value: "custom" });
    }

    let selectedInstallOption = find(AUTO_INSTALL_OPTIONS, ["value", props.autoInstallOption]);
    if (!selectedInstallOption) {
      selectedInstallOption = find(AUTO_INSTALL_OPTIONS, ["value", "none"])
    }

    this.state = {
      updateCheckerSpec: props.updateCheckerSpec,
      submitUpdateCheckerSpecErr: "",
      selectedSchedule,
    };
  }

  onSubmitUpdateCheckerSpec = () => {
    const { updateCheckerSpec } = this.state;
    const { appSlug } = this.props;

    this.setState({
      submitUpdateCheckerSpecErr: ""
    });

    fetch(`${window.env.API_ENDPOINT}/app/${appSlug}/updatecheckerspec`, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "PUT",
      body: JSON.stringify({
        updateCheckerSpec: updateCheckerSpec,
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

  handleInstallOptionChange = selectedInstallOption => {
    this.setState({
      selectedInstallOption,
      selectedInstallOptionValue: selectedInstallOption.value,
    });
  }

  render() {
    const { isOpen, onRequestClose, gitopsEnabled } = this.props;
    const { updateCheckerSpec, selectedSchedule, selectedInstallOption, submitUpdateCheckerSpecErr } = this.state;

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
            <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Cron expression</p>
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
            <div className="info-box u-marginTop--15">
              <span className="u-fontSize--small">
                You can enter <span className="u-fontWeight--bold u-textColor--primary">@never</span> to disable scheduled update checks
              </span>
            </div>
          </div>
          <div className="flex-column flex1 u-marginTop--15">
            <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Automatically install new versions</p>
            <Select
              className="replicated-select-container flex1"
              classNamePrefix="replicated-select"
              placeholder="Automatically install new versions"
              options={AUTO_INSTALL_OPTIONS}
              isSearchable={false}
              getOptionValue={(option) => option.label}
              value={selectedInstallOption}
              onChange={this.handleInstallOptionChange}
              isOptionSelected={(option) => { option.value === selectedInstallOption }}
            />
            <span className="u-marginTop--10 u-fontSize--small u-textColor--info u-fontWeight--medium">Releases without a valid semver will <span className="u-fontWeight--bold">not</span> be automatically installed.</span>
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