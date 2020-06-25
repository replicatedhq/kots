import React, { Component } from "react";
import { compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom"
import Helmet from "react-helmet";

import Loader from "../shared/Loader";
import SnapshotStorageDestination from "./SnapshotStorageDestination";
import ConfigureSnapshots from "./ConfigureSnapshots";

import "../../scss/components/shared/SnapshotForm.scss";
import { Utilities } from "../../utilities/utilities";


class Snapshots extends Component {
  state = {
    snapshotSettings: null,
    isLoadingSnapshotSettings: true,
    snapshotSettingsErr: false,
    snapshotSettingsErrMsg: "",
    toggleSnapshotView: false,
    hideCheckVeleroButton: false,
    updateConfirm: false,
    updatingSettings: false,
    updateErrorMsg: ""
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

  componentDidMount = () => {
    this.fetchSnapshotSettings();
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


  render() {
    const { isLoadingSnapshotSettings, toggleSnapshotView, snapshotSettings, hideCheckVeleroButton, updateConfirm, updatingSettings, updateErrorMsg, isEmptyView } = this.state;
    const isLicenseUpload = !!this.props.history.location.search;

    if (isLoadingSnapshotSettings) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    return (
      <div className="container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20 u-marginTop--10 alignItems--center">
        <Helmet>
          <title>Snapshots</title>
        </Helmet>
        {(toggleSnapshotView || (snapshotSettings?.veleroVersion !== "")) ?
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
            isLicenseUpload={isLicenseUpload} /> :
          <ConfigureSnapshots
            snapshotSettings={snapshotSettings}
            fetchSnapshotSettings={this.fetchSnapshotSettings}
            renderNotVeleroMessage={this.renderNotVeleroMessage}
            hideCheckVeleroButton={hideCheckVeleroButton} />}
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
)(Snapshots);
