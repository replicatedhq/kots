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
    configureSnapshotsModal: false
  };

  fetchSnapshotSettings = (isCheckForVelero) => {
    this.setState({
      isLoadingSnapshotSettings: true,
      snapshotSettingsErr: false,
      snapshotSettingsErrMsg: "",
      hideCheckVeleroButton: isCheckForVelero ? true : false
    });

    fetch(`${window.env.API_ENDPOINT}/snapshots/settings`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    })
      .then(res => res.json())
      .then(result => {
        this.setState({
          snapshotSettings: result,
          isLoadingSnapshotSettings: false,
          snapshotSettingsErr: false,
          snapshotSettingsErrMsg: "",
        })
        if (result.veleroVersion === "") {
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
      this.setState({ configureSnapshotsModal: true });
    }
  }

  componentDidUpdate(lastProps, lastState) {
    if (this.state.snapshotSettings !== lastState.snapshotSettings && this.state.snapshotSettings) {
      if (this.state.snapshotSettings?.veleroVersion === "") {
        this.props.history.push("/snapshots/settings?configure=true");
        this.setState({ configureSnapshotsModal: true });
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
          updatingSettings: false
        });
      });
  }

  renderNotVeleroMessage = () => {
    return <p className="u-color--chestnut u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-marginTop--12">Not able to find Velero</p>
  }

  toggleConfigureModal = () => {
    if (this.state.configureSnapshotsModal) {
      this.setState({ configureSnapshotsModal: false }, () => {
        this.props.history.push("/snapshots/settings");
      });
    } else {
      this.setState({ configureSnapshotsModal: true }, () => {
        this.props.history.push("/snapshots/settings?configure=true");
      });
    }
  };


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
            <span className="icon snapshot-warning-icon" />
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
            configureSnapshotsModal={this.state.configureSnapshotsModal}
            toggleConfigureModal={this.toggleConfigureModal}
            isKurlEnabled={this.props.isKurlEnabled} />
        </div>
      </div>
    );
  }
}

export default withRouter(SnapshotSettings);
