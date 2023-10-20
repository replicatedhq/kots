import { Component } from "react";
import {
  getCronFrequency,
  getReadableCronDescriptor,
} from "../../utilities/utilities";
import "../../scss/components/shared/SnapshotForm.scss";

export default class ScheduleSnapshotForm extends Component {
  constructor() {
    super();
    this.state = {
      maxRetain: "4",
      autoEnabled: false,
      timeout: 10,
      selectedSchedule: "weekly",
      frequency: "0 0 12 ? * MON *",
      s3bucket: "",
      bucketRegion: "",
      bucketPrefix: "",
      bucketKeyId: "",
      bucketKeySecret: "",
    };
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
    const frequency = getCronFrequency(selectedSchedule.value);
    this.setState(
      {
        selectedSchedule: selectedSchedule.value,
        frequency,
      },
      () => {
        this.getReadableCronExpression();
      }
    );
  };

  handleTimeoutChange = (timeout) => {
    this.setState({ timeout });
  };

  componentDidMount = () => {
    this.getReadableCronExpression();
  };

  render() {
    return (
      <form>
        <h2 className="u-fontSize--largest u-textColor--primary u-fontWeight--bold u-lineHeight--normal">
          Schedule a new snapshot
        </h2>
        <div className="flex-column flex1 u-marginTop--20"></div>

        <h2 className="u-fontSize--largest u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginTop--20">
          S3 config options
        </h2>
        <div className="flex-column flex1 u-marginTop--20">
          <div className="flex u-marginBottom--20">
            <div className="flex1 u-paddingRight--10">
              <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Bucket
              </p>
              <input
                type="text"
                className="Input"
                placeholder="Bucket name"
                value={this.state.s3bucket}
                onChange={(e) => {
                  this.handleFormChange("s3bucket", e);
                }}
              />
            </div>
            <div className="flex1 u-paddingRight--10">
              <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Region
              </p>
              <input
                type="text"
                className="Input"
                placeholder="Bucket region"
                value={this.state.bucketRegion}
                onChange={(e) => {
                  this.handleFormChange("bucketRegion", e);
                }}
              />
            </div>
            <div className="flex1 u-paddingLeft--10">
              <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Prefix <span className="optional-text">(optional)</span>
              </p>
              <input
                type="text"
                className="Input"
                placeholder="Bucket prefix"
                value={this.state.bucketPrefix}
                onChange={(e) => {
                  this.handleFormChange("bucketPrefix", e);
                }}
              />
            </div>
          </div>

          <div className="flex u-marginBottom--20">
            <div className="flex1 u-paddingRight--10">
              <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Access key ID <span className="optional-text">(optional)</span>
              </p>
              <input
                type="text"
                className="Input"
                placeholder="Bucket key Id"
                value={this.state.bucketKeyId}
                onChange={(e) => {
                  this.handleFormChange("bucketKeyId", e);
                }}
              />
            </div>
            <div className="flex1 u-paddingRight--10">
              <p className="u-fontSize--normal u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-marginBottom--10">
                Access key secret{" "}
                <span className="optional-text">(optional)</span>
              </p>
              <input
                type="text"
                className="Input"
                placeholder="Bucket key secret"
                value={this.state.bucketKeySecret}
                onChange={(e) => {
                  this.handleFormChange("bucketKeySecret", e);
                }}
              />
            </div>
          </div>
        </div>
      </form>
    );
  }
}
