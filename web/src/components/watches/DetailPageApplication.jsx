import React, { Component } from "react";
import Helmet from "react-helmet";
import { withRouter, Link } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import WatchContributors from "./WatchContributors";
import truncateMiddle from "truncate-middle";
import Modal from "react-modal";
import Loader from "../shared/Loader";
import PaperIcon from "../shared/PaperIcon";
import { isSingleTenant } from "@src/utilities/utilities";

import {
  Utilities,
  getClusterType,
  getWatchMetadata,
  getReadableLicenseType,
  getLicenseExpiryDate,
  getWatchLicenseFromState,
  isKotsApplication,
} from "@src/utilities/utilities";

import {
  updateWatch,
  deleteWatch,
  createEditSession,
 } from "@src/mutations/WatchMutations";

 import { deleteKotsApp, updateKotsApp } from "@src/mutations/AppsMutations";
 import isEmpty from "lodash/isEmpty";
 import { getWatchLicense } from "@src/queries/WatchQueries";

class DetailPageApplication extends Component {

    state = {
      appName: "",
      iconUri: "",
      editWatchLoading: false,
      showConfirmDelete: false,
      showEditModal: false,
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
      errorCustomizingCluster: false,
      preparingAppUpdate: false,
      watchLicense: null,
    }

  onFormChange = (event) => {
    const { value, name } = event.target;
    this.setState({
      [name]: value
    });
  }

  setWatchState = (watch) => {
    const isKotsApp = isKotsApplication(watch);
    this.setState({
      appName: isKotsApp ? watch.name : watch.watchName,
      iconUri: isKotsApp ? watch.iconUri : watch.watchIcon
    });
  }

  updateWatchInfo = async e => {
    e.preventDefault();
    const { appName, iconUri } = this.state;
    const { watch, updateCallback, updateWatch, updateKotsApp, refetchListApps } = this.props;
    this.setState({ editWatchLoading: true });

    const isKotsApp = isKotsApplication(watch);
    if (isKotsApp) {
      await updateKotsApp(watch.id, appName, iconUri).catch(error => {
        console.error("[DetailPageApplication]: Error updating App info: ", error);
        this.setState({
          editWatchLoading: false
        });
      });
    } else {
      await updateWatch(watch.id, appName, iconUri).catch(error => {
        console.error("[DetailPageApplication]: Error updating Watch info: ", error);
        this.setState({
          editWatchLoading: false
        });
      });
    }

    await refetchListApps();

    this.setState({
      editWatchLoading: false,
      showEditModal: false
    });

    if (updateCallback && typeof updateCallback === "function") {
      updateCallback();
    }
  }

  toggleEditModal = () => {
    const { showEditModal } = this.state;
    this.setState({
      showEditModal: !showEditModal
    });
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
    const isKotsApp = isKotsApplication(watch);
    const childWatchIds = this.state.showConfirmDelete || isKotsApp ? [] : watch.watches.map((w) => w.id);
    this.setState({
      showConfirmDelete: !this.state.showConfirmDelete,
      childWatchIds
    });
  }

  handleDeleteApp = async () => {
    const { watch } = this.props;
    const { confirmAppName, childWatchIds } = this.state;
    const isKotsApp = isKotsApplication(watch);
    const watchName = isKotsApp ? watch.name : watch.watchName;
    const canDelete = confirmAppName === watchName;
    this.setState({ confirmDeleteErr: false });
    if (canDelete) {
      this.setState({ deleteAppLoading: true });
      if (isKotsApp) {
        try {
          await this.props.deleteKotsApp(watch.slug);
          await this.props.refetchListApps();
          this.props.history.push("/watches");
        } catch (error) {
          this.setState({ deleteAppLoading: false })
        }
      } else {
        try {
          await this.props.deleteWatch(watch.id, childWatchIds)
          await this.props.refetchListApps();
          this.props.history.push("/watches");
        } catch (error) {
          this.setState({ deleteAppLoading: false })
        }
      }
    } else {
      this.setState({ confirmDeleteErr: true });
    }
  }

  navigateToFiles = (watch) => {
    this.props.history.push(`/app/${watch.slug}/tree/${watch.currentSequence}`)
  }

  handleEditWatchClick = (watch) => {
    const isCluster = watch.cluster;
    if (isCluster) {
      this.setState({ errorCustomizingCluster: false, [`preparing${watch.id}`]: true });
    } else {
      this.setState({ preparingAppUpdate: true });
    }

    this.props.client.mutate({
      mutation: createEditSession,
      variables: {
        watchId: watch.id,
      },
    })
    .then(({ data }) => {
      if (isCluster) {
        this.setState({ [`preparing${watch.id}`]: false });
      } else {
        this.setState({ preparingAppUpdate: false });
      }
      this.props.onActiveInitSession(data.createEditSession.id);
      this.props.history.push("/ship/edit");
    })
    .catch(() => {
      if (isCluster) {
        this.setState({ errorCustomizingCluster: true, [`preparing${watch.id}`]: false })
      } else {
        this.setState({ preparingAppUpdate: false });
      }
    });
  }

  componentDidUpdate(lastProps) {
    const { watch } = this.props;
    if (watch !== lastProps.watch && watch) {
      this.setWatchState(watch)
    }
    const isKotsApp = isKotsApplication(watch);
    if (!isKotsApp) {
      // current license info
      if (this.props.getWatchLicense?.error && !this.state.watchLicense) {
        // no current license found in db, construct from stateJSON
        const watchLicense = getWatchLicenseFromState(this.props.watch);
        this.setState({ watchLicense });
      } else if (this.props.getWatchLicense !== lastProps.getWatchLicense && this.props.getWatchLicense) {
        const { getWatchLicense } = this.props.getWatchLicense;
        if (getWatchLicense) {
          this.setState({ watchLicense: getWatchLicense });
        }
      }
    }
  }

  componentDidMount() {
    const { watch } = this.props;
    if (watch) {
      this.setWatchState(watch);
    }
    const isKotsApp = isKotsApplication(watch);
    if (!isKotsApp && this.props.getWatchLicense) {
      const { getWatchLicense } = this.props.getWatchLicense;
      if (getWatchLicense) {
        this.setState({ watchLicense: getWatchLicense });
      }
    }
  }

  render() {
    const { watch, updateCallback } = this.props;
    const { preparingAppUpdate, watchLicense } = this.state;
    const appMeta = getWatchMetadata(watch.metadata);
    const isKotsApp = isKotsApplication(watch);
    const childWatches = isKotsApp ? watch.downstreams : watch.watches;

    return (
      <div className="DetailPageApplication--wrapper container flex-column flex1 alignItems--center u-overflow--auto u-paddingTop--30 u-paddingBottom--20">
        <Helmet>
          <title>{`${isKotsApp ? watch.name : watch.watchName} Config Overview`}</title>
        </Helmet>
        <div className="DetailPageApplication flex flex1">
          <div className="flex1 flex-column u-paddingRight--30">
            <div className="flex">
              <div className="flex flex-auto">
                <div
                  style={{ backgroundImage: `url(${isKotsApp ? watch.iconUri : watch.watchIcon})`}}
                  className="DetailPageApplication--appIcon u-position--relative">
                  <PaperIcon
                    className="u-position--absolute"
                    height="25px"
                    width="25px"
                    iconClass="edit-icon"
                    onClick={this.toggleEditModal}
                  />
                </div>
              </div>
              <div className="flex-column flex1 justifyContent--center u-marginLeft--10 u-paddingLeft--5">
                <p className="u-fontSize--30 u-color--tuna u-fontWeight--bold">{isKotsApp ? watch.name : watch.watchName}</p>
                {!isKotsApp && !isEmpty(appMeta) && appMeta.applicationType === "replicated.app" &&
                  <div className="u-marginTop--10 flex-column">
                    {watchLicense
                      ?
                      <div className="flex u-color--dustyGray u-fontWeight--medium u-fontSize--normal">
                        <span className="u-marginRight--30">Expires: <span className="u-fontWeight--bold u-color--tundora">{getLicenseExpiryDate(watchLicense)}</span></span>
                        <span>Type: <span className="u-fontWeight--bold u-color--tundora">{getReadableLicenseType(watchLicense.type)}</span></span>
                      </div>
                      :
                      <Loader size="12" />
                    }
                    <Link to={`/watch/${watch.slug}/license`} className="u-marginTop--10 u-fontSize--small replicated-link">License details</Link>
                  </div>
                }
              </div>
            </div>
            {!watch.cluster &&
              <div className="u-marginTop--30">
                <div className="midstream-banner">
                  <p className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-color--nevada">This is a “Midstream”. Midstreams are a single place that you can apply patches globally.</p>
                </div>
              </div>
            }

            {isKotsApp &&
              <div className="u-marginTop--30 u-paddingTop--10">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">Application links</p>
                {childWatches && childWatches.map((childWatch) => {
                  return (
                    <div className="DetailPage--linksRow flex" key={childWatch.cluster.id}>
                      {childWatch.links.map((link) => {
                        return (
                          <div key={`${childWatch.cluster.id}-${link.title}`}>
                            {childWatches.length === 1 ?
                              <a href={link.uri} target="_blank" rel="noopener noreferrer"><button className="btn secondary">{link.title}</button></a>
                            :
                              <a href={link.uri} target="_blank" rel="noopener noreferrer"><button className="btn secondary">{childWatch.name} - {link.title}</button></a>
                            }
                          </div>
                        );
                      })}
                    </div>
                  );
                })}
              </div>
            }

            <div className="u-marginTop--30 u-paddingTop--10">
              <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">Edit application</p>
              <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--10">Update patches for your application. These patches will be applied to deployments on all clusters. To update patches for a cluster, find it below click “Customize” on the cluster you want to edit.</p>
              <div className="u-marginTop--10 u-paddingTop--5">
                <button disabled={preparingAppUpdate} onClick={() => isKotsApp ? this.navigateToFiles(watch) : this.handleEditWatchClick(watch)} className="btn secondary">{preparingAppUpdate ? "Preparing" : "Edit"} {isKotsApp ? watch.name : watch.watchName}</button>
              </div>
            </div>

            <div className="u-marginTop--30 u-paddingTop--10">
            {!childWatches?.length ?
              <div>
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">Downstreams</p>
                <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--10">You have not deployed your application to any downstream clusters. Get started by selecting a downstream cluster from the Downstreams tab.</p>
                <Link to={`/${isKotsApp ? "app" : "watch"}/${watch.slug}/downstreams`} className="btn secondary">Select a downstream cluster</Link>
              </div>
            :
              <div>
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">Downstreams</p>
                <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--10">Your app can be deployed to as many clusters as you would like. Each cluster can have it’s own configuration and patches for your kubernetes YAML.</p>
                <div className="flex flex-column u-marginTop--10 u-paddingTop--5">
                  {childWatches && childWatches.map((childWatch) => {
                    const childCluster = childWatch.cluster;
                    const clusterType = getClusterType(childCluster.gitOpsRef);
                    let versionNode = (
                      <div className="flex alignItems--center">
                        <div className="icon checkmark-icon"/>
                        <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-color--dustyGray">Up to date</span>
                      </div>
                    );
                    if (childWatch.pendingVersions?.length) {
                      versionNode = (
                        <div className="flex alignItems--center">
                          <div className="icon exclamationMark--icon"/>
                          <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-color--orange">
                            {childWatch.pendingVersions?.length === 1 ? "1" : "2+"} {childWatch.pendingVersions?.length >= 2 ? "versions" : "version"} behind
                          </span>
                        </div>
                      );
                    }
                    if (!childWatch.currentVersion) {
                      versionNode = (
                        <div className="flex alignItems--center">
                          <div className="icon blueCircleMinus--icon"/>
                          <span className="u-marginLeft--5 u-fontSize--small u-fontWeight--medium u-color--dustyGray">No deployments made</span>
                        </div>
                      );
                    }
                    if (childCluster) {
                      return (
                        <div key={childCluster.id} className="DetailPage--downstreamRow flex">
                          <div className="flex1 flex alignItems--center">
                            <span className={`flex-auto icon clusterType ${clusterType}`}></span>
                            <span className="u-fontSize--normal u-color--tundora u-fontWeight--bold u-marginLeft--5" title={childCluster.title}>{truncateMiddle(childCluster.title, 15, 10, "...")}</span>
                          </div>
                          <div className="flex1">
                            {versionNode}
                          </div>
                          <div className="flex-auto">
                            {this.state[`preparing${childWatch.id}`]
                              ? <Loader size="16"/>
                              : <span onClick={() => isKotsApp ? this.navigateToFiles(watch) : this.handleEditWatchClick(childWatch)} className="u-fontSize--small replicated-link">Customize</span>
                            }
                          </div>
                        </div>
                      );
                    }
                  })}
                </div>
                <div className="u-marginTop--10 u-paddingTop--5">
                  <Link to={`/${isKotsApp ? "app" : "watch"}/${watch.slug}/downstreams`} className="btn secondary">See all downstreams</Link>
                </div>
              </div>
            }
            </div>

            {(!isEmpty(appMeta) && appMeta.applicationType === "replicated.app") &&
              <div className="u-marginTop--30 u-paddingTop--10 u-marginBottom--10 flex">
                <div className="flex1 u-paddingRight--15">
                  <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">Get help with your application</p>
                  <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--10">Generate a support bundle for your application to send to the vendor.</p>
                  <div className="u-marginTop--10">
                    <Link to={`/${isKotsApp ? "app" : "watch"}/${watch.slug}/troubleshoot/generate`} className="btn secondary">Generate a support bundle</Link>
                  </div>
                </div>
                <div className="flex1 u-paddingLeft--15">
                  <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">Application config</p>
                  <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--10">Quickly see a ready-only preview of your application config for reference.</p>
                  <div className="u-marginTop--10">
                    <Link to={`/${isKotsApp ? "app" : "watch"}/${watch.slug}/config`} className="btn secondary">See application config</Link>
                  </div>
                </div>
              </div>
            }

            {!isSingleTenant() && (
              <div className="u-marginTop--30 u-borderTop--gray u-paddingTop--30 u-paddingBottom--20">
                <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--normal">Delete application</p>
                <p className="u-fontSize--small u-color--dustyGray u-lineHeight--normal u-marginBottom--10">Removing {this.state.appName} will permanently delete all data and integrations associated with it and will not be&nbsp;recoverable.</p>
                <div className="u-marginTop--10">
                  <button type="button" className="btn primary red" onClick={this.toggleConfirmDelete}>Delete application</button>
                </div>
              </div>
            )}
          </div>
          {!isSingleTenant() && (
            <div className="flex1 flex-column detail-right-sidebar u-paddingLeft--30">
              {watch?.currentVersion &&
                <div>
                  <p className="uppercase-title">Current Version</p>
                  <p className="u-fontSize--jumbo2 u-fontWeight--bold u-color--tuna">
                    {watch?.currentVersion?.title}
                  </p>
                </div>
              }
              <WatchContributors
                title="contributors"
                className="u-marginTop--30"
                refetchWatch={this.props.refetchWatch}
                contributors={watch.contributors || []}
                watchName={isKotsApp ? watch.name : watch.watchName}
                watchId={watch.id}
                watchCallback={updateCallback}
                slug={watch.slug}
              />
            </div>
          )}
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
  }),
  graphql(getWatchLicense, {
    name: "getWatchLicense",
    skip: props => {
      const { owner } = props.match.params;
      // Skip if it's a KOTS app
      if (!owner) {
        return true;
      }
      return false;
    },
    options: props => {
      return {
        variables: {
          watchId: props.watch.id
        },
        fetchPolicy: "no-cache"
      };
    }
  }),
)(DetailPageApplication);
