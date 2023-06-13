import * as React from "react";
import { Switch, Route, Link, Outlet } from "react-router-dom";
import { withRouter } from "@src/utilities/react-router-utilities";
import dayjs from "dayjs";
import Modal from "react-modal";

import Loader from "../shared/Loader";
import AnalyzerInsights from "./AnalyzerInsights";
import AnalyzerFileTree from "./AnalyzerFileTree";
import AnalyzerRedactorReport from "./AnalyzerRedactorReport";
import PodAnalyzerDetails from "./PodAnalyzerDetails";
import ErrorModal from "../modals/ErrorModal";
import { Utilities } from "../../utilities/utilities";
import { Repeater } from "@src/utilities/repeater";
import "../../scss/components/troubleshoot/SupportBundleAnalysis.scss";
import "@src/scss/components/AirgapUploadProgress.scss";
import download from "downloadjs";
import Icon from "../Icon";

import { KotsPageTitle } from "@components/Head";
import { isEmpty } from "lodash";
import { useSelectedApp } from "@features/App";

let percentage;
export class SupportBundleAnalysis extends React.Component {
  constructor(props) {
    super();
    this.state = {
      activeTab:
        props.location.pathname.indexOf("/contents") !== -1
          ? "fileTree"
          : location.pathname.indexOf("/redactor") !== -1
          ? "redactorReport"
          : "bundleAnalysis",
      filterTiles: "0",
      downloadingBundle: false,
      downloadBundleErrMsg: "",
      sendingBundle: false,
      sendingBundleErrMsg: "",
      showPodAnalyzerDetailsModal: false,
    };
    this.pollingRef = React.createRef();
  }

  togglePodDetailsModal = (selectedPod) => {
    this.setState({
      showPodAnalyzerDetailsModal: !this.state.showPodAnalyzerDetailsModal,
      selectedPod,
    });
  };

  sendBundleToVendor = async () => {
    this.setState({
      sendingBundle: true,
      sendingBundleErrMsg: "",
      downloadBundleErrMsg: "",
    });
    fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/app/${this.props.params.slug}/supportbundle/${this.props.params.bundleSlug}/share`,
      {
        method: "POST",
        credentials: "include",
      }
    )
      .then(async (result) => {
        if (!result.ok) {
          const text = await result.text();
          let msg = `Unable to send bundle to vendor: Status ${result.status}, please try again.`;
          if (text) {
            msg = `Unable to send bundle to vendor: ${text}`;
          }
          this.setState({ sendingBundle: false, sendingBundleErrMsg: msg });
          return;
        }
        await this.getSupportBundle();
        this.setState({ sendingBundle: false, sendingBundleErrMsg: "" });
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          sendingBundle: false,
          sendingBundleErrMsg: err
            ? `Unable to send bundle to vendor: ${err.message}`
            : "Something went wrong, please try again.",
        });
      });
  };

  downloadBundle = async (bundle) => {
    this.setState({
      downloadingBundle: true,
      downloadBundleErrMsg: "",
      sendingBundleErrMsg: "",
    });
    fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/supportbundle/${bundle.id}/download`,
      {
        method: "GET",
        credentials: "include",
      }
    )
      .then(async (result) => {
        if (!result.ok) {
          this.setState({
            downloadingBundle: false,
            downloadBundleErrMsg: `Unable to download bundle: Status ${result.status}, please try again.`,
          });
          return;
        }

        let filename = "";
        const disposition = result.headers.get("Content-Disposition");
        if (disposition) {
          filename = disposition.split("filename=")[1];
        } else {
          const createdAt = dayjs(bundle.createdAt).format(
            "YYYY-MM-DDTHH_mm_ss"
          );
          filename = `supportbundle-${createdAt}.tar.gz`;
        }

        const blob = await result.blob();
        download(blob, filename, "application/gzip");

        this.setState({ downloadingBundle: false, downloadBundleErrMsg: "" });
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          downloadingBundle: false,
          downloadBundleErrMsg: err
            ? `Unable to download bundle: ${err.message}`
            : "Something went wrong, please try again.",
        });
      });
  };

  getSupportBundle = async () => {
    this.props.outletContext.updateState({
      displayErrorModal: false,
      getSupportBundleErrMsg: "",
      loading: true,
    });

    fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/supportbundle/${this.props.params.bundleSlug}`,
      {
        headers: {
          "Content-Type": "application/json",
        },
        method: "GET",
        credentials: "include",
      }
    )
      .then(async (res) => {
        if (!res.ok) {
          this.props.outletContext.updateState({
            displayErrorModal: true,
            getSupportBundleErrMsg: `Unexpected status code: ${res.status}`,
            loading: false,
          });
          return;
        }
        const bundle = await res.json();
        this.props.outletContext.updateState({
          displayErrorModal: false,
          getSupportBundleErrMsg: "",
          loading: false,
          bundle: bundle,
        });

        if (bundle.status === "running") {
          this.pollingRef.current = setInterval(() => {
            this.props.outletContext.pollForBundleAnalysisProgress();
          }, 1000);
        }
      })
      .catch((err) => {
        console.log(err);
        this.props.outletContext.updateState({
          displayErrorModal: false,
          loading: false,
          getSupportBundleErrMsg: err
            ? err.message
            : "Something went wrong, please try again.",
        });
      });
  };

  toggleErrorModal = () => {
    this.props.outletContext.updateState({
      displayErrorModal: !this.props.outletContext.displayErrorModal,
    });
  };

  toggleAnalysisAction = (active) => {
    this.setState({
      activeTab: active,
    });
  };

  moveBar(progressData) {
    const elem = document.getElementById("supportBundleStatusBar");
    const calcPercent =
      (progressData.collectorsCompleted / progressData.collectorCount) * 100;
    percentage = calcPercent > 98 ? 98 : calcPercent.toFixed();
    if (elem) {
      elem.style.width = percentage + "%";
    }
  }

  componentDidMount() {
    this.getSupportBundle();
  }
  componentWillUnmount() {
    clearInterval(this.pollingRef.current);
  }

  componentDidUpdate = (lastProps) => {
    const { location } = this.props;
    const { bundle } = this.props.outletContext;
    if (location !== lastProps.location) {
      this.setState({
        activeTab:
          location.pathname.indexOf("/contents") !== -1
            ? "fileTree"
            : location.pathname.indexOf("/redactor") !== -1
            ? "redactorReport"
            : "bundleAnalysis",
      });
    }
    if (
      bundle?.status !== "running" &&
      bundle?.status !== lastProps.outletContext.bundle.status
    ) {
      clearInterval(this.pollingRef.current);
    }
  };

  render() {
    const { watch } = this.props.outletContext;

    const { bundleProgress, getSupportBundleErrMsg, loading, bundle } =
      this.props.outletContext;

    if (loading) {
      return (
        <div className="flex-column flex1 justifyContent--center alignItems--center">
          <Loader size="60" />
        </div>
      );
    }

    // TODO: make this into a reusable component
    let progressBar;

    if (bundleProgress?.collectorsCompleted > 0) {
      this.moveBar(bundleProgress);
      progressBar = (
        <div className="progressbar">
          <div
            className="progressbar-meter"
            id="supportBundleStatusBar"
            style={{ width: "0px" }}
          />
        </div>
      );
    } else {
      percentage = "0";
      progressBar = (
        <div className="progressbar">
          <div
            className="progressbar-meter"
            id="supportBundleStatusBar"
            style={{ width: "0px" }}
          />
        </div>
      );
    }

    let statusDiv = (
      <div className="u-marginTop--20 u-fontWeight--medium u-lineHeight--medium u-textAlign--center">
        <div className="flex flex1 u-marginBottom--10 justifyContent--center alignItems--center u-textColor--secondary">
          {bundleProgress?.message && (
            <Loader className="flex u-marginRight--5" size="24" />
          )}
          {percentage >= 98 ? (
            <p>Almost done, finalizing your bundle...</p>
          ) : (
            <p>Analyzing {bundleProgress?.message}</p>
          )}
        </div>
      </div>
    );

    const insightsUrl = `/app/:slug/troubleshoot/analyze/:bundleSlug`;
    const fileTreeUrl = `/app/:slug/troubleshoot/analyze/:bundleSlug/contents/*`;
    const redactorUrl = `/app/:slug/troubleshoot/analyze/:bundleSlug/redactor/report`;

    const showSendSupportBundleBtn =
      watch.isSupportBundleUploadSupported && !watch.isAirgap;

    const context = {
      status: bundle.status,
      refetchSupportBundle: this.getSupportBundle,
      insights: bundle.analysis?.insights,
      openPodDetailsModal: this.togglePodDetailsModal,
      watchSlug: watch.slug,
      bundle: bundle,
      downloadBundle: () => this.downloadBundle(bundle),
    };

    return (
      <div className="container u-marginTop--20 u-paddingBottom--30 flex1 flex-column">
        <KotsPageTitle pageName="Support Bundle Analysis" showAppSlug />
        <div className="flex1 flex-column">
          {bundle && (
            <div className="flex1 flex-column">
              <div className="u-position--relative flex-auto u-marginBottom--20 flex justifyContent--spaceBetween">
                <div className="flex flex1 u-marginTop--10 u-marginBottom--10">
                  <div className="flex-column flex1">
                    <div className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginBottom--20">
                      <Link
                        to={`/app/${this.props.params.slug}/troubleshoot`}
                        className="link u-marginRight--5"
                      >
                        Support bundles
                      </Link>{" "}
                      &gt;{" "}
                      <span className="u-marginLeft--5">
                        {dayjs(bundle.createdAt).format("MMMM D, YYYY")}
                      </span>
                    </div>
                    <div className="flex flex1 justifyContent--spaceBetween">
                      <div className="flex flex-column">
                        <h2 className="u-fontSize--header2 u-fontWeight--bold card-item-title flex alignContent--center alignItems--center">
                          Support bundle analysis
                        </h2>
                      </div>
                    </div>
                    <div className="upload-date-container flex u-marginTop--5 alignItems--center">
                      <div className="flex alignSelf--center">
                        <p className="flex u-fontSize--normal u-textColor--bodyCopy u-fontWeight--medium">
                          Collected on{" "}
                          <span className="u-fontWeight--bold u-marginLeft--5">
                            {dayjs(bundle.createdAt).format(
                              "MMMM D, YYYY @ h:mm a"
                            )}
                          </span>
                        </p>
                      </div>
                    </div>
                  </div>
                  {this.props.outletContext.bundle.status !== "running" && (
                    <div className="flex flex-auto alignItems--center justifyContent--flexEnd">
                      {this.state.downloadBundleErrMsg && (
                        <p className="u-textColor--error u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-marginRight--10">
                          {this.state.downloadBundleErrMsg}
                        </p>
                      )}
                      {this.state.sendingBundleErrMsg && (
                        <p className="u-textColor--error u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-marginRight--10">
                          {this.state.sendingBundleErrMsg}
                        </p>
                      )}
                      {showSendSupportBundleBtn &&
                        (this.state.sendingBundle ? (
                          <Loader className="u-marginRight--10" size="30" />
                        ) : bundle.sharedAt ? (
                          <div className="sentToVendorWrapper flex alignItems--flexEnd u-paddingLeft--10 u-paddingRight--10 u-marginRight--10">
                            <Icon
                              icon="paper-airplane"
                              size={16}
                              style={{ marginRight: 7 }}
                            />
                            <span className="u-fontWeight--bold u-fontSize--small u-color--mutedteal">
                              Sent to vendor on{" "}
                              {Utilities.dateFormat(
                                bundle.sharedAt,
                                "MM/DD/YYYY"
                              )}
                            </span>
                          </div>
                        ) : !this.props.outletContext.watch.isAirgap ? (
                          <button
                            className="btn primary lightBlue u-marginRight--10"
                            onClick={this.sendBundleToVendor}
                          >
                            Send bundle to vendor
                          </button>
                        ) : null)}
                      {this.state.downloadingBundle ? (
                        <Loader size="30" />
                      ) : (
                        <button
                          className={`btn ${
                            showSendSupportBundleBtn
                              ? "secondary blue"
                              : "primary lightBlue"
                          }`}
                          onClick={() => this.downloadBundle(bundle)}
                        >
                          {" "}
                          Download bundle{" "}
                        </button>
                      )}
                    </div>
                  )}
                </div>
              </div>
              {watch.licenseType === "community" && (
                <div className="flex">
                  <div className="CommunityLicenseBundle--wrapper flex flex1 alignItems--center">
                    <div className="flex flex-auto">
                      <span className="icon communityIcon"></span>
                    </div>
                    <div className="flex1 flex-column u-marginLeft--10">
                      <p className="u-textColor--accent u-fontSize--large u-fontWeight--bold u-lineHeight--medium u-marginBottom--5">
                        {" "}
                        This bundle was uploaded by a customer under a Community
                        license type.{" "}
                      </p>
                      <p className="u-textColor--info u-fontSize--normal u-lineHeight--medium">
                        {" "}
                        Customers with Community licenses are using the free,
                        Community-Supported version of{" "}
                        {this.props.outletContext.watch.name}.{" "}
                      </p>
                    </div>
                  </div>
                </div>
              )}
              <div className="flex-column flex1">
                {bundle.status === "running" &&
                isEmpty(bundle.analysis?.insights) ? (
                  <div className="flex flex-column flex1 justifyContent--center alignItems--center u-marginTop--20 SupportBundleDetails--Progress">
                    <div className="flex justifyContent--center alignItems--center">
                      <span className="u-fontWeight--bold u-fontSize--normal u-textColor--secondary u-marginRight--10">
                        {percentage + "%"}
                      </span>
                      {progressBar}
                      <span className="u-fontWeight--bold u-fontSize--normal u-textColor--secondary u-marginRight--10">
                        100%
                      </span>
                    </div>
                    {statusDiv}
                  </div>
                ) : (
                  <div className="SupportBundleTabs--wrapper flex1 flex-column">
                    <div className="tab-items flex">
                      <Link
                        to={`/app/${this.props.params.slug}/troubleshoot/analyze/${bundle.slug}`}
                        className={`${
                          this.state.activeTab === "bundleAnalysis"
                            ? "is-active"
                            : ""
                        } tab-item blue`}
                        onClick={() =>
                          this.toggleAnalysisAction("bundleAnalysis")
                        }
                      >
                        Analysis insights
                      </Link>
                      <Link
                        to={`/app/${this.props.params.slug}/troubleshoot/analyze/${bundle.slug}/contents/`}
                        className={`${
                          this.state.activeTab === "fileTree" ? "is-active" : ""
                        } tab-item blue`}
                        onClick={() => this.toggleAnalysisAction("fileTree")}
                      >
                        File inspector
                      </Link>
                      <Link
                        to={`/app/${this.props.params.slug}/troubleshoot/analyze/${bundle.slug}/redactor/report`}
                        className={`${
                          this.state.activeTab === "redactorReport"
                            ? "is-active"
                            : ""
                        } tab-item blue`}
                        onClick={() =>
                          this.toggleAnalysisAction("redactorReport")
                        }
                      >
                        Redactor report
                      </Link>
                    </div>
                    <div className="flex-column flex1 action-content">
                      <Outlet context={context} />
                    </div>
                  </div>
                )}
              </div>
            </div>
          )}
        </div>
        {getSupportBundleErrMsg && (
          <ErrorModal
            errorModal={this.props.outletContext.displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            errMsg={getSupportBundleErrMsg}
            tryAgain={this.getSupportBundle}
            err="Failed to get bundle"
            loading={this.props.outletContext.loading}
            appSlug={this.props.params.slug}
          />
        )}
        {this.state.showPodAnalyzerDetailsModal && (
          <Modal
            isOpen={true}
            shouldReturnFocusAfterClose={false}
            onRequestClose={() => this.togglePodDetailsModal({})}
            ariaHideApp={false}
            contentLabel="Modal"
            className="Modal PodAnalyzerDetailsModal LargeSize"
          >
            <div className="Modal-body">
              <PodAnalyzerDetails
                bundleId={bundle.id}
                pod={this.state.selectedPod}
              />
              <div className="u-marginTop--10">
                <button
                  type="button"
                  className="btn primary blue"
                  onClick={() => this.togglePodDetailsModal({})}
                >
                  Ok, got it!
                </button>
              </div>
            </div>
          </Modal>
        )}
      </div>
    );
  }
}

export default withRouter(SupportBundleAnalysis);
