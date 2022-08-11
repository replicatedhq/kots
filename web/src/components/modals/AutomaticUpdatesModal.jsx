import React from "react";
import Modal from "react-modal";
import Select from "react-select";
import find from "lodash/find";
import {
  Utilities,
  getReadableCronDescriptor,
} from "../../utilities/utilities";

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
    value: "@never",
    label: "Never",
  },
  {
    value: "custom",
    label: "Custom",
  },
];

const DISABLED_AUTO_DEPLOY_OPTION = {
  value: "disabled",
  label: "Do not automatically deploy new versions",
};
const SEMVER_PATCH_AUTO_DEPLOY_OPTION = {
  value: "semver-patch",
  label: "Automatically deploy new patch versions",
};

const SEMVER_MINOR_PATCH_AUTO_DEPLOY_OPTION = {
  value: "semver-minor-patch",
  label: "Automatically deploy new patch and minor versions",
};

const SEMVER_MAJOR_MINOR_PATCH_AUTO_DEPLOY_OPTION = {
  value: "semver-major-minor-patch",
  label: "Automatically deploy new patch, minor, and major versions",
};

const SEQUENCE_AUTO_DEPLOY_OPTION = {
  value: "sequence",
  label: "Automatically deploy the most recent update",
};

// All available options for automatic deployments
const AUTO_DEPLOY_OPTIONS = [
  DISABLED_AUTO_DEPLOY_OPTION,
  SEMVER_PATCH_AUTO_DEPLOY_OPTION,
  SEMVER_MINOR_PATCH_AUTO_DEPLOY_OPTION,
  SEMVER_MAJOR_MINOR_PATCH_AUTO_DEPLOY_OPTION,
  SEQUENCE_AUTO_DEPLOY_OPTION,
];

// Valid automatic deployment options for licenses with semver required
const SEMVER_AUTO_DEPLOY_OPTIONS = [
  DISABLED_AUTO_DEPLOY_OPTION,
  SEMVER_PATCH_AUTO_DEPLOY_OPTION,
  SEMVER_MINOR_PATCH_AUTO_DEPLOY_OPTION,
  SEMVER_MAJOR_MINOR_PATCH_AUTO_DEPLOY_OPTION,
];

export default class AutomaticUpdatesModal extends React.Component {
  constructor(props) {
    super(props);

    let selectedSchedule = find(SCHEDULES, { value: props.updateCheckerSpec });
    if (!selectedSchedule) {
      selectedSchedule = find(SCHEDULES, { value: "custom" });
    }

    let selectedAutoDeploy = find(AUTO_DEPLOY_OPTIONS, [
      "value",
      props.autoDeploy,
    ]);
    if (!selectedAutoDeploy) {
      selectedAutoDeploy = find(AUTO_DEPLOY_OPTIONS, ["value", "disabled"]);
    }

    this.state = {
      updateCheckerSpec: props.updateCheckerSpec,
      configureAutomaticUpdatesErr: "",
      selectedSchedule,
      selectedAutoDeploy,
    };
  }

  onConfigureAutomaticUpdates = () => {
    const { updateCheckerSpec, selectedAutoDeploy } = this.state;
    const { appSlug } = this.props;

    this.setState({
      configureAutomaticUpdatesErr: "",
    });

    fetch(`${process.env.API_ENDPOINT}/app/${appSlug}/automaticupdates`, {
      headers: {
        Authorization: Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "PUT",
      body: JSON.stringify({
        updateCheckerSpec: updateCheckerSpec,
        autoDeploy: selectedAutoDeploy.value,
      }),
    })
      .then(async (res) => {
        if (!res.ok) {
          const response = await res.json();
          this.setState({
            configureAutomaticUpdatesErr: response?.error,
          });
          return;
        }

        this.setState({
          configureAutomaticUpdatesErr: "",
        });

        if (this.props.onAutomaticUpdatesConfigured) {
          this.props.onAutomaticUpdatesConfigured();
        }
      })
      .catch((err) => {
        this.setState({
          configureAutomaticUpdatesErr: String(err),
        });
      });
  };

  getReadableCronExpression = () => {
    const { updateCheckerSpec } = this.state;
    try {
      const readable = getReadableCronDescriptor(updateCheckerSpec);
      if (readable.includes("undefined")) {
        return "";
      } else {
        return readable;
      }
    } catch (error) {
      return "";
    }
  };

  handleScheduleChange = (selectedSchedule) => {
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
  };

  handleAutoDeployOptionChange = (selectedAutoDeploy) => {
    this.setState({
      selectedAutoDeploy: { ...selectedAutoDeploy },
    });
  };

  handleSequenceAutoUpdatesChange = (sequenceAutoDeployEnabled) => {
    if (sequenceAutoDeployEnabled) {
      this.setState({
        selectedAutoDeploy: { ...SEQUENCE_AUTO_DEPLOY_OPTION },
      });
    } else {
      this.setState({
        selectedAutoDeploy: { ...DISABLED_AUTO_DEPLOY_OPTION },
      });
    }
  };

  render() {
    const { isOpen, onRequestClose, isSemverRequired, gitopsIsConnected } =
      this.props;
    const {
      updateCheckerSpec,
      selectedSchedule,
      selectedAutoDeploy,
      configureAutomaticUpdatesErr,
    } = this.state;

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
          <span className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-marginBottom--15">
            Configure automatic updates
          </span>
          {gitopsIsConnected ? (
            <p className="u-fontSize--normal u-lineHeight--normal u-textColor--bodyCopy u-marginBottom--20">
              Configure how often you would like to automatically check for
              updates.
              <br />A commit will be made if an update was found.
            </p>
          ) : (
            <p className="u-fontSize--normal u-lineHeight--normal u-textColor--bodyCopy u-marginBottom--20">
              Configure how often you would like to automatically check for
              updates, and whether updates will be deployed automatically.
            </p>
          )}
          <div className="flex-column flex1">
            <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
              Automatically check for updates
            </p>
            <span className="u-fontSize--normal u-marginTop--5 u-textColor--info u-lineHeight--more u-marginBottom--15">
              Choose how frequently your application checks for updates. A
              custom schedule can be defined with a cron expression.
            </span>
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
                isOptionSelected={(option) => {
                  option.value === selectedSchedule;
                }}
              />
              <div className="flex-column flex2 u-marginLeft--10">
                <input
                  type="text"
                  className="Input u-marginBottom--5"
                  placeholder="0 0 * * MON"
                  value={updateCheckerSpec}
                  onChange={(e) => {
                    const schedule = find(SCHEDULES, { value: e.target.value });
                    const selectedSchedule = schedule
                      ? schedule
                      : find(SCHEDULES, { value: "custom" });
                    this.setState({
                      updateCheckerSpec: e.target.value,
                      selectedSchedule,
                    });
                  }}
                />
                {selectedSchedule.value === "@default" ? (
                  <span className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy">
                    Every 4 hours
                  </span>
                ) : humanReadableCron ? (
                  <span className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy">
                    {humanReadableCron}
                  </span>
                ) : null}
              </div>
            </div>
          </div>
          {!gitopsIsConnected && (
            <div className="flex-column flex1 u-marginTop--15">
              <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
                Automatically deploy new versions
              </p>
              {isSemverRequired ? (
                <>
                  <span className="u-marginTop--5 u-marginBottom--15 u-fontSize--normal u-textColor--info u-lineHeight--more u-fontWeight--medium">
                    Choose which versions will be deployed automatically. New
                    versions will never be deployed automatically when you
                    manually check for updates.
                  </span>
                  <Select
                    className="replicated-select-container flex1"
                    classNamePrefix="replicated-select"
                    placeholder="Automatically deploy new versions"
                    options={SEMVER_AUTO_DEPLOY_OPTIONS}
                    isSearchable={false}
                    getOptionValue={(option) => option.label}
                    value={selectedAutoDeploy}
                    onChange={this.handleAutoDeployOptionChange}
                    isOptionSelected={(option) => {
                      option.value === selectedAutoDeploy;
                    }}
                  />
                </>
              ) : (
                <>
                  <span className="u-marginTop--5 u-marginBottom--15 u-fontSize--normal u-textColor--info u-lineHeight--more u-fontWeight--medium">
                    Choose whether new versions will be deployed automatically.
                    New versions will never be deployed automatically when you
                    manually check for updates.
                  </span>
                  <div className="BoxedCheckbox-wrapper flex1 u-textAlign--left">
                    <div
                      className={`flex-auto flex ${
                        "sequence" === selectedAutoDeploy.value
                          ? "is-active"
                          : ""
                      }`}
                    >
                      <input
                        type="checkbox"
                        className="u-cursor--pointer"
                        id="sequenceAutoUpdatesEnabled"
                        checked={"sequence" === selectedAutoDeploy.value}
                        onChange={(e) => {
                          this.handleSequenceAutoUpdatesChange(
                            e.target.checked
                          );
                        }}
                      />
                      <label
                        htmlFor="sequenceAutoUpdatesEnabled"
                        className="flex1 flex u-width--full u-position--relative u-cursor--pointer u-userSelect--none"
                        style={{ marginTop: "2px" }}
                      >
                        <div className="flex flex-column u-marginLeft--5 justifyContent--center">
                          <p className="u-textColor--primary u-fontSize--normal u-fontWeight--medium">
                            Enable automatic deployment
                          </p>
                        </div>
                      </label>
                    </div>
                  </div>
                </>
              )}
            </div>
          )}
          {configureAutomaticUpdatesErr && (
            <span className="u-textColor--error u-fontSize--normal u-fontWeight--bold u-marginTop--15">
              Error: {configureAutomaticUpdatesErr}
            </span>
          )}
          <div className="flex u-marginTop--20">
            <button
              className="btn primary blue"
              onClick={this.onConfigureAutomaticUpdates}
            >
              Update
            </button>
            <button
              className="btn secondary u-marginLeft--10"
              onClick={onRequestClose}
            >
              Cancel
            </button>
          </div>
        </div>
      </Modal>
    );
  }
}
