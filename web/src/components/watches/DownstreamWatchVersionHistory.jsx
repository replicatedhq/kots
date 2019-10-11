import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import { compose, withApollo, graphql } from "react-apollo";
import classNames from "classnames";
import MonacoEditor from "react-monaco-editor";
import Loader from "../shared/Loader";
import DownstreamVersionRow from "./DownstreamVersionRow";
import filter from "lodash/filter";
import Modal from "react-modal";
import map from "lodash/map";

import MarkdownRenderer from "@src/components/shared/MarkdownRenderer";
import { getDownstreamHistory } from "../../queries/WatchQueries";
import { getKotsDownstreamHistory, getKotsDownstreamOutput } from "../../queries/AppsQueries";

import "@src/scss/components/watches/WatchVersionHistory.scss";
import { isKotsApplication, hasPendingPreflight, getPreflightResultState } from "../../utilities/utilities";

class DownstreamWatchVersionHistory extends Component {
  state = {
    showSkipModal: false,
    showDeployWarningModal: false,
    deployParams: {},
    deployingSequence: null,
    releaseNotes: null,
    logs: null,
    selectedTab: null,
    logsLoading: false
  }

  handleMakeCurrent = async (upstreamSlug, sequence, clusterSlug, status) => {
    if (this.props.makeCurrentVersion && typeof this.props.makeCurrentVersion === "function") {
      await this.setDeploySequence(sequence);
      if (status === "pending_preflight") {
        this.setState({
          showSkipModal: true,
          deployParams: {
            upstreamSlug,
            sequence,
            clusterSlug
          }
        });
        return;
      }
      const version = this.props.data?.getKotsDownstreamHistory?.find( v => v.sequence === sequence);
      // If status is undefined - this is a force deploy.
      if (version?.preflightResult && status === "pending") {
        const preflightResults = JSON.parse(version.preflightResult);
        const preflightState = getPreflightResultState(preflightResults);

        if (preflightState === "fail") {
          this.setState({
            showDeployWarningModal: true,
            deployParams: {
              upstreamSlug,
              sequence,
              clusterSlug
            }
          });
          return;
        }
      }

      await this.props.makeCurrentVersion(upstreamSlug, sequence, clusterSlug);
      await this.props.data.refetch();
      this.setState({
        showSkipModal: false,
        showDeployWarningModal: false,
        deployParams: {},
        deployingSequence: null
      });
    }
  }
  setDeploySequence = deployingSequence => {
    return new Promise( resolve => {
      this.setState({
        deployingSequence
      }, resolve);
    })
  }

  hideSkipModal = () => {
    this.setState({
      showSkipModal: false,
      deployingSequence: null
    });
  }

  hideDeployWarningModal = () => {
    this.setState({
      showDeployWarningModal: false,
      deployingSequence: null
    });
  }

  showReleaseNotes = notes => {
    this.setState({
      releaseNotes: notes
    });
  }

  hideReleaseNotes = () => {
    this.setState({
      releaseNotes: null
    });
  }

  showLogsModal = () => {
    this.setState({
      showLogsModal: true
    });
  }

  hideLogsModal = () => {
    this.setState({
      showLogsModal: false
    });
  }

  onForceDeployClick = () => {
    // Parameters are stored in state until deployed, then cleared after deploy
    const { upstreamSlug, sequence, clusterSlug } = this.state.deployParams;

    this.handleMakeCurrent(upstreamSlug, sequence, clusterSlug);
  }

  getActiveDownstreamVersion = versionHistory => {
    if (!versionHistory.length) {
      return null;
    }
    const deployed = filter(versionHistory, version => version.status === "deployed");
    deployed.sort((v1, v2) => v1.sequence > v2.sequence);
    return deployed.length ? deployed[0] : null;
  }

  handleViewLogs = async version => {
    this.showLogsModal();
    this.setState({ logsLoading: true });
    const { match } = this.props;
    this.props.client.query({
      query: getKotsDownstreamOutput,
      fetchPolicy: "no-cache",
      variables: {
        appSlug: match.params.slug,
        clusterSlug: match.params.downstreamSlug,
        sequence: version.sequence
      }
    }).then(result => {
      const logs = result.data.getKotsDownstreamOutput;
      const selectedTab = Object.keys(logs)[0];
      this.setState({ logs, selectedTab, logsLoading: false });
    }).catch(err => {
      console.log(err);
      this.setState({ logsLoading: false });
    });
  }

  renderLogsTabs = () => {
    const { logs, selectedTab } = this.state;
    if (!logs) {
      return null;
    }
    const tabs = Object.keys(logs);
    return (
      <div className="flex action-tab-bar u-marginTop--10">
        {map(tabs, tab =>  (
          <div className={`tab-item blue ${tab === selectedTab && "is-active"}`} key={tab} onClick={() => this.setState({ selectedTab: tab })}>
            {tab}
          </div>
        ))}
      </div>
    );
  }

  render() {
    const { watch, match, data } = this.props;
    const { showSkipModal, showDeployWarningModal, releaseNotes, showLogsModal, logsLoading, logs, selectedTab } = this.state;
    const { watches, downstreams } = watch;
    const isKots = isKotsApplication(watch);
    const _slug = isKots ? match.params.downstreamSlug : `${match.params.downstreamOwner}/${match.params.downstreamSlug}`;
    const downstreamWatch = isKots ? downstreams.find(w => w.cluster.slug === _slug) : watches.find(w => w.slug === _slug );
    let versionHistory = [];
    if (isKots && data?.getKotsDownstreamHistory?.length) {
      versionHistory = data.getKotsDownstreamHistory;
    } else if (data?.getDownstreamHistory?.length) {
      versionHistory = data.getDownstreamHistory;
    }
    const activeDownstreamVersion = this.getActiveDownstreamVersion(versionHistory);
    const downstreamSlug = downstreamWatch ? downstreamWatch.cluster?.slug : "";
    const isGit = downstreamWatch?.cluster?.gitOpsRef;

    const centeredLoader = (
      <div className="flex-column flex1 alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );

    if (isKots && hasPendingPreflight(versionHistory)) {
      data?.startPolling(2000);
    } else {
      if (this.props.refreshAppData) {
        this.props.refreshAppData();
      }
      data?.stopPolling();
    }

    if (data.loading) {
      return centeredLoader;
    }

    return (
      <div className="flex-column flex1 u-position--relative u-padding--20 u-overflow--auto">
        <p className="flex-auto u-fontSize--larger u-fontWeight--bold u-color--tuna u-paddingBottom--20">Downstream version history: {downstreamSlug}</p>

        <div className="flex-column flex-auto ActiveRelease-wrapper">
          <div className="flex alignItems--center u-borderBottom--gray u-paddingBottom--5">
            <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna">Active release</p>
          </div>
          <div>
            {activeDownstreamVersion ?
              <DownstreamVersionRow
                key="current-downstream-version"
                downstreamWatch={downstreamWatch}
                version={activeDownstreamVersion}
                isKots={isKots}
                urlParams={match.params}
                onReleaseNotesClick={this.showReleaseNotes}
                handleMakeCurrent={this.handleMakeCurrent}
                handleViewLogs={this.handleViewLogs}
              />
            :
              <div className="no-current-version u-textAlign--center">
                <p className="u-fontSize--large u-color--tundora u-fontWeight--bold u-lineHeight--normal">No active release found on {downstreamSlug}</p>
                <p className="u-fontSize--normal u-color--dustygray u-fontWeight--medium u-lineHeight--normal">{isGit ? "When a PR is merged" : "When a version has been deployed"}, the current version will be shown here</p>
              </div>
            }
          </div>
        </div>

        <div className="flex1 flex-column u-paddingTop--20 u-marginTop--20">
          <div className="flex alignItems--center u-borderBottom--gray u-paddingBottom--5">
            <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna">All releases</p>
          </div>
          <div className={classNames("flex-column", { "flex1": data.loading })}>
            {versionHistory?.length > 0 && versionHistory.map( version => (
              <DownstreamVersionRow
                hasPreflight={watch.hasPreflight}
                key={`${version.title}-${version.sequence}`}
                downstreamWatch={downstreamWatch}
                isDeploying={version.sequence === this.state.deployingSequence}
                version={version}
                isKots={isKots}
                urlParams={match.params}
                onReleaseNotesClick={this.showReleaseNotes}
                handleMakeCurrent={this.handleMakeCurrent}
                handleViewLogs={this.handleViewLogs}
              />
            ))}
          </div>
        </div>

        <Modal
          isOpen={showSkipModal}
          onRequestClose={this.hideSkipModal}
          shouldReturnFocusAfterClose={false}
          contentLabel="Skip preflight checks"
          ariaHideApp={false}
          className="Modal SkipModal"
        >
          <div className="Modal-body">
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">
              Preflight checks have not finished yet. Are you sure you want to deploy this version?
            </p>
            <div className="u-marginTop--10 flex">
              <button
                onClick={this.onForceDeployClick}
                type="button"
                className="btn green primary">
                Deploy this version
              </button>
              <button type="button" onClick={this.hideSkipModal} className="btn secondary u-marginLeft--20">Cancel</button>
            </div>
          </div>
        </Modal>

        <Modal
          isOpen={showDeployWarningModal}
          onRequestClose={this.hideDeployWarningModal}
          shouldReturnFocusAfterClose={false}
          contentLabel="Skip preflight checks"
          ariaHideApp={false}
          className="Modal"
        >
          <div className="Modal-body">
            <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--normal u-marginBottom--20">
              Preflight checks for this version are currently failing. Are you sure you want to make this the current version?
            </p>
            <div className="u-marginTop--10 flex">
              <button
                onClick={this.onForceDeployClick}
                type="button"
                className="btn green primary"
              >
                Deploy this version
              </button>
              <button
                onClick={this.hideDeployWarningModal}
                type="button"
                className="btn secondary u-marginLeft--20"
              >
                Cancel
              </button>
            </div>
          </div>
        </Modal>

        <Modal
          isOpen={!!releaseNotes}
          onRequestClose={this.hideReleaseNotes}
          contentLabel="Release Notes"
          ariaHideApp={false}
          className="Modal DefaultSize"
        >
          <div className="flex-column">
            <MarkdownRenderer>
              {this.state.releaseNotes || ""}
            </MarkdownRenderer>
          </div>
          <div className="flex u-marginTop--10 u-marginLeft--10 u-marginBottom--10">
            <button className="btn primary" onClick={this.hideReleaseNotes}>Close</button>
          </div>
        </Modal>

        <Modal
          isOpen={showLogsModal}
          onRequestClose={this.hideLogsModal}
          shouldReturnFocusAfterClose={false}
          contentLabel="View logs"
          ariaHideApp={false}
          className="Modal logs-modal"
        >
          <div className="Modal-body flex flex1">
            {!logs || !selectedTab || logsLoading ? (
              <div className="flex-column flex1 alignItems--center justifyContent--center">
                <Loader size="60" />
              </div>
            ) : (
              <div className="flex-column flex1">
                {this.renderLogsTabs()}
                <div className="flex-column flex1 u-border--gray monaco-editor-wrapper">
                  <MonacoEditor
                    language="json"
                    value={logs[selectedTab]}
                    height="100%"
                    width="100%"
                    options={{
                      readOnly: true,
                      contextmenu: false,
                      minimap: {
                        enabled: false
                      },
                      scrollBeyondLastLine: false,
                    }}
                  />
                </div>
                <div className="u-marginTop--20 flex" onClick={this.hideLogsModal}>
                  <button type="button" className="btn primary" onClick={this.hideWarningModal}>Ok, got it!</button>
                </div>
              </div>
            )}
          </div>
        </Modal>
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  graphql(getKotsDownstreamHistory, {
    skip: props => {
      return props.match.params.downstreamOwner;
    },
    options: ({ match }) => ({
      variables: {
        upstreamSlug: match.params.slug,
        clusterSlug: match.params.downstreamSlug,
      },
      fetchPolicy: "no-cache"
    })
  }),
  graphql(getDownstreamHistory, {
    skip: props => {
      return !props.match.params.downstreamOwner;
    },
    options: ({ match }) => ({
      variables: {
        slug: `${match.params.downstreamOwner}/${match.params.downstreamSlug}`
      },
      fetchPolicy: "no-cache"
    })
  }),
)(DownstreamWatchVersionHistory);
