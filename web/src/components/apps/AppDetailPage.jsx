import React, { Component, Fragment } from "react";
import classNames from "classnames";
import { withRouter, Switch, Route } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import Modal from "react-modal";

import withTheme from "@src/components/context/withTheme";
import { getKotsApp, listDownstreamsForApp } from "@src/queries/AppsQueries";
import { checkForKotsUpdates } from "../../mutations/AppsMutations";
import { createKotsDownstream, deleteKotsDownstream, deployKotsVersion } from "../../mutations/AppsMutations";
import WatchSidebarItem from "@src/components/watches/WatchSidebarItem";
import { KotsSidebarItem } from "@src/components/watches/WatchSidebarItem";
import { HelmChartSidebarItem } from "@src/components/watches/WatchSidebarItem";
import NotFound from "../static/NotFound";
import DetailPageApplication from "../watches/DetailPageApplication";
import DetailPageIntegrations from "../watches/DetailPageIntegrations";
import AddClusterModal from "../shared/modals/AddClusterModal";
import CodeSnippet from "../shared/CodeSnippet";
import DeploymentClusters from "../watches/DeploymentClusters";
import DownstreamTree from "../../components/tree/KotsApplicationTree";
import AppVersionHistory from "./AppVersionHistory";
import DownstreamWatchVersionHistory from "../watches/DownstreamWatchVersionHistory";
import DownstreamWatchVersionDiff from "../watches/DownstreamWatchVersionDiff";
import PreflightResultPage from "../PreflightResultPage";
import AppConfig from "../apps/AppConfig";
import WatchLicense from "../watches/WatchLicense";
import SubNavBar from "@src/components/shared/SubNavBar";
import SidebarLayout from "../layout/SidebarLayout/SidebarLayout";
import SideBar from "../shared/SideBar";
import Loader from "../shared/Loader";
import SupportBundleList from "../troubleshoot/SupportBundleList";
import SupportBundleAnalysis from "../troubleshoot/SupportBundleAnalysis";
import GenerateSupportBundle from "../troubleshoot/GenerateSupportBundle";
import AppSettings from "./AppSettings";

import "../../scss/components/watches/WatchDetailPage.scss";

let loadingTextTimer = null;
class AppDetailPage extends Component {
  constructor(props) {
    super(props);
    this.state = {
      preparingUpdate: "",
      clusterParentSlug: "",
      selectedWatchName: "",
      clusterToRemove: {},
      watchToEdit: {},
      existingDeploymentClusters: [],
      checkingForUpdates: false,
      checkingUpdateText: "Checking for updates",
      updateError: false,
      displayDownloadCommandModal: false
    }
  }

  static defaultProps = {
    getKotsAppQuery: {
      loading: true
    }
  }

  componentDidUpdate(lastProps) {
    const { getThemeState, setThemeState, match, listApps, history, getKotsAppQuery } = this.props;
    const { search } = this.props.location;
    const slug = `${match.params.owner}/${match.params.slug}`;
    const currentWatch = listApps?.find(w => w.slug === slug);

    // Handle updating the app theme state when a watch changes.
    if (currentWatch?.watchIcon) {
      const { navbarLogo, ...rest } = getThemeState();
      if (navbarLogo === null || navbarLogo !== currentWatch.watchIcon) {

        setThemeState({
          ...rest,
          navbarLogo: currentWatch.watchIcon
        });
      }
    }

    if (getKotsAppQuery.getKotsApp && getKotsAppQuery.getKotsApp !== lastProps.getKotsAppQuery.getKotsApp) {
      const URLParams = new URLSearchParams(search);
      if (URLParams.get("add")) {
        this.handleAddNewClusterClick(getKotsAppQuery.getKotsApp);
        history.replace(this.props.location.pathname); // remove query param so refreshing the page doesn't trigger the modal again.
      }
    }

    // Used for a fresh reload
    if (history.location.pathname === "/apps") {
      this.checkForFirstApp();
    }

  }

  componentWillUnmount() {
    clearInterval(this.interval);
    this.props.clearThemeState();
  }

  makeCurrentRelease = async (upstreamSlug, sequence, clusterSlug) => {
    await this.props.deployKotsVersion(upstreamSlug, sequence, clusterSlug).then(() => {
      this.refetchGraphQLData();
    })
  }

  onCheckForUpdates = async () => {
    const { client, getKotsAppQuery } = this.props;
    const { getKotsApp: app } = getKotsAppQuery;

    this.setState({ checkingForUpdates: true });

    loadingTextTimer = setTimeout(() => {
      this.setState({ checkingUpdateText: "Almost there, hold tight..." });
    }, 10000);

    await client.mutate({
      mutation: checkForKotsUpdates,
      variables: {
        appId: app.id,
      }
    }).catch(() => {
      this.setState({ updateError: true });
    }).finally(() => {
      this.refetchGraphQLData();
      clearTimeout(loadingTextTimer);
      this.setState({
        checkingForUpdates: false,
        checkingUpdateText: "Checking for updates"
      });
    });
  }

  addClusterToApp = async (clusterId) => {
    const app = this.props.getKotsAppQuery.getKotsApp;
    try {
      await this.props.createKotsDownstream(app.id, clusterId);
      await this.props.listDownstreamsForAppQuery.refetch();
      this.closeAddClusterModal();
    } catch (error) {
      console.log(error);
    }
  }

  toggleDisplayDownloadModal = () => {
    this.setState({ displayDownloadCommandModal: !this.state.displayDownloadCommandModal });
  }

  createDownstreamForCluster = () => {
    const { clusterParentSlug } = this.state;
    localStorage.setItem("clusterRedirect", `/watch/${clusterParentSlug}/downstreams?add=1`);
    this.props.history.push("/cluster/create");
  }

  handleViewFiles = () => {
    const { slug } = this.props.match.params;
    const currentSequence = this.props.getKotsAppQuery?.getKotsApp?.currentSequence;
    this.props.history.push(`/app/${slug}/tree/${currentSequence}`);
  }

  handleAddNewClusterClick = (app) => {
    const downstreams = this.props.listDownstreamsForAppQuery.listDownstreamsForApp;
    const existingIds = downstreams ? downstreams.map(d => d.id) : [];
    this.setState({
      addNewClusterModal: true,
      clusterParentSlug: app.slug,
      selectedAppName: app.name,
      existingDeploymentClusters: existingIds
    });
  }

  toggleDeleteDeploymentModal = (cluster) => {
    const name = this.props.getKotsAppQuery?.getKotsApp?.name;
    this.setState({
      clusterToRemove: cluster,
      selectedWatchName: name,
      displayRemoveClusterModal: !this.state.displayRemoveClusterModal
    });
  }

  onDeleteDeployment = async () => {
    const { clusterToRemove } = this.state;
    const { slug } = this.props.match.params;
    try {
      await this.props.deleteKotsDownstream(slug, clusterToRemove.id);
      await this.props.listDownstreamsForAppQuery.refetch();
      this.setState({
        clusterToRemove: {},
        selectedWatchName: "",
        displayRemoveClusterModal: false
      });
    } catch (error) {
      console.log(error);
    }
  }

  closeAddClusterModal = () => {
    this.setState({
      addNewClusterModal: false,
      clusterParentSlug: "",
      selectedAppName: "",
      existingDeploymentClusters: []
    })
  }

  handleUploadNewVersionClick = () => {
    this.props.history.push(`/${this.props.match.params.slug}/airgap`);
  }

  /**
   * Refetch all the GraphQL data for this component and all its children
   *
   * @return {undefined}
   */
  refetchGraphQLData = () => {
    this.props.getKotsAppQuery.refetch();
    this.props.refetchListApps();
  }

  /**
   *  Runs on mount and on update. Also handles redirect logic
   *  if no apps are found, or the first app is found.
   */
  checkForFirstApp = () => {
    const { history, rootDidInitialAppFetch, listApps } = this.props;
    if (!rootDidInitialAppFetch) {
      return;
    }
    const firstApp = listApps?.find( app => app.name);

    if (firstApp) {
      history.replace(`/app/${firstApp.slug}`);
    } else {
      history.replace(window.env.NO_APPS_REDIRECT);
    }
  }

  componentDidMount() {
    const { history } = this.props;

    if (history.location.pathname === "/apps") {
      return this.checkForFirstApp();
    }

  }

  render() {
    const {
      match,
      getKotsAppQuery,
      listApps,
      refetchListApps,
      rootDidInitialAppFetch
    } = this.props;
    const {
      displayRemoveClusterModal,
      addNewClusterModal,
      clusterToRemove,
      checkingUpdateText,
      updateError,
      displayDownloadCommandModal
    } = this.state;

    const centeredLoader = (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );

    const app = getKotsAppQuery?.getKotsApp;
    const refreshAppData = getKotsAppQuery.refetch;
    const loading = getKotsAppQuery?.loading || !rootDidInitialAppFetch;

    if (!rootDidInitialAppFetch) {
      return centeredLoader;
    }


    return (
      <div className="WatchDetailPage--wrapper flex-column flex1 u-overflow--auto">
        <SidebarLayout
          className="flex flex1 u-minHeight--full u-overflow--hidden"
          condition={listApps?.length > 1}
          sidebar={(
            <SideBar
              items={listApps?.map((item, idx) => {
                let sidebarItemNode;
                if (item.name) {
                  const slugFromRoute = match.params.slug;
                  sidebarItemNode = (
                    <KotsSidebarItem
                      key={idx}
                      className={classNames({
                        selected: (
                          item.slug === slugFromRoute &&
                          match.params.owner !== "helm"
                        )
                      })}
                      app={item} />
                  );
                } else if (item.slug) {
                  const slugFromRoute = `${match.params.owner}/${match.params.slug}`;
                  sidebarItemNode = (
                    <WatchSidebarItem
                      key={idx}
                      className={classNames({
                        selected: (
                          item.slug === slugFromRoute &&
                          match.params.owner !== "helm"
                        )
                      })}
                      watch={item} />
                  );
                } else if (item.helmName) {
                  sidebarItemNode = (
                    <HelmChartSidebarItem
                      key={idx}
                      className={classNames({ selected: item.id === match.params.slug })}
                      helmChart={item} />
                  );
                }
                return sidebarItemNode;
              })}
            />
          )}>
          <div className="flex-column flex1 u-width--full u-height--full">
            {loading
              ? centeredLoader
              : (
                <Fragment>
                  <SubNavBar
                    className="flex"
                    activeTab={match.params.tab || "app"}
                    watch={app}
                    isKurlEnabled={this.props.isKurlEnabled}
                  />
                  <Switch>
                    <Route exact path="/app/:slug" render={() =>
                      <DetailPageApplication
                        watch={app}
                        refetchListApps={refetchListApps}
                        refetchWatch={this.props.getKotsAppQuery?.refetch}
                        updateCallback={this.refetchGraphQLData}
                        onActiveInitSession={this.props.onActiveInitSession}
                      />}
                    />
                    <Route exact path="/app/:slug/downstreams" render={() =>
                      <div className="container">
                        <DeploymentClusters
                          appDetailPage={true}
                          parentWatch={app}
                          title={app.name}
                          kotsApp={true}
                          parentClusterName={app.name}
                          displayDownloadCommand={this.toggleDisplayDownloadModal}
                          preparingUpdate={this.state.preparingUpdate}
                          childWatches={app.downstreams}
                          handleAddNewCluster={() => this.handleAddNewClusterClick(app)}
                          handleViewFiles={this.handleViewFiles}
                          installLatestVersion={this.makeCurrentRelease}
                          toggleDeleteDeploymentModal={this.toggleDeleteDeploymentModal}
                        />
                      </div>
                    } />
                    <Route exact path="/app/:slug/integrations" render={() => <DetailPageIntegrations watch={app} />} />

                    <Route exact path="/app/:slug/tree/:sequence" render={props => <DownstreamTree {...props} appNameSpace={this.props.appNameSpace} />} />

                    <Route exact path="/app/:slug/version-history" render={() =>
                      <AppVersionHistory
                        app={app}
                        match={this.props.match}
                        onUploadNewVersion={this.handleUploadNewVersionClick}
                        onCheckForUpdates={this.onCheckForUpdates}
                        checkingForUpdates={this.state.checkingForUpdates}
                        checkingUpdateText={checkingUpdateText}
                        handleAddNewCluster={() => this.handleAddNewClusterClick(app)}
                        errorCheckingUpdate={updateError}
                        makeCurrentVersion={this.makeCurrentRelease}
                      />
                    } />
                    <Route exact path="/app/:slug/downstreams/:downstreamSlug/version-history" render={() =>
                      <DownstreamWatchVersionHistory
                        watch={app}
                        makeCurrentVersion={this.makeCurrentRelease}
                        refreshAppData={refreshAppData}
                      />
                    } />
                    <Route exact path="/app/:slug/version-history/diff/:firstSequence/:secondSequence" render={() => 
                      <DownstreamWatchVersionDiff 
                        watch={app}
                      /> 
                      }
                    />
                    <Route exact path="/app/:slug/downstreams/:downstreamSlug/version-history/preflight/:sequence" render={() => <PreflightResultPage />} />
                    <Route exact path="/app/:slug/config" render={() =>
                      <AppConfig
                        app={app}
                        refreshAppData={refreshAppData}
                      />
                    } />
                    <Route exact path="/app/:slug/troubleshoot" render={() =>
                      <SupportBundleList
                        watch={app}
                      />
                    } />
                    <Route exact path="/app/:slug/troubleshoot/generate" render={() =>
                      <GenerateSupportBundle
                        watch={app}
                      />
                    } />
                    <Route path="/app/:slug/troubleshoot/analyze/:bundleSlug" render={() =>
                      <SupportBundleAnalysis
                        watch={app}
                      />
                    } />
                    <Route exact path="/app/:slug/license" render={() =>
                      <WatchLicense
                        watch={app}
                      />
                    } />
                    <Route exact path="/app/:slug/airgap-settings" render={() =>
                      <AppSettings
                        app={app}
                      />
                    } />
                    <Route component={NotFound} />
                  </Switch>
                </Fragment>
              )
            }
          </div>
        </SidebarLayout>
        {addNewClusterModal &&
          <Modal
            isOpen={addNewClusterModal}
            onRequestClose={this.closeAddClusterModal}
            shouldReturnFocusAfterClose={false}
            contentLabel="Add cluster modal"
            ariaHideApp={false}
            className="AddNewClusterModal--wrapper Modal"
          >
            <div className="Modal-body">
              <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Add {this.state.selectedWatchName} to a new downstream</h2>
              <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Select one of your existing downstreams to deploy to.</p>
              <AddClusterModal
                onAddCluster={this.addClusterToApp}
                watch={app}
                onRequestClose={this.closeAddClusterModal}
                createDownstreamForCluster={this.createDownstreamForCluster}
                existingDeploymentClusters={this.state.existingDeploymentClusters}
              />
            </div>
          </Modal>
        }
        {displayRemoveClusterModal &&
          <Modal
            isOpen={displayRemoveClusterModal}
            onRequestClose={() => this.toggleDeleteDeploymentModal({}, "")}
            shouldReturnFocusAfterClose={false}
            contentLabel="Add cluster modal"
            ariaHideApp={false}
            className="RemoveClusterFromWatchModal--wrapper Modal"
          >
            <div className="Modal-body">
              <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Remove {this.state.selectedWatchName} from {clusterToRemove.title}</h2>
              <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">This application will no longer be deployed to {clusterToRemove.title}.</p>
              <div className="u-marginTop--10 flex">
                <button onClick={() => this.toggleDeleteDeploymentModal({}, "")} className="btn secondary u-marginRight--10">Cancel</button>
                <button onClick={this.onDeleteDeployment} className="btn green primary">Delete deployment</button>
              </div>
            </div>
          </Modal>
        }
        {displayDownloadCommandModal &&
          <Modal
            isOpen={displayDownloadCommandModal}
            onRequestClose={this.toggleDisplayDownloadModal}
            shouldReturnFocusAfterClose={false}
            contentLabel="Download cluster command modal"
            ariaHideApp={false}
            className="DisplayDownloadCommandModal--wrapper Modal"
          >
            <div className="Modal-body">
              <h2 className="u-fontSize--largest u-color--tuna u-fontWeight--bold u-lineHeight--normal">Download assets</h2>
              <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">Run this command in your cluster to download the assets.</p>
              <CodeSnippet
                language="bash"
                canCopy={true}
                onCopyText={<span className="u-color--chateauGreen">Command has been copied to your clipboard</span>}
              >
                kubectl krew install kots
                {`kubectl kots download <namespace>`}
              </CodeSnippet>
              <div className="u-marginTop--10 flex">
                <button onClick={this.toggleDisplayDownloadModal} className="btn green primary">Ok, got it!</button>
              </div>
            </div>
          </Modal>
        }
      </div>
    );
  }
}

export { AppDetailPage };
export default compose(
  withApollo,
  withRouter,
  withTheme,
  graphql(getKotsApp, {
    name: "getKotsAppQuery",
    skip: props => {
      const { slug } = props.match.params;

      // Skip if no variables (user at "/watches" URL)
      if (!slug) {
        return true;
      }

      return false;

    },
    options: props => {
      const { slug } = props.match.params;
      return {
        fetchPolicy: "no-cache",
        variables: {
          slug: slug
        }
      }
    }
  }),
  graphql(listDownstreamsForApp, {
    name: "listDownstreamsForAppQuery",
    skip: props => {
      const { slug } = props.match.params;

      // Skip if no variables (user at "/watches" URL)
      if (!slug) {
        return true;
      }

      return false;

    },
    options: props => {
      const { slug } = props.match.params;
      return {
        fetchPolicy: "no-cache",
        variables: {
          slug: slug
        }
      }
    }
  }),
  graphql(createKotsDownstream, {
    props: ({ mutate }) => ({
      createKotsDownstream: (appId, clusterId) => mutate({ variables: { appId, clusterId } })
    })
  }),
  graphql(deleteKotsDownstream, {
    props: ({ mutate }) => ({
      deleteKotsDownstream: (slug, clusterId) => mutate({ variables: { slug, clusterId } })
    })
  }),
  graphql(deployKotsVersion, {
    props: ({ mutate }) => ({
      deployKotsVersion: (upstreamSlug, sequence, clusterSlug) => mutate({ variables: { upstreamSlug, sequence, clusterSlug } })
    })
  }),
)(AppDetailPage);
