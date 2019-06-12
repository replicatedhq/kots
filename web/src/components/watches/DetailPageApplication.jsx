import * as React from "react";
import { withRouter, Link } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import WatchContributors from "./WatchContributors";
import Modal from "react-modal";
import Loader from "../shared/Loader";
import {
  Utilities,
  getClusterType,
  getAppData,
  getReadableLicenseType
} from "@src/utilities/utilities";
import {
  updateWatch,
  deleteWatch,
  createEditSession
 } from "@src/mutations/WatchMutations";

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
      },
      errorCustomizingCluster: false
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

  handleClusterMangeClick = (watchId) => {
    this.setState({ errorCustomizingCluster: false, [`preparing${watchId}`]: true });
    this.props.client.mutate({
      mutation: createEditSession,
      variables: {
        watchId: watchId,
      },
    })
    .then(({ data }) => {
      this.setState({ [`preparing${watchId}`]: false });
      this.props.onActiveInitSession(data.createEditSession.id);
      this.props.history.push("/ship/edit");
    })
    .catch(() => this.setState({ errorCustomizingCluster: true, [`preparing${watchId}`]: false }));
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
    const { watch, updateCallback } = this.props;
    const childWatches = watch.watches;
    const appMeta = getAppData(watch.metadata);

    // TODO: We shuold probably return something different if it never expires to avoid this hack string check.
    const expDate = appMeta.license.expiresAt === "0001-01-01T00:00:00Z" ? "Never" : Utilities.dateFormat(appMeta.license.expiresAt, "MMM D, YYYY");
    return (
      <div className="DetailPageApplication--wrapper flex-column flex1 container alignItems--center u-overflow--auto u-paddingBottom--20">
        <div className="DetailPageApplication flex flex1">
          <div className="flex1 flex-column u-paddingRight--30">
            <div className="flex">
              <div className="flex flex-auto">
                <span style={{ backgroundImage: `url(${watch.watchIcon})`}} className="DetailPageApplication--appIcon"></span>
              </div>
              <div className="flex-column flex1 u-marginLeft--10 u-paddingLeft--5">
                <p className="u-fontSize--30 u-color--tuna u-fontWeight--bold">{watch.watchName}</p>
                {appMeta.applicationType === "replicated.app" &&
                  <div className="u-marginTop--10 flex-column">
                    <div className="flex u-color--dustyGray u-fontWeight--medium u-fontSize--normal">
                      <span className="u-marginRight--30">Expires: <span className="u-fontWeight--bold u-color--tundora">{expDate}</span></span>
                      <span>Type: <span className="u-fontWeight--bold u-color--tundora">{getReadableLicenseType(appMeta.license.type)}</span></span>
                    </div>
                    <Link to={`/watch/${watch.slug}/license`} className="u-marginTop--10 u-fontSize--small replicated-link">License details</Link>
                  </div>
                }
              </div>
            </div>

            <div className="u-marginTop--30 u-paddingTop--10">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">Downstreams</p>
              <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--10">Your app can be deployed to as many clusters as you would like. Each cluster can have it’s own configuration and patches for your kubernetes YAML.</p>
              <div className="flex flex-column u-marginTop--10 u-paddingTop--5">
                {childWatches && childWatches.map((childWatch) => {
                  const childCluster = childWatch.cluster;
                  const clusterType = getClusterType(childCluster.gitOpsRef);
                  if (childCluster) {
                    return (
                      <div key={childCluster.id} className="DetailPage--downstreamRow flex">
                        <div className="flex1 flex alignItems--center">
                          <span className={`icon clusterType ${clusterType}`}></span>
                          <span className="u-fontSize--normal u-color--tundora u-fontWeight--bold u-marginLeft--5">{childCluster.title}</span>
                        </div>
                        <div className="flex1">
                        </div>
                        <div className="flex-auto">
                          {this.state[`preparing${childWatch.id}`] ?
                            <Loader size="16"/>
                          :
                            <span onClick={() => this.handleClusterMangeClick(childWatch.id)} className="u-fontSize--small replicated-link">Customize</span>
                          }
                        </div>
                      </div>
                    );
                  }
                })}
              </div>
              <div className="u-marginTop--10 u-paddingTop--5">
                <Link to={`/watch/${watch.slug}/downstreams`} className="btn secondary">See downstreams</Link>
              </div>
            </div>

            <div className="u-marginTop--30 u-paddingTop--10 flex">
              <div className="flex1 u-paddingRight--15">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">Get help with your application</p>
                <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--10">Generate a support bundle for your application to send to the vendor.</p>
                <div className="u-marginTop--10">
                  <Link to={`/watch/${watch.slug}/troubleshoot`} className="btn secondary">Generate a support bundle</Link>
                </div>
              </div>
              <div className="flex1 u-paddingLeft--15">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">Application config</p>
                <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--10">Quickly see a ready-only preview of your application config for reference.</p>
                <div className="u-marginTop--10">
                  <Link to={`/watch/${watch.slug}/config`} className="btn secondary">See application config</Link>
                </div>
              </div>
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
