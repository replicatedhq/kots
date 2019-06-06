import * as React from "react";
import { withRouter } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import WatchContributors from "./WatchContributors";
import Modal from "react-modal";
import { Utilities } from "../../utilities/utilities";
import { updateWatch, deleteWatch } from "../../mutations/WatchMutations";
import Select from "react-select";
import Loader from "../shared/Loader";

class DetailPageApplication extends React.Component {

    state = {
      appName: "",
      iconUri: "",
      nameLoading: false,
      iconLoading: false,
      showConfirmDelete: false,
      confirmAppName: "",
      deleteAppLoading: false,
      confirmDeleteErr: false,
      isDownloadingAssets: false,
      isDownloadingMidstreamAssets: false,
      downloadCluster: {
        value: "",
        label: "Select a cluster",
        watchId: ""
      }
    }

  onFormChange = (event) => {
    const { value, name } = event.target;
    this.setState({
      [name]: value
    });
  }

  setWatchState = (watch) => {
    this.setState({
      appName: watch.watchName,
      iconUri: watch.watchIcon
    });
  }

  updateName = async () => {
    const { appName } = this.state;
    const { watch, updateCallback } = this.props;
    this.setState({ nameLoading: true })
    await this.props.updateWatch(watch.id, appName, null)
      .then(() => {
        this.setState({ nameLoading: false });
        if (updateCallback && typeof updateCallback === "function") {
          updateCallback();
        }
      }).catch(() => this.setState({ nameLoading: false }));
  }

  updateIcon = async () => {
    const { iconUri } = this.state;
    const { watch, updateCallback } = this.props;
    this.setState({ iconLoading: true });
    await this.props.updateWatch(watch.id, null, iconUri)
      .then(() => {
        this.setState({ iconLoading: false });
        if (updateCallback && typeof updateCallback === "function") {
          updateCallback();
        }
      }).catch(() => this.setState({ iconLoading: false }));
  }

  onDownloadClusterChange = (selectedOption) => {
    this.setState({ downloadCluster: selectedOption });
  }

  downloadAssetsForCluster = async () => {
    const { downloadCluster } = this.state;
    this.setState({ isDownloadingAssets: true });
    await Utilities.handleDownload(downloadCluster.watchId);
    this.setState({ isDownloadingAssets: false });
  }

  downloadAssetsForMidsttream = async (watchId) => {
    this.setState({ isDownloadingMidstreamAssets: true });
    await Utilities.handleDownload(watchId);
    this.setState({ isDownloadingMidstreamAssets: false });
  }

  handleEnterPress = (e) => {
    if (e.charCode === 13) {
      this.handleDeleteApp();
    }
  }

  toggleConfirmDelete = () => {
    const { watch } = this.props;
    const childWatchIds = this.state.showConfirmDelete ? [] : watch.watches.map((w) => w.id);
    this.setState({
      showConfirmDelete: !this.state.showConfirmDelete,
      childWatchIds
    });
  }

  handleDeleteApp = async () => {
    const { watch } = this.props;
    const { confirmAppName, childWatchIds } = this.state;
    const canDelete = confirmAppName === watch.watchName;
    this.setState({ confirmDeleteErr: false });
    if (canDelete) {
      this.setState({ deleteAppLoading: true });
      await this.props.deleteWatch(watch.id, childWatchIds)
        .then(() => this.props.history.push("/watches"))
        .catch(() => this.setState({ deleteAppLoading: false }));
    } else {
      this.setState({ confirmDeleteErr: true });
    }
  }

  componentDidUpdate(lastProps) {
    const { watch } = this.props;
    if (watch !== lastProps.watch && watch) {
      this.setWatchState(watch)
    }
  }

  componentDidMount() {
    const { watch } = this.props;
    if (watch) {
      this.setWatchState(watch);
    }
  }

  render() {
    const { isDownloadingAssets, isDownloadingMidstreamAssets } = this.state;
    const { watch, updateCallback } = this.props;
    const childWatches = watch.watches;
    let options = [];
    if (watch.cluster) {
      options = [{
        value: watch.cluster.id,
        label: watch.cluster.title,
        watchId: watch.id
      }]
    } else {
      options = childWatches && childWatches.map((childWatch) => {
        const childCluster = childWatch.cluster;
        if (childCluster) {
          return ({
            value: childCluster.id,
            label: childCluster.title,
            watchId: childWatch.id
          });
        } else {
          return {}
        }
      });
    }

    return (
      <div className="DetailPageApplication--wrapper flex-column flex1 container alignItems--center u-overflow--auto u-paddingBottom--20">
        <div className="DetailPageApplication flex flex1">
          <div className="flex1 flex-column u-paddingRight--30">
            <div className="flex-column">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">App name</p>
              <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--10">You can name your app whatever you want.</p>
              <div className="flex">
                <input
                  type="text"
                  className="Input"
                  placeholder="What is your application called?"
                  value={this.state.appName || ""}
                  name="appName"
                  onChange={this.onFormChange}
                />
                <div className="u-marginLeft--10">
                  <button type="button" className="btn secondary" onClick={this.updateName} disabled={this.state.nameLoading}>{this.state.nameLoading ? "Saving" : "Save"}</button>
                </div>
              </div>
            </div>
            <div className="u-marginTop--30">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">App icon</p>
              <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--10">Link to any URI for an app icon.</p>
              <div className="flex">
                <input
                  type="text"
                  className="Input"
                  placeholder="Add a URL"
                  value={this.state.iconUri || ""}
                  name="iconUri"
                  onChange={this.onFormChange}
                />
                <div className="u-marginLeft--10">
                  <button type="button" className="btn secondary" onClick={this.updateIcon} disabled={this.state.iconLoading}>{this.state.iconLoading ? "Saving" : "Save"}</button>
                </div>
              </div>
            </div>
            {!watch.cluster &&
              <div className="u-marginTop--30">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">Download assets for {watch.watchName}</p>
                <p className="u-fontWeight--medium u-fontSize--small u-color--dustyGray u-marginTop--5 u-lineHeight--medium">This will download the YAML for your mid-stream watch.</p>
                {isDownloadingMidstreamAssets ?
                  <div className="flex-column flex1 alignItems--center justifyContent--center">
                    <Loader size="60" />
                  </div>
                  :
                  <div className="u-marginTop--10">
                    <button onClick={() => this.downloadAssetsForMidsttream(watch.id)} className="btn green secondary">Download generated YAML</button>
                  </div>
                }
              </div>
            }
            <div className="u-marginTop--30">
              {options.length > 0 && (
                <>
                  <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">Download assets from a deployment cluster</p>
                  <p className="u-fontWeight--medium u-fontSize--small u-color--dustyGray u-marginTop--5 u-lineHeight--medium">Select the cluster you would like to download your Ship YAML assets from.</p>
                  <div className="u-marginTop--10 flex">
                    <div className="flex1">
                      <Select
                        className="replicated-select-container"
                        classNamePrefix="replicated-select"
                        options={options}
                        getOptionLabel={(downloadCluster) => downloadCluster.label}
                        value={this.state.downloadCluster}
                        onChange={this.onDownloadClusterChange}
                        isOptionSelected={(option) => { option.value === this.state.downloadCluster.value }}
                      />
                    </div>
                    <div className="flex1"></div>
                  </div>
                  {isDownloadingAssets ?
                    <div className="flex-column flex1 alignItems--center justifyContent--center">
                      <Loader size="60" />
                    </div>
                    :
                    <div className="u-marginTop--10">
                      <button disabled={this.state.downloadCluster.value === ""} onClick={() => this.downloadAssetsForCluster()} className="btn green secondary">Download generated YAML</button>
                    </div>
                  }
                </>
              )}
            </div>
            <div className="u-marginTop--30 u-borderTop--gray u-paddingTop--30">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">Delete application</p>
              <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--10">Removing {this.state.appName} will permanently delete all data and integrations associated with it and will not be&nbsp;recoverable.</p>
              <div className="u-marginTop--10">
                <button type="button" className="btn primary red" onClick={this.toggleConfirmDelete}>Delete application</button>
              </div>
            </div>
          </div>
          <div className="flex1 flex-column detail-right-sidebar u-paddingLeft--30">
            <div>
                <p className="uppercase-title">Current Version</p>
                <p className="u-fontSize--jumbo2 u-fontWeight--bold u-color--tuna">
                  {watch.currentVersion.title}
                </p>
            </div>
            <WatchContributors
              title="contributors"
              className="u-marginTop--30"
              contributors={watch.contributors || []}
              watchName={watch.watchName}
              watchId={watch.id}
              watchCallback={updateCallback}
              slug={watch.slug}
            />
          </div>
        </div>
        <Modal
          isOpen={this.state.showConfirmDelete}
          onRequestClose={this.toggleConfirmDelete}
          shouldReturnFocusAfterClose={false}
          contentLabel="Modal"
          ariaHideApp={false}
          className="Modal SmallSize"
        >
          <div className="Modal-body flex-column flex1">
            <h2 className="u-fontSize--largest u-fontWeight--bold u-color--tuna u-marginBottom--10">Are you sure you want to delete {this.state.appName}?</h2>
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">To delete {this.state.appName}, type its name in the field below</p>
            <input
              type="text"
              className="Input"
              placeholder="Type the app name here"
              value={this.state.confirmAppName}
              onKeyPress={this.handleEnterPress}
              name="confirmAppName"
              onChange={this.onFormChange}
              autoFocus
            />
            {this.state.confirmDeleteErr && <p className="u-fontSize--small u-color--chestnut u-marginTop--10">Names did not match</p>}
            <div className="u-marginTop--20 flex justifyContent--flexEnd">
              <button type="button" className="btn primary red" onClick={this.handleDeleteApp} disabled={this.state.deleteAppLoading}>{this.state.deleteAppLoading ? "Deleting" : "Delete"}</button>
            </div>
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
  graphql(deleteWatch, {
    props: ({ mutate }) => ({
      deleteWatch: (watchId, childWatchIds) => mutate({ variables: { watchId, childWatchIds } })
    })
  })
)(DetailPageApplication);
