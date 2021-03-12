import React, { Component } from "react";
import { withRouter } from "react-router-dom"
import Helmet from "react-helmet";
import isEmpty from "lodash/isEmpty";

import Loader from "../shared/Loader";
import SnapshotStorageDestination from "./SnapshotStorageDestination";

import "../../scss/components/shared/SnapshotForm.scss";
import { Utilities } from "../../utilities/utilities";


class SnapshotSettings extends Component {
  state = {
    snapshotSettings: null,
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
  };

  fetchSnapshotSettings = (isCheckForVelero) => {
    this.setState({
      isLoadingSnapshotSettings: true,
      snapshotSettingsErr: false,
      snapshotSettingsErrMsg: "",
      hideCheckVeleroButton: isCheckForVelero ? true : false,
      minimalRBACKotsadmNamespace: "",
    });

    fetch(`${window.env.API_ENDPOINT}/snapshots/settings`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    })
      .then(async res => {
        if (!res.ok && res.status === 409) {
          const result = await res.json();
          if (result.kotsadmRequiresVeleroAccess) {
            this.setState({
              isLoadingSnapshotSettings: false,
            });
            setTimeout(() => {
              this.setState({ hideCheckVeleroButton: false });
            }, 5000);
            this.openConfigureSnapshotsMinimalRBACModal(result.kotsadmRequiresVeleroAccess, result.kotsadmNamespace);
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
      .catch(err => {
        this.setState({
          isLoadingSnapshotSettings: false,
          snapshotSettingsErr: true,
          snapshotSettingsErrMsg: err,
        })
      })
  }

  componentDidMount() {
    this.fetchSnapshotSettings();

    if (!isEmpty(this.props.location.search)) {
      if (this.props.location.search === "?configure=true") {
        this.setState({ showConfigureSnapshotsModal: true });
      }
    }
  }

  componentDidUpdate(_, lastState) {
    if (this.state.snapshotSettings !== lastState.snapshotSettings && this.state.snapshotSettings) {
      if (!this.state.snapshotSettings?.veleroVersion) {
        this.props.history.replace("/snapshots/settings?configure=true");
        this.setState({ showConfigureSnapshotsModal: true });
      }
    }
  }

  toggleSnapshotView = (isEmptyView) => {
    this.setState({ toggleSnapshotView: !this.state.toggleSnapshotView, isEmptyView: isEmptyView ? isEmptyView : false });
  }

  updateSettings = (payload) => {
    this.setState({ updatingSettings: true, updateErrorMsg: "" });

    fetch(`${window.env.API_ENDPOINT}/snapshots/settings`, {
      method: "PUT",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify(payload)
    })
      .then(async (res) => {
        if (!res.ok && res.status === 409) {
          const result = await res.json();
          if (result.kotsadmRequiresVeleroAccess) {
            this.setState({
              updatingSettings: false,
            });
            this.openConfigureSnapshotsMinimalRBACModal(result.kotsadmRequiresVeleroAccess, result.kotsadmNamespace);
            return;
          }
          this.setState({
            updatingSettings: false,
            showResetFileSystemWarningModal: true,
            resetFileSystemWarningMessage: result.error,
          })
          return;
        }

        const settingsResponse = await res.json();
        if (!res.ok) {
          this.setState({
            updatingSettings: false,
            updateErrorMsg: settingsResponse.error
          })
          return;
        }

        if (settingsResponse.success) {
          this.setState({
            snapshotSettings: settingsResponse,
            updatingSettings: false,
            updateConfirm: true,
            updateErrorMsg: ""
          });
          setTimeout(() => {
            this.setState({ updateConfirm: false })
          }, 3000);
        } else {
          this.setState({
            updatingSettings: false,
            updateErrorMsg: settingsResponse.error
          })
        }
      })
      .catch((err) => {
        console.error(err);
        this.setState({
          updatingSettings: false,
          updateErrorMsg: "Something went wrong, please try again."
        });
      });
  }

  renderNotVeleroMessage = () => {
    return <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginTop--12">Not able to find Velero</p>
  }

  openConfigureSnapshotsMinimalRBACModal = (kotsadmRequiresVeleroAccess, minimalRBACKotsadmNamespace) => {
    this.setState({ showConfigureSnapshotsModal: true, kotsadmRequiresVeleroAccess, minimalRBACKotsadmNamespace }, () => {
      this.props.history.replace("/snapshots/settings?configure=true");
    });
  }

  toggleConfigureSnapshotsModal = () => {
    if (this.state.showConfigureSnapshotsModal) {
      this.setState({ showConfigureSnapshotsModal: false }, () => {
        this.props.history.replace("/snapshots/settings");
      });
    } else {
      this.setState({ showConfigureSnapshotsModal: true }, () => {
        this.props.history.replace("/snapshots/settings?configure=true");
      });
    }
  };

  hideResetFileSystemWarningModal = () => {
    this.setState({ showResetFileSystemWarningModal: false });
  }

  render() {
    const { isLoadingSnapshotSettings, snapshotSettings, hideCheckVeleroButton, updateConfirm, updatingSettings, updateErrorMsg, isEmptyView } = this.state;
    const isLicenseUpload = !!this.props.history.location.search;

    if (isLoadingSnapshotSettings) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    const isVeleroCorrectVersion = snapshotSettings?.isVeleroRunning && snapshotSettings?.veleroVersion.includes("v1.5");

    return (
      <div className="flex1 flex-column u-overflow--auto">
        <Helmet>
          <title>Snapshot Settings</title>
        </Helmet>
        {!isVeleroCorrectVersion ?
          <div className="VeleroWarningBlock">
            <span className="icon small-warning-icon" />
            <p> To use snapshots reliably you have to install velero version 1.5.1 </p>
          </div>
          : null}
        <div className="container flex-column flex1u-paddingTop--30 u-paddingBottom--20 u-marginTop--10 alignItems--center">
          <SnapshotStorageDestination
            snapshotSettings={snapshotSettings}
            updateSettings={this.updateSettings}
            fetchSnapshotSettings={this.fetchSnapshotSettings}
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
            openConfigureSnapshotsMinimalRBACModal={this.openConfigureSnapshotsMinimalRBACModal}
            kotsadmRequiresVeleroAccess={this.state.kotsadmRequiresVeleroAccess}
            minimalRBACKotsadmNamespace={this.state.minimalRBACKotsadmNamespace}
            showResetFileSystemWarningModal={this.state.showResetFileSystemWarningModal}
            resetFileSystemWarningMessage={this.state.resetFileSystemWarningMessage}
            hideResetFileSystemWarningModal={this.hideResetFileSystemWarningModal}
            isKurlEnabled={this.props.isKurlEnabled}
            apps={this.props.apps}
          />
        </div>
      </div>
    );
  }
}

export default withRouter(SnapshotSettings);
