import React, { Component } from "react";
import Helmet from "react-helmet";
import { withRouter } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import Modal from "react-modal";
import PaperIcon from "../shared/PaperIcon";
import DashboardCard from "./DashboardCard";

import {
  updateWatch,
  deleteWatch,
} from "@src/mutations/WatchMutations";

import { checkForKotsUpdates } from "../../mutations/AppsMutations";

import { deleteKotsApp, updateKotsApp } from "@src/mutations/AppsMutations";

import "../../scss/components/watches/Dashboard.scss";

class Dashboard extends Component {

  state = {
    appName: "",
    iconUri: "",
    showEditModal: false,
    editWatchLoading: false,
    currentVersion: {},
    downstreams: [],
    checkingForUpdates: false,
    checkingUpdateText: "Checking for updates",
    errorCheckingUpdate: false
  }

  updateWatchInfo = async e => {
    e.preventDefault();
    const { appName, iconUri } = this.state;
    const { app, updateCallback, updateKotsApp, refetchListApps } = this.props;
    this.setState({ editWatchLoading: true });

    await updateKotsApp(app.id, appName, iconUri).catch(error => {
      console.error("[DetailPageApplication]: Error updating App info: ", error);
      this.setState({
        editWatchLoading: false
      });
    });


    await refetchListApps();

    this.setState({
      editWatchLoading: false,
      showEditModal: false
    });

    if (updateCallback && typeof updateCallback === "function") {
      updateCallback();
    }
  }

  handleEnterPress = (e) => {
    if (e.charCode === 13) {
      this.handleDeleteApp();
    }
  }

  onFormChange = (event) => {
    const { value, name } = event.target;
    this.setState({
      [name]: value
    });
  }

  toggleEditModal = () => {
    const { showEditModal } = this.state;
    this.setState({
      showEditModal: !showEditModal
    });
  }

  setWatchState = (app) => {
    this.setState({
      appName: app.name,
      iconUri: app.iconUri,
      currentVersion: app.downstreams[0].currentVersion,
      downstreams: app.downstreams[0]
    });
  }


  toggleEditModal = () => {
    const { showEditModal } = this.state;
    this.setState({
      showEditModal: !showEditModal
    });
  }

  componentDidUpdate(lastProps) {
    const { app } = this.props;
    if (app !== lastProps.app && app) {
      this.setWatchState(app)
    }
  }

  componentDidMount() {
    const { app } = this.props;
    if (app) {
      this.setWatchState(app);
    }
  }

  onCheckForUpdates = async () => {
    const { client, app } = this.props;

    this.setState({ checkingForUpdates: true });

    this.loadingTextTimer = setTimeout(() => {
      this.setState({ checkingUpdateText: "Almost there, hold tight..." });
    }, 10000);

    await client.mutate({
      mutation: checkForKotsUpdates,
      variables: {
        appId: app.id,
      }
    }).catch(() => {
      this.setState({ errorCheckingUpdate: true });
    }).finally(() => {
      clearTimeout(this.loadingTextTimer);
      this.setState({
        checkingForUpdates: false,
        checkingUpdateText: "Checking for updates"
      });
      if (this.props.updateCallback) {
        this.props.updateCallback();
      }
    });
  }

  onUploadNewVersion = () => {
    this.props.history.push(`/${this.props.match.params.slug}/airgap`);
  }

  render() {
    const { appName, iconUri, currentVersion, downstreams, checkingForUpdates, checkingUpdateText, errorCheckingUpdate } = this.state;
    const { app } = this.props;
    const isAirgap = app.isAirgap;


    return (
      <div className="flex-column flex1 u-position--relative u-overflow--auto u-padding--20">
        <Helmet>
          <title>{appName}</title>
        </Helmet>
        <div className="Dashboard flex flex-auto justifyContent--center alignSelf--center alignItems--center">
          <div className="flex1 flex-column">
            <div className="flex flex1">
              <div className="flex flex1 alignItems--center u-marginLeft--30">
              <div className="flex flex-auto">
                <div
                  style={{ backgroundImage: `url(${iconUri})` }}
                  className="Dashboard--appIcon u-position--relative">
                  <PaperIcon
                    className="u-position--absolute"
                    height="25px"
                    width="25px"
                    iconClass="edit-icon"
                    onClick={this.toggleEditModal}
                  />
                </div>
              </div>
                <p className="u-fontSize--30 u-color--tuna u-fontWeight--bold u-marginLeft--20">{appName}</p>
              </div>
            </div>
            <div className="u-marginTop--30 u-paddingTop--10 flex-auto flex flexWrap--wrap u-width--full alignItems--center justifyContent--center">
              <DashboardCard
                cardName="Application"
                application={true}
                cardIcon="applicationIcon"
              />
              <DashboardCard
                cardName={`Version: ${currentVersion.title}`}
                cardIcon="versionIcon"
                versionHistory={true}
                currentVersion={currentVersion}
                downstreams={downstreams}
                isAirgap={isAirgap}
                app={app}
                url={this.props.match.url}
                checkingForUpdates={checkingForUpdates}
                checkingUpdateText={checkingUpdateText}
                errorCheckingUpdate={errorCheckingUpdate}
                onCheckForUpdates={() => this.onCheckForUpdates()}
                onUploadNewVersion={() => this.onUploadNewVersion()}
              />
              <DashboardCard
                cardName="License"
                cardIcon="licenseIcon"
                license={true}
              />
            </div>
          </div>
        </div>
        <Modal
          isOpen={this.state.showEditModal}
          onRequestClose={this.toggleEditModal}
          contentLabel="Yes"
          ariaHideApp={false}
          className="Modal SmallSize EditWatchModal">
          <div className="Modal-body flex-column flex1">
            <h2 className="u-fontSize--largest u-fontWeight--bold u-color--tuna u-marginBottom--10">Edit Application</h2>
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">You can edit the name and icon of your application</p>
            <h3 className="u-fontSize--normal u-fontWeight--bold u-color--tuna u-marginBottom--10">Application Name</h3>
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">This name will be shown throughout this dashboard.</p>
            <form className="EditWatchForm flex-column" onSubmit={this.updateWatchInfo}>
              <input
                type="text"
                className="Input u-marginBottom--20"
                placeholder="Type the app name here"
                value={this.state.appName}
                onKeyPress={this.handleEnterPress}
                name="appName"
                onChange={this.onFormChange}
              />
              <h3 className="u-fontSize--normal u-fontWeight--bold u-color--tuna u-marginBottom--10">Application Icon</h3>
              <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Provide a link to a URI to use as your app icon.</p>
              <input
                type="text"
                className="Input u-marginBotton--20"
                placeholder="Enter the link here"
                value={this.state.iconUri}
                onKeyPress={this.handleEnterPress}
                name="iconUri"
                onChange={this.onFormChange}
              />
              <div className="flex justifyContent--flexEnd u-marginTop--20">
                <button
                  type="button"
                  onClick={this.toggleEditModal}
                  className="btn secondary force-gray u-marginRight--20">
                  Cancel
              </button>
                <button
                  type="submit"
                  className="btn secondary green">
                  {
                    this.state.editWatchLoading
                      ? "Saving"
                      : "Save Application Details"
                  }
                </button>
              </div>
            </form>
          </div>
        </Modal>
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  graphql(updateWatch, {
    props: ({ mutate }) => ({
      updateWatch: (watchId, watchName, iconUri) => mutate({ variables: { watchId, watchName, iconUri } })
    })
  }),
  graphql(updateKotsApp, {
    props: ({ mutate }) => ({
      updateKotsApp: (appId, appName, iconUri) => mutate({ variables: { appId, appName, iconUri } })
    })
  }),
  graphql(deleteWatch, {
    props: ({ mutate }) => ({
      deleteWatch: (watchId, childWatchIds) => mutate({ variables: { watchId, childWatchIds } })
    })
  }),
  graphql(deleteKotsApp, {
    props: ({ mutate }) => ({
      deleteKotsApp: (slug) => mutate({ variables: { slug } })
    })
  })
)(Dashboard);
