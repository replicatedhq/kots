import { Component } from "react";
import { Utilities } from "../utilities/utilities";
import "../scss/components/RestoreSnapshotRow.scss";
import Icon from "./Icon";

class RestoreSnapshotRow extends Component {
  state = {
    toggleVolumes: false,
    isLoadingBackupInfo: false,
    backupInfoMsg: "",
  };

  showVolumes = () => {
    this.setState({ toggleVolumes: !this.state.toggleVolumes });
  };

  fetchBackupInfo = (snapshotName) => {
    this.setState({
      isLoadingBackupInfo: true,
      backupInfoMsg: "",
    });

    fetch(`${process.env.API_ENDPOINT}/snapshot/${snapshotName}`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
    })
      .then((res) => res.json())
      .then((result) => {
        if (result.success) {
          this.setState({
            isLoadingBackupInfo: false,
            backupInfo: result.backupDetail,
            backupInfoMsg: "",
          });
        } else {
          this.setState({
            isLoadingBackupInfo: false,
            backupInfoMsg: result.error,
          });
        }
      })
      .catch((err) => {
        this.setState({
          isLoadingBackupInfo: false,
          backupInfoMsg: err,
        });
      });
  };

  componentDidUpdate(lastProps, lastState) {
    if (
      this.state.toggleVolumes !== lastState.toggleVolumes &&
      this.state.toggleVolumes
    ) {
      if (this.props.snapshot) {
        this.fetchBackupInfo(this.props.snapshot.name);
      }
    }
  }

  render() {
    const { toggleVolumes, backupInfo } = this.state;
    const { snapshot, isBackupSelected } = this.props;

    return (
      <div
        className={`flex flex-auto RestoreSnapshotRow--wrapper ${
          isBackupSelected && "backupSelected"
        }`}
        key={snapshot.name}
      >
        <div className="flex flex1 alignItems--center">
          <div className="flex flex-column">
            <div className="flex flex-column">
              <div className="flex flex-auto alignItems--center u-fontWeight--bold u-textColor--primary">
                <p className="u-fontSize--normal u-textColor--primary u-lineHeight--normal u-fontWeight--bold u-marginRight--10">
                  {snapshot?.name}
                </p>
                <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--bold u-lineHeight--normal u-marginRight--20">
                  {" "}
                  {Utilities.dateFormat(
                    snapshot?.startedAt,
                    "MM/DD/YY @ hh:mm a z"
                  )}
                </p>
              </div>
              <div className="flex alignItems--center u-marginTop--10">
                <span className="flex alignItems--center u-fontSize--normal u-textColor--bodyCopy u-fontWeight--bold u-lineHeight--normal u-marginRight--20">
                  <span className="icon snapshot-volume-icon" />{" "}
                  {snapshot?.volumeCount} volume
                  {snapshot?.volumeCount === 1 ? "" : "s"}
                  {!isBackupSelected && snapshot?.volumeCount > 0 ? (
                    <span onClick={this.showVolumes}>
                      <Icon
                        icon={toggleVolumes ? "up-arrow" : "down-arrow"}
                        size={12}
                        className="clickable u-marginLeft--5"
                      />
                    </span>
                  ) : null}{" "}
                </span>
                <span className="flex alignItems--center u-fontSize--normal u-textColor--bodyCopy u-fontWeight--bold u-lineHeight--normal u-marginRight--20">
                  <span className="icon snapshot-volume-size-icon" />{" "}
                  {Utilities.bytesToSize(snapshot?.volumeBytes)}{" "}
                </span>
              </div>
            </div>
            {toggleVolumes && (
              <div className="Timeline--wrapper" style={{ marginLeft: "9px" }}>
                {backupInfo?.volumes?.map((volume, i) => {
                  return (
                    <div
                      className="flex flex1 alignItems--center section u-marginTop--10"
                      key={`${volume.name}-${i}`}
                    >
                      <div className="flex flex-column">
                        <p className="u-fontSize--normal u-lineHeight--normal u-fontWeight--bold u-textColor--primary">
                          {volume.name}
                        </p>
                        <p className="u-fontSize--normal u-textColor--bodyCopy u-fontWeight--bold u-lineHeight--normal">
                          {" "}
                          {volume.sizeBytesHuman}{" "}
                        </p>
                      </div>
                    </div>
                  );
                })}
              </div>
            )}
          </div>
        </div>
        <div
          className={`flex-auto ${
            toggleVolumes
              ? ""
              : "flex justifyContent--center alignItems--center"
          }`}
        >
          {isBackupSelected ? (
            <button
              className="btn secondary blue"
              onClick={() => this.props.useDifferentBackup()}
            >
              {" "}
              Use a different backup{" "}
            </button>
          ) : (
            <button
              className="btn secondary blue"
              onClick={() => this.props.useBackup(snapshot)}
            >
              {" "}
              Use backup{" "}
            </button>
          )}
        </div>
      </div>
    );
  }
}

export default RestoreSnapshotRow;
