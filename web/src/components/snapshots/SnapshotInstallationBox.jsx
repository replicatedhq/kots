import React, { Component } from "react";
import Icon from "../Icon";

export default class SnapshotInstallationBox extends Component {
  renderVeleroErrors = (snapshotSettings) => {
    if (!snapshotSettings?.isVeleroRunning && snapshotSettings?.veleroVersion) {
      return (
        <div className="flex u-marginBottom--20">
          <div className="flex u-marginRight--20">
            <span className="icon redWarningIcon" />
          </div>
          <div className="flex flex-column">
            <p className="u-textColor--error u-fontSize--larger u-fontWeight--bold">
              {" "}
              Velero is not running{" "}
            </p>
            <p className="u-fontSize--small u-textColor--bodyCopy u-lineHeight--normal u-fontWeight--medium u-marginTop--10">
              Velero has been detected, but it's not running successfully. To
              continue configuring and using snapshots Velero has to be running
              reliably.
              <a
                href="https://velero.io/docs/main/troubleshooting/"
                target="_blank"
                rel="noopener noreferrer"
                className="replicated-link u-marginLeft--5"
              >
                Get help
              </a>
            </p>
          </div>
        </div>
      );
    }
  };

  renderResticErrors = (snapshotSettings) => {
    if (snapshotSettings?.veleroVersion && !snapshotSettings?.resticVersion) {
      return (
        <div className="flex u-marginBottom--20">
          <div className="flex u-marginRight--20">
            <span className="icon redWarningIcon" />
          </div>
          <div className="flex flex-column">
            <p className="u-textColor--error u-fontSize--larger u-fontWeight--bold">
              {" "}
              Restic integration not found{" "}
            </p>
            <p className="u-fontSize--small u-textColor--bodyCopy u-lineHeight--normal u-fontWeight--medium u-marginTop--10">
              The Admin Console requires the Velero restic integration to use
              Snapshots, but it was not found. Please install the Velero restic
              integration to continue.
              <a
                href="https://velero.io/"
                target="_blank"
                rel="noopener noreferrer"
                className="replicated-link u-marginLeft--5"
              >
                Get help
              </a>
            </p>
          </div>
        </div>
      );
    } else if (
      snapshotSettings?.veleroVersion &&
      snapshotSettings?.resticVersion &&
      !snapshotSettings?.isResticRunning
    ) {
      return (
        <div className="flex u-marginBottom--20">
          <div className="flex u-marginRight--20">
            <span className="icon redWarningIcon" />
          </div>
          <div className="flex flex-column">
            <p className="u-textColor--error u-fontSize--larger u-fontWeight--bold">
              {" "}
              Restic is not working{" "}
            </p>
            <p className="u-fontSize--small u-textColor--bodyCopy u-lineHeight--normal u-fontWeight--medium u-marginTop--10">
              Velero and the restic integration have been detected, but restic
              is not running successfully. To continue configuring and using
              snapshots Restic has to be running reliably.
              <a
                href="https://velero.io/docs/main/restic/#troubleshooting"
                target="_blank"
                rel="noopener noreferrer"
                className="replicated-link u-marginLeft--5"
              >
                Get help
              </a>
            </p>
          </div>
        </div>
      );
    }
  };

  render() {
    const {
      snapshotSettings,
      hideCheckVeleroButton,
      fetchSnapshotSettings,
      renderNotVeleroMessage,
    } = this.props;

    return (
      <div className="flex1 flex-column">
        {this.renderVeleroErrors(snapshotSettings)}
        {this.renderResticErrors(snapshotSettings)}
        <div className="CheckVelero--wrapper flex1 flex-column justifyContent--center">
          <p className="u-textColor--primary u-fontSize--large u-fontWeight--bold">
            Check Velero installation
          </p>
          {!hideCheckVeleroButton ? (
            <div className="u-marginTop--12">
              <button
                className="btn secondary blue"
                onClick={() => fetchSnapshotSettings(true)}
              >
                Check for Velero
              </button>
            </div>
          ) : (
            renderNotVeleroMessage()
          )}
          {snapshotSettings?.veleroVersion ? (
            <span className="flex alignItems--center u-marginTop--10 u-fontSize--small u-fontWeight--medium u-textColor--primary">
              <Icon
                icon="check-circle-filled"
                size={16}
                className="u-marginRight--5 success-color"
              />
              Velero is installed on your cluster
            </span>
          ) : null}
        </div>
      </div>
    );
  }
}
