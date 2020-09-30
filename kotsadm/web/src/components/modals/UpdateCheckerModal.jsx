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

export default class UpdateCheckerModal extends React.Component {
  constructor(props) {
    super(props);

    let selectedSchedule = find(SCHEDULES, { value: props.updateCheckerSpec });
    if (!selectedSchedule) {
      selectedSchedule = find(SCHEDULES, { value: "custom" });
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

  render() {
    const { isOpen, onRequestClose, gitopsEnabled } = this.props;
    const { updateCheckerSpec, selectedSchedule, submitUpdateCheckerSpecErr } = this.state;

    const humanReadableCron = this.getReadableCronExpression(updateCheckerSpec);

    return (
      <Modal
        isOpen={isOpen}
        onRequestClose={onRequestClose}
        shouldReturnFocusAfterClose={false}
        contentLabel="Update Checker"
        ariaHideApp={false}
        className="Modal SmallSize"
      >
        <div className="u-position--relative flex-column u-padding--20">
          <span className="u-fontSize--largest u-fontWeight--bold u-color--tuna u-marginBottom--15">Configure automatic update checks</span>
          {gitopsEnabled ? 
            <p className="u-fontSize--normal u-lineHeight--normal u-color--dustyGray u-marginBottom--20">
              Configure how often you would like to automatically check for updates.<br/>A commit will be made if an update was found.
            </p>
            :
            <p className="u-fontSize--normal u-lineHeight--normal u-color--dustyGray u-marginBottom--20">
              Configure how often you would like to automatically check for updates.<br/>This will only download updates, not deploy them.
            </p>
          }
          <div className="info-box u-marginBottom--20">
            <span className="u-fontSize--small">
              You can enter <span className="u-fontWeight--bold u-color--tuna">@never</span> to disable scheduled update checks
            </span>
          </div>
          <div className="flex-column flex1 u-paddingLeft--5">
            <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">Cron expression</p>
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
                menuPlacement="top"
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
                  <span className="u-fontSize--small u-fontWeight--medium u-color--dustyGray">Every 4 hours</span>
                  :
                  humanReadableCron ?
                    <span className="u-fontSize--small u-fontWeight--medium u-color--dustyGray">{humanReadableCron}</span>
                    :
                    null
                }
              </div>
            </div>
            {submitUpdateCheckerSpecErr && <span className="u-color--chestnut u-fontSize--small u-fontWeight--bold u-marginTop--15">Error: {submitUpdateCheckerSpecErr}</span>}
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