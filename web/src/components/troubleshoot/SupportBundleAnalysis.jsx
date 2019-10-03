import * as React from "react";
import { withRouter, Switch, Route, Link } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import dayjs from "dayjs";
import Modal from "react-modal";

import Loader from "../shared/Loader";
import AnalyzerInsights from "./AnalyzerInsights";
import AnalyzerFileTree from "./AnalyzerFileTree";
import { isKotsApplication } from "@src/utilities/utilities";
import { getSupportBundle } from "../../queries/TroubleshootQueries";
import { updateSupportBundle } from "../../mutations/TroubleshootMutations";
import "../../scss/components/troubleshoot/SupportBundleAnalysis.scss";

export class SupportBundleAnalysis extends React.Component {
  constructor(props) {
    super();
    this.state = {
      activeTab: props.location.pathname.indexOf("/contents") !== -1 ? "fileTree" : "bundleAnalysis",
      fullscreenTree: false,
      filterTiles: "0",
      displayShareBundleModal: false
    };
  }

  shareBundle = (unShare = false) => {
    const { getSupportBundle } = this.props;
    const bundle = getSupportBundle.getSupportBundle;
    this.setState({ shareBundleLoading: true });
    this.props.client.mutate({
      mutation: updateSupportBundle,
      variables: {
        id: bundle.id,
        shareTeamIDs: unShare ? [] : [
          "replicated",
        ],
      },
    })
      .then(() => {
        this.props.getSupportBundle.refetch();
        this.setState({ shareBundleLoading: false, displayShareBundleModal: false });
      }).catch(() => this.setState({ shareBundleLoading: false }));
  }

  toggleConfirmShareModal = () => {
    this.setState({ displayShareBundleModal: !this.state.displayShareBundleModal });
  }

  reAnalyzeBundle = async (callback) => {
    try {
      const bundleId = this.props.getSupportBundle.getSupportBundle.id;
      const bundleUrl = `${window.env.TROUBLESHOOT_ENDPOINT}/analyze/${bundleId}`;

      const response = await fetch(bundleUrl, {
        method: "POST"
      });
      const analyzedBundle = await response.json();
      if (callback && typeof callback === "function") {
        callback(analyzedBundle, analyzedBundle.status === "analysis_error");
      }
      this.props.getSupportBundle.refetch();
    } catch (error) {
      if (callback && typeof callback === "function") {
        callback(undefined, error);
      }
    }
  }

  downloadBundle = () => {
    const hiddenIFrameID = "hiddenDownloader";
    let iframe = document.getElementById(hiddenIFrameID);
    const url = this.state.bundle.signedUri;
    if (iframe === null) {
      iframe = document.createElement("iframe");
      iframe.id = hiddenIFrameID;
      iframe.style.display = "none";
      document.body.appendChild(iframe);
    }
    iframe.src = url;
  }

  toggleAnalysisAction = (active) => {
    this.setState({
      activeTab: active,
    });
  }

  toggleFullscreen = () => {
    this.setState({
      fullscreenTree: !this.state.fullscreenTree
    });
  }

  handleEscClose = (e) => {
    if (e.keyCode == 27 && this.state.fullscreenTree) {
      e.preventDefault();
      this.toggleFullscreen();
    }
  }

  renderSharedContext = () => {
    // const sharedIds = bundle.teamShareIds || [];
    // const isShared = sharedIds.length;
    let shareContext = null;

    // if (isShared) {
    //   shareContext = (
    //     <div className="flex flex-auto">
    //       <span className="u-marginRight--10 u-fontSize--normal u-fontWeight--medium u-color--tundora alignSelf--center">Shared with Replicated</span>
    //       <button className="btn secondary flex-auto u-marginRight--10" onClick={() => this.shareBundle(true)}>Unshare</button>
    //     </div>
    //   )
    // } else {
    //   shareContext = <button className="btn secondary flex-auto u-marginRight--10" onClick={this.toggleConfirmShareModal}>Share with Replicated</button>
    // }
    return shareContext;
  }

  componentDidUpdate(lastState) {
    if (this.state.fullscreenTree !== lastState.fullscreenTree && this.state.fullscreenTree) {
      window.addEventListener("keydown", this.handleEscClose);
    }
  }

  componentWillUnmount() {
    window.removeEventListener("keydown", this.handleEscClose);
  }

  render() {
    const { watch, getSupportBundle } = this.props;
    const bundle = getSupportBundle?.getSupportBundle;
    const isKotsApp = isKotsApplication(watch);

    if (getSupportBundle.loading) {
      return (
        <div className="flex-column flex1 justifyContent--center alignItems--center">
          <Loader size="60" color="#44bb66" />
        </div>
      )
    }

    let insightsUrl;
    if (isKotsApp) {
      insightsUrl = `/app/:slug/troubleshoot/analyze/:bundleSlug`;
    } else {
      insightsUrl = `/watch/:owner/:slug/troubleshoot/analyze/:bundleSlug`;
    }

    let fileTreeUrl;
    if (isKotsApp) {
      fileTreeUrl = `/app/:slug/troubleshoot/analyze/:bundleSlug/contents/*`;
    } else {
      fileTreeUrl = `/watch/:owner/:slug/troubleshoot/analyze/:bundleSlug/contents/*`;
    }

    return (
      <div className="container u-marginTop--20 u-paddingBottom--30 flex1 flex-column">
        <div className="flex1 flex-column">
          {bundle &&
            <div className="flex1 flex-column">
              <div className="u-position--relative flex-auto u-marginBottom--20 flex justifyContent--spaceBetween">
                <div className="flex flex1 u-marginTop--10 u-marginBottom--10">
                  <div className="flex1">
                    <div className="flex flex1 justifyContent--spaceBetween">
                      <div className="flex flex-column">
                        <h2 className="u-fontSize--header2 u-fontWeight--bold u-color--tuna flex alignContent--center alignItems--center">Support bundle analysis</h2>
                      </div>
                      <div className="flex flex-auto alignItems--center">
                        {this.renderSharedContext(bundle)}
                      </div>
                    </div>
                    <div className="upload-date-container flex u-marginTop--5 alignItems--center">
                      <div className="flex alignSelf--center">
                        <p className="flex u-fontSize--normal u-color--dustyGray u-fontWeight--medium">Uploaded on <span className="u-fontWeight--bold u-marginLeft--5">{dayjs(bundle.createdAt).format("MMMM D, YYYY @ h:mm a")}</span></p>
                      </div>
                    </div>
                  </div>
                </div>
              </div>
              <div className="flex-column flex1">
                <div className="customer-actions-wrapper flex1 flex-column">
                  <div className="flex action-tab-bar">
                    <Link to={`/${isKotsApp ? "app" : "watch"}/${watch.slug}/troubleshoot/analyze/${bundle.slug}`} className={`${this.state.activeTab === "bundleAnalysis" ? "is-active" : ""} tab-item blue`} onClick={() => this.toggleAnalysisAction("bundleAnalysis")}>Analysis overview</Link>
                    <Link to={`/${isKotsApp ? "app" : "watch"}/${watch.slug}/troubleshoot/analyze/${bundle.slug}/contents/`} className={`${this.state.activeTab === "fileTree" ? "is-active" : ""} tab-item blue`} onClick={() => this.toggleAnalysisAction("fileTree")}>File inspector</Link>
                  </div>
                  <div className="flex-column flex1 action-content">
                    <Switch>
                      <Route exact path={insightsUrl} render={() =>
                        <AnalyzerInsights
                          status={bundle.status}
                          refetchSupportBundle={this.props.getSupportBundle.refetch}
                          insights={bundle.analysis?.insights}
                          reAnalyzeBundle={this.reAnalyzeBundle}
                        />
                      } />
                      <Route exact path={fileTreeUrl} render={() =>
                        <AnalyzerFileTree
                          watchSlug={watch.slug}
                          bundle={bundle}
                          downloadBundle={this.downloadBundle}
                          isKotsApp={isKotsApp}
                        />
                      } />
                    </Switch>
                  </div>
                </div>
              </div>
            </div>
          }
        </div>
        {this.state.displayShareBundleModal &&
          <Modal
            isOpen={this.state.displayShareBundleModal}
            onRequestClose={() => this.toggleConfirmShareModal()}
            shouldReturnFocusAfterClose={false}
            contentLabel="Modal"
            ariaHideApp={false}
            className="console Modal DefaultSize"
          >
            <div className="Modal-header">
              <p>Share this Support Bundle</p>
            </div>
            <div className="flex flex-column u-paddingLeft--20 u-paddingRight--20 u-paddingBottom--20">
              <span className="u-fontSize--large u-fontWeight--normal u-color--dustyGray u-lineHeight--more">By sharing this bundle, Replicated will be able to view analyzers and inspect all files. Only this bundle will be visible, no other bundles will be seen by our team.</span>
              <div className="flex justifyContent--flexEnd u-marginTop--30">
                <button className="btn secondary flex-auto u-marginRight--10" onClick={() => { this.toggleConfirmShareModal() }}>Cancel</button>
                <button className="btn secondary green flex-auto" disabled={this.state.shareBundleLoading} onClick={() => this.shareBundle()}>{this.state.shareBundleLoading ? "Sharing" : "Share bundle"}</button>
              </div>
            </div>
          </Modal>
        }
      </div>
    );
  }
}

export default withRouter(compose(
  withApollo,
  graphql(getSupportBundle, {
    name: "getSupportBundle",
    options: ({ match }) => ({
      variables: {
        watchSlug: match.params.bundleSlug
      },
      fetchPolicy: "no-cache"
    }),
  })
)(SupportBundleAnalysis));
