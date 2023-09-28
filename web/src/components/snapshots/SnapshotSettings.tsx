import React, { Component } from "react";
import { RouterProps, withRouter } from "@src/utilities/react-router-utilities";
import { KotsPageTitle } from "@components/Head";
import isEmpty from "lodash/isEmpty";
import isEqual from "lodash/isEqual";

import Loader from "../shared/Loader";
import SnapshotStorageDestination from "./SnapshotStorageDestination";

import "../../scss/components/shared/SnapshotForm.scss";
import { isVeleroCorrectVersion } from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";
import { App } from "@types";
import Icon from "../Icon";

type Props = {
  appsList: App[];
  isKurlEnabled?: boolean;
} & RouterProps;

type State = {
  snapshotSettings?: {
    isVeleroRunning: boolean;
    veleroVersion: string;
    veleroPod?: object;
    nodeAgentPods?: object[];
  };
  isLoadingSnapshotSettings: boolean;
  snapshotSettingsErr: boolean;
  snapshotSettingsErrMsg: string;
  toggleSnapshotView: boolean;
  hideCheckVeleroButton: boolean;
  updateConfirm: boolean;
  updatingSettings: boolean;
  updateErrorMsg: string;
  showConfigureSnapshotsModal: boolean;
  kotsadmRequiresVeleroAccess: boolean;
  minimalRBACKotsadmNamespace: string;
  showResetFileSystemWarningModal: boolean;
  resetFileSystemWarningMessage: string;
  snapshotSettingsJob: Repeater;
  checkForVeleroAndNodeAgent: boolean;
  isEmptyView?: boolean;
  veleroUpdated?: boolean;
  nodeAgentUpdated?: boolean;
};

class SnapshotSettings extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      snapshotSettings: undefined,
      isLoadingSnapshotSettings: true,
      snapshotSettingsErr: false,
      snapshotSettingsErrMsg: "",
      toggleSnapshotView: false,
      hideCheckVeleroButton: false,
      updateConfirm: false,
      updatingSettings: false,
      updateErrorMsg: "",
      showConfigureSnapshotsModal: false,
      kotsadmRequiresVeleroAccess: false,
      minimalRBACKotsadmNamespace: "",
      showResetFileSystemWarningModal: false,
      resetFileSystemWarningMessage: "",
      snapshotSettingsJob: new Repeater(),
      checkForVeleroAndNodeAgent: false,
    };
  }

  fetchSnapshotSettings = (isCheckForVelero?: boolean) => {
    this.setState({
      isLoadingSnapshotSettings: true,
      snapshotSettingsErr: false,
      snapshotSettingsErrMsg: "",
      hideCheckVeleroButton: isCheckForVelero ? true : false,
      minimalRBACKotsadmNamespace: "",
    });

    return fetch(`${process.env.API_ENDPOINT}/snapshots/settings`, {
      method: "GET",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
    })
      .then(async (res) => {
        if (!res.ok && res.status === 409) {
          const result = await res.json();
          if (result.kotsadmRequiresVeleroAccess) {
            this.setState({
              isLoadingSnapshotSettings: false,
            });
            setTimeout(() => {
              this.setState({ hideCheckVeleroButton: false });
            }, 5000);
            this.openConfigureSnapshotsMinimalRBACModal(
              result.kotsadmRequiresVeleroAccess,
              result.kotsadmNamespace
            );
            return;
          }
        }

        const result = await res.json();

        this.setState({
          snapshotSettings: result,
          kotsadmRequiresVeleroAccess: false,
          isLoadingSnapshotSettings: false,
          snapshotSettingsErr: false,
          snapshotSettingsErrMsg: "",
        });
        if (!result.veleroVersion) {
          setTimeout(() => {
            this.setState({ hideCheckVeleroButton: false });
          }, 5000);
        } else {
          this.setState({ hideCheckVeleroButton: false });
        }
      })
      .catch((err) => {
        this.setState({
          isLoadingSnapshotSettings: false,
          snapshotSettingsErr: true,
          snapshotSettingsErrMsg: err,
        });
      });
  };

  componentDidMount() {
    this.fetchSnapshotSettings();

    if (!isEmpty(this.props.location.search)) {
      if (this.props.location.search === "?configure=true") {
        this.setState({ showConfigureSnapshotsModal: true });
      }
    }
  }

  componentWillUnmount() {
    this.state.snapshotSettingsJob.stop();
  }

  pollSnapshotSettingsOnUpdate = () => {
    this.setState({ checkForVeleroAndNodeAgent: true });
    this.state.snapshotSettingsJob.start(this.fetchSnapshotSettings, 2000);
  };

  componentDidUpdate = (_: Props, lastState: State) => {
    if (
      this.state.snapshotSettings !== lastState.snapshotSettings &&
      this.state.snapshotSettings
    ) {
      if (
        this.state.snapshotSettings?.isVeleroRunning &&
        !this.state.snapshotSettings?.veleroVersion &&
        !this.state.updatingSettings
      ) {
        this.props.navigate("/snapshots/settings?configure=true", {
          replace: true,
        });
        this.setState({ showConfigureSnapshotsModal: true });
      }

      if (this.state.checkForVeleroAndNodeAgent) {
        if (
          this.state.snapshotSettings?.veleroPod !==
            lastState.snapshotSettings?.veleroPod &&
          !isEmpty(this.state.snapshotSettings?.veleroPod)
        ) {
          this.setState({ veleroUpdated: true });
        }

        let sortedStateNodeAgentPods: object[] | undefined = [];
        let sortedLastStateNodeAgentPods: object[] | undefined = [];
        if (!isEmpty(this.state.snapshotSettings?.nodeAgentPods)) {
          sortedStateNodeAgentPods =
            this.state.snapshotSettings?.nodeAgentPods?.sort();
        }
        if (!isEmpty(lastState.snapshotSettings?.nodeAgentPods)) {
          sortedLastStateNodeAgentPods =
            lastState.snapshotSettings?.nodeAgentPods?.sort();
        }
        if (
          !isEqual(sortedStateNodeAgentPods, sortedLastStateNodeAgentPods) &&
          !isEmpty(this.state.snapshotSettings?.nodeAgentPods)
        ) {
          this.setState({ nodeAgentUpdated: true });
        }

        if (
          this.state.updatingSettings &&
          this.state.veleroUpdated &&
          this.state.nodeAgentUpdated
        ) {
          this.setState({
            updatingSettings: false,
            updateConfirm: true,
            checkForVeleroAndNodeAgent: false,
            veleroUpdated: false,
            nodeAgentUpdated: false,
          });

          setTimeout(() => {
            this.setState({ updateConfirm: false });
          }, 5000);

          this.state.snapshotSettingsJob.stop();
        }
      }
    }
  };

  toggleSnapshotView = (isEmptyView?: boolean) => {
    this.setState({
      toggleSnapshotView: !this.state.toggleSnapshotView,
      isEmptyView: isEmptyView ? isEmptyView : false,
    });
  };

  updateSettings = (payload: object) => {
    this.setState({
      updatingSettings: true,
      updateErrorMsg: "",
      updateConfirm: false,
    });

    fetch(`${process.env.API_ENDPOINT}/snapshots/settings`, {
      method: "PUT",
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
      body: JSON.stringify(payload),
    })
      .then(async (res) => {
        if (!res.ok && res.status === 409) {
          const result = await res.json();
          if (result.kotsadmRequiresVeleroAccess) {
            this.setState({
              updatingSettings: false,
            });
            this.openConfigureSnapshotsMinimalRBACModal(
              result.kotsadmRequiresVeleroAccess,
              result.kotsadmNamespace
            );
            return;
          }
          this.setState({
            updatingSettings: false,
            showResetFileSystemWarningModal: true,
            resetFileSystemWarningMessage: result.error,
          });
          return;
        }

        const settingsResponse = await res.json();
        if (!res.ok) {
          this.setState({
            updatingSettings: false,
            updateErrorMsg: settingsResponse.error,
          });
          return;
        }

        if (settingsResponse.success) {
          this.setState(
            {
              snapshotSettings: settingsResponse,
              updateErrorMsg: "",
            },
            this.pollSnapshotSettingsOnUpdate
          );
        } else {
          this.setState({
            updatingSettings: false,
            updateErrorMsg: settingsResponse.error,
          });
        }
      })
      .catch((err) => {
        console.error(err);
        this.setState({
          updatingSettings: false,
          updateErrorMsg: "Something went wrong, please try again.",
        });
      });
  };

  renderNotVeleroMessage = () => {
    return (
      <p className="u-textColor--error u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginTop--12">
        Not able to find Velero
      </p>
    );
  };

  openConfigureSnapshotsMinimalRBACModal = (
    kotsadmRequiresVeleroAccess: boolean,
    minimalRBACKotsadmNamespace: string
  ): void => {
    this.setState(
      {
        showConfigureSnapshotsModal: true,
        kotsadmRequiresVeleroAccess,
        minimalRBACKotsadmNamespace,
      },
      () => {
        this.props.navigate("/snapshots/settings?configure=true", {
          replace: true,
        });
      }
    );
  };

  toggleConfigureSnapshotsModal = () => {
    if (this.state.showConfigureSnapshotsModal) {
      this.setState({ showConfigureSnapshotsModal: false }, () => {
        this.props.navigate("/snapshots/settings", {
          replace: true,
        });
      });
    } else {
      this.setState({ showConfigureSnapshotsModal: true }, () => {
        this.props.navigate("/snapshots/settings?configure=true", {
          replace: true,
        });
      });
    }
  };

  hideResetFileSystemWarningModal = () => {
    this.setState({ showResetFileSystemWarningModal: false });
  };

  render() {
    const {
      isLoadingSnapshotSettings,
      snapshotSettings,
      hideCheckVeleroButton,
      updateConfirm,
      updatingSettings,
      updateErrorMsg,
      isEmptyView,
      checkForVeleroAndNodeAgent,
    } = this.state;
    const isLicenseUpload = !!this.props.location.search;

    if (isLoadingSnapshotSettings && !checkForVeleroAndNodeAgent) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    return (
      <div className="flex1 flex-column u-overflow--auto">
        <KotsPageTitle pageName="Snapshot Settings" />
        {!isVeleroCorrectVersion(snapshotSettings) &&
        !checkForVeleroAndNodeAgent ? (
          <div className="VeleroWarningBlock">
            <Icon icon={"warning"} size={24} className="warning-color" />
            <p>
              {" "}
              To use snapshots reliably, install Velero version 1.5.1 or greater{" "}
            </p>
          </div>
        ) : null}
        <div className="container flex-column flex1u-paddingTop--30 u-paddingBottom--20 u-marginTop--10 alignItems--center">
          <SnapshotStorageDestination
            snapshotSettings={snapshotSettings}
            updateSettings={this.updateSettings}
            fetchSnapshotSettings={this.fetchSnapshotSettings}
            checkForVeleroAndNodeAgent={checkForVeleroAndNodeAgent}
            updateConfirm={updateConfirm}
            updatingSettings={updatingSettings}
            updateErrorMsg={updateErrorMsg}
            renderNotVeleroMessage={this.renderNotVeleroMessage}
            toggleSnapshotView={this.toggleSnapshotView}
            isEmptyView={isEmptyView}
            hideCheckVeleroButton={hideCheckVeleroButton}
            isLicenseUpload={isLicenseUpload}
            showConfigureSnapshotsModal={this.state.showConfigureSnapshotsModal}
            toggleConfigureSnapshotsModal={this.toggleConfigureSnapshotsModal}
            openConfigureSnapshotsMinimalRBACModal={
              this.openConfigureSnapshotsMinimalRBACModal
            }
            kotsadmRequiresVeleroAccess={this.state.kotsadmRequiresVeleroAccess}
            minimalRBACKotsadmNamespace={this.state.minimalRBACKotsadmNamespace}
            showResetFileSystemWarningModal={
              this.state.showResetFileSystemWarningModal
            }
            resetFileSystemWarningMessage={
              this.state.resetFileSystemWarningMessage
            }
            hideResetFileSystemWarningModal={
              this.hideResetFileSystemWarningModal
            }
            isKurlEnabled={this.props.isKurlEnabled}
            apps={this.props.appsList}
          />
        </div>
      </div>
    );
  }
}

export default withRouter(SnapshotSettings);
