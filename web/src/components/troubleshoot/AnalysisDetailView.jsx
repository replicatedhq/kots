import * as React from "react";
import { withRouter, Link } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import dayjs from "dayjs";
import Modal from "react-modal";

import Loader from "../shared/Loader";
import AnalyzerInsights from "./AnalyzerInsights";
import AnalyzerFileTree from "./AnalyzerFileTree";

import { getAnalysisInsights } from "../../queries/TroubleshootQueries";
import { updateSupportBundle, markSupportBundleUploaded } from "../../mutations/SupportBundleMutations";
import "../../scss/components/support/SupportBundleAnalysis.scss";

export class AnalysisDetailView extends React.Component {
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
    this.setState({ shareBundleLoading: true });
    this.props.client.mutate({
      mutation: updateSupportBundle,
      variables: {
        id: this.state.bundle.id,
        shareTeamIDs: unShare ? [] : [
          "replicated",
        ],
      },
    })
      .then(() => {
        this.props.data.refetch();
        this.setState({ shareBundleLoading: false, displayShareBundleModal: false });
      }).catch(() => this.setState({ shareBundleLoading: false }));
  }

  toggleConfirmShareModal = () => {
    this.setState({ displayShareBundleModal: !this.state.displayShareBundleModal });
  }

  reAnalyzeBundle = (callback) => {
    this.props.markSupportBundleUploaded(this.props.data.supportBundleForSlug.bundle.id)
      .then(async (response) => {
        await this.props.data.refetch();
        if (callback && typeof callback === "function") {
          callback(response);
        }
      })
      .catch((error) => {
        if (callback && typeof callback === "function") {
          callback(undefined, error);
        }
      });
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

  renderAnalysisTab = () => {
    switch (this.state.activeTab) {
    case "bundleAnalysis":
      return (
        <div className="flex flex-column flex1 insights-wrapper u-overflow--hidden">
          <AnalyzerInsights
            insights={this.props.data.supportBundleForSlug.insights}
            reAnalyzeBundle={this.reAnalyzeBundle}
          />
        </div>
      )
    case "fileTree":
      return (
        <div className={`flex-column flex1 insights-wrapper ${this.state.fullscreenTree ? "fullscreen-tree-view" : ""}`} >
          <AnalyzerFileTree
            toggleFullscreen={() => this.toggleFullscreen()}
            isFullscreen={this.state.fullscreenTree}
            appSlug={this.props.match.params.appSlug}
            reAnalyzeBundle={this.reAnalyzeBundle}
          />
        </div>
      )
    default:
      return <div>nothing selected</div>
    }
  }

  renderSharedContext = (bundle) => {
    const notSameTeam = bundle.teamId !== VendorUtilities.getTeamId();
    const isSameTeam = !notSameTeam;
    const sharedIds = bundle.teamShareIds || [];
    const isShared = sharedIds.length;
    let shareContext;

    if (notSameTeam) {
      shareContext = <span className="u-marginRight--normal u-fontSize--normal u-color--chateauGreen">Shared by <span className="u-fontWeight--bold">{bundle.teamName}</span></span>
    } else if (isSameTeam && isShared) {
      shareContext = (
        <div className="flex flex-auto">
          <span className="u-marginRight--normal u-fontSize--normal u-fontWeight--medium u-color--tundora alignSelf--center">Shared with Replicated</span>
          <button className="Button secondary button flex-auto u-marginRight--normal" onClick={() => this.shareBundle(true)}>Unshare</button>
        </div>
      )
    } else {
      shareContext = <button className="Button secondary button flex-auto u-marginRight--normal" onClick={this.toggleConfirmShareModal}>Share with Replicated</button>
    }
    return shareContext;
  }

  componentDidUpdate(lastProps, lastState) {
    if (this.props.data.supportBundleForSlug !== lastProps.data.supportBundleForSlug && this.props.data.supportBundleForSlug) {
      this.setState({
        bundle: this.props.data.supportBundleForSlug.bundle,
        insights: ConsoleUtilities.sortAnalyzers(this.props.data.supportBundleForSlug.insights)
      });
    }

    if (this.state.fullscreenTree !== lastState.fullscreenTree && this.state.fullscreenTree) {
      window.addEventListener("keydown", this.handleEscClose);
    }
  }

  componentDidMount() {
    if (this.props.data.supportBundleForSlug) {
      this.setState({
        bundle: this.props.data.supportBundleForSlug.bundle,
        insights: ConsoleUtilities.sortAnalyzers(this.props.data.supportBundleForSlug.insights)
      });
    }
  }

  componentWillUnmount() {
    window.removeEventListener("keydown", this.handleEscClose);
  }

  render() {
    const bundle = this.props.data && this.props.data.supportBundleForSlug && this.props.data.supportBundleForSlug.bundle;

    if (this.props.data.loading) {
      return (
        <div className="flex-column flex1 justifyContent--center alignItems--center">
          <Loader size="60" color="#337AB7" />
        </div>
      )
    }


    return (
      <div className="console container u-marginTop--more u-paddingBottom--normal flex1 flex">
        <div className="flex1 flex-column">
          <div className="flex flex1">
            <div className="flex1 flex-column">
              <div className="CreateAction u-marginBottom--small">
                <Link to={`/troubleshoot`} className="u-paddingLeft--normal u-marginTop--small
                          u-fontSize--normal u-color--astral
                          u-fontWeight--medium
                          u-cursor--pointer">
                  <span className="icon clickable u-backArrowIcon"></span>Support bundles
                </Link>
              </div>
              {bundle ?
                <div className="flex1 flex-column">
                  <div className="u-position--relative flex-auto u-marginBottom--more flex justifyContent--spaceBetween">
                    <div className="flex flex1 u-marginTop--normal u-marginBottom--normal">
                      <div className="flex1">
                        <div className="flex flex1 justifyContent--spaceBetween">
                          <div className="flex flex-column">
                            <h2 className="u-fontSize--header2 u-fontWeight--bold u-color--tuna flex alignContent--center alignItems--center">Support bundle analysis</h2>
                          </div>
                          <div className="flex flex-auto alignItems--center">
                            {this.renderSharedContext(bundle)}
                          </div>
                        </div>
                        <div className="upload-date-container flex u-marginTop--small alignItems--center">
                          <div className="flex alignSelf--center">
                            <p className="flex u-fontSize--normal u-color--dustyGray u-fontWeight--medium">Uploaded on </p>
                          </div>
                          <div className="flex flex u-fontSize--normal u-color--dustyGray u-fontWeight--medium">
                            <span className="u-fontWeight--bold u-marginLeft--small flex-auto"> {dayjs(bundle.createdAt).format("MMMM Do YYYY, h:mm a")}</span>
                          </div>
                        </div>
                      </div>
                    </div>
                  </div>
                  <div className="flex-column flex1">
                    <div className="customer-actions-wrapper flex1 flex-column">
                      <div className="flex action-tab-bar">
                        <Link to={`/troubleshoot/analyze/${bundle.slug}`} className={`${this.state.activeTab === "bundleAnalysis" ? "is-active" : ""} tab-item`} onClick={() => this.toggleAnalysisAction("bundleAnalysis")}>Analysis overview</Link>
                        <Link to={`/troubleshoot/analyze/${bundle.slug}/contents`} className={`${this.state.activeTab === "fileTree" ? "is-active" : ""} tab-item`} onClick={() => this.toggleAnalysisAction("fileTree")}>File inspector</Link>
                      </div>
                      <div className="flex flex1 action-content u-marginBottom--40">
                        {this.renderAnalysisTab()}
                      </div>
                    </div>
                  </div>
                </div>
                : null}
            </div>
          </div>
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
            <div className="flex flex-column u-paddingLeft--more u-paddingRight--more u-paddingBottom--more">
              <span className="u-fontSize--large u-fontWeight--normal u-color--dustyGray u-lineHeight--more">By sharing this bundle, Replicated will be able to view analyzers and inspect all files. Only this bundle will be visible, no other bundles will be seen by our team.</span>
              <div className="flex justifyContent--flexEnd u-marginTop--30">
                <button className="btn secondary blue flex-auto u-marginRight--normal" onClick={() => { this.toggleConfirmShareModal() }}>Cancel</button>
                <button className="btn primary flex-auto" disabled={this.state.shareBundleLoading} onClick={() => this.shareBundle()}>{this.state.shareBundleLoading ? "Sharing" : "Share bundle"}</button>
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
  graphql(getAnalysisInsights, {
    options: ({ match }) => ({
      variables: {
        slug: match.params.bundleSlug
      },
      fetchPolicy: "no-cache"
    }),
  }),
  graphql(markSupportBundleUploaded, {
    props: ({ mutate }) => ({
      markSupportBundleUploaded: (id) => mutate({ variables: { id } })
    })
  })
)(AnalysisDetailView));
