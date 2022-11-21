import * as React from "react";
import {
  withRouter,
  withRouterType,
} from "@src/utilities/react-router-utilities";
import Loader from "../shared/Loader";
import dayjs from "dayjs";
import filter from "lodash/filter";
import isEmpty from "lodash/isEmpty";
import { Utilities } from "../../utilities/utilities";
import download from "downloadjs";
import Icon from "../Icon";
import "@src/scss/components/AirgapUploadProgress.scss";

import {
  SupportBundle,
  SupportBundleInsight,
  SupportBundleProgress,
} from "@types";

let percentage: number;

type Props = {
  bundle: SupportBundle;
  isAirgap: boolean;
  isCustomer: boolean;
  isSupportBundleUploadSupported: boolean;
  loadingBundle: boolean;
  progressData: SupportBundleProgress;
  refetchBundleList: () => void;
  watchSlug: string;
} & withRouterType;

type State = {
  downloadBundleErrMsg?: string;
  downloadingBundle: boolean;
  errorInsights?: SupportBundleInsight[];
  otherInsights?: SupportBundleInsight[];
  sendingBundle: boolean;
  sendingBundleErrMsg?: string;
  warningInsights?: SupportBundleInsight[];
};

class SupportBundleRow extends React.Component<Props, State> {
  constructor(props: Props) {
    super(props);

    this.state = {
      downloadingBundle: false,
      sendingBundle: false,
    };
  }

  renderSharedContext = () => {
    const { bundle } = this.props;
    if (!bundle) {
      return null;
    }
  };

  componentDidMount() {
    if (this.props.bundle) {
      this.buildInsights();
    }
  }

  handleBundleClick = (bundle: SupportBundle) => {
    const { watchSlug } = this.props;
    this.props.history.push(
      `/app/${watchSlug}/troubleshoot/analyze/${bundle.slug}`
    );
  };

  downloadBundle = async (bundle: SupportBundle) => {
    this.setState({ downloadingBundle: true, downloadBundleErrMsg: "" });
    fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/supportbundle/${bundle.id}/download`,
      {
        method: "GET",
        headers: {
          Authorization: Utilities.getToken(),
        },
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

  sendBundleToVendor = async (bundleSlug: string) => {
    this.setState({
      sendingBundle: true,
      sendingBundleErrMsg: "",
      downloadBundleErrMsg: "",
    });
    fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/app/${this.props.match.params.slug}/supportbundle/${bundleSlug}/share`,
      {
        method: "POST",
        headers: {
          Authorization: Utilities.getToken(),
        },
      }
    )
      .then(async (result) => {
        if (!result.ok) {
          this.setState({
            sendingBundle: false,
            sendingBundleErrMsg: `Unable to send bundle to vendor: Status ${result.status}, please try again.`,
          });
          return;
        }
        await this.props.refetchBundleList();
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

  buildInsights = () => {
    const { bundle } = this.props;
    if (!bundle?.analysis?.insights) {
      return;
    }
    const errorInsights = filter(bundle.analysis.insights, [
      "severity",
      "error",
    ]);
    const warningInsights = filter(bundle.analysis.insights, [
      "severity",
      "warn",
    ]);
    const otherInsights = filter(bundle.analysis.insights, (item) => {
      return (
        item.severity === null ||
        item.severity === "info" ||
        item.severity === "debug"
      );
    });
    this.setState({
      errorInsights,
      warningInsights,
      otherInsights,
    });
  };

  moveBar(progressData: SupportBundleProgress) {
    const elem = document.getElementById("supportBundleStatusBar");
    const calcPercent = Math.round(
      (progressData.collectorsCompleted / progressData.collectorCount) * 100
    );
    percentage = calcPercent > 98 ? 98 : calcPercent;
    if (elem) {
      elem.style.width = percentage.toString() + "%";
    }
  }

  render() {
    const {
      bundle,
      isSupportBundleUploadSupported,
      isAirgap,
      progressData,
      loadingBundle,
    } = this.props;
    const { errorInsights, warningInsights, otherInsights } = this.state;

    const showSendSupportBundleLink =
      isSupportBundleUploadSupported && !isAirgap;

    if (!bundle) {
      return null;
    }

    let noInsightsMessage;
    if (bundle && isEmpty(bundle?.analysis?.insights?.length)) {
      if (bundle.status === "uploaded" || bundle.status === "analyzing") {
        noInsightsMessage = (
          <div className="flex">
            <Loader size="14" />
            <p className="u-fontSize--small u-fontWeight--medium u-marginLeft--5 u-textColor--accent">
              We are still analyzing your bundle
            </p>
          </div>
        );
      } else {
        noInsightsMessage = (
          <p className="u-fontSize--small u-fontWeight--medium u-textColor--accent">
            Unable to surface insights for this bundle
          </p>
        );
      }
    }

    let progressBar;

    let statusDiv = (
      <div className="u-fontWeight--bold u-fontSize--small .u-textColor--bodyCopy u-lineHeight--medium u-textAlign--center">
        <div className="flex flex1 u-marginBottom--10 justifyContent--center alignItems--center ">
          {progressData?.message && (
            <Loader className="flex u-marginRight--5" size="24" />
          )}
          {percentage >= 98 ? (
            <p>Almost done, finalizing your bundle...</p>
          ) : (
            <p>Analyzing {progressData?.message}</p>
          )}
        </div>
      </div>
    );

    if (progressData.collectorsCompleted > 0) {
      this.moveBar(progressData);
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
      percentage = 0;
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

    return (
      <div className="SupportBundle--Row u-position--relative">
        <div>
          <div className="bundle-row-wrapper card-item">
            <div className="bundle-row flex flex1">
              <div
                className="flex flex1 flex-column"
                onClick={() => this.handleBundleClick(bundle)}
              >
                <div className="flex">
                  {!this.props.isCustomer ? (
                    <div className="flex-column flex1 flex-verticalCenter">
                      <span className="u-fontSize--large card-item-title u-fontWeight--medium u-cursor--pointer card-item-title">
                        <span>
                          Collected on{" "}
                          <span className="u-fontWeight--bold">
                            {dayjs(bundle.createdAt).format(
                              "MMMM D, YYYY @ h:mm a"
                            )}
                          </span>
                        </span>
                      </span>
                    </div>
                  ) : (
                    <div className="flex-column flex1 flex-verticalCenter">
                      <span>
                        <span className="u-fontSize--large u-cursor--pointer u-textColor--primary u-fontWeight--medium">
                          Collected on{" "}
                          <span className="u-fontWeight--medium">
                            {dayjs(bundle.createdAt).format(
                              "MMMM D, YYYY @ h:mm a"
                            )}
                          </span>
                        </span>
                        {this.renderSharedContext()}
                      </span>
                    </div>
                  )}
                </div>
                <div className="flex u-marginTop--15">
                  {this.props.loadingBundle ? (
                    statusDiv
                  ) : bundle?.analysis?.insights?.length ? (
                    <div className="flex flex1 alignItems--center">
                      {errorInsights && errorInsights.length > 0 && (
                        <span className="flex alignItems--center u-marginRight--30 u-fontSize--small u-fontWeight--medium u-textColor--error">
                          <Icon
                            icon={"warning-circle-filled"}
                            size={15}
                            className="error-color u-marginRight--5"
                          />
                          {errorInsights.length} error
                          {errorInsights.length > 1 ? "s" : ""} found
                        </span>
                      )}
                      {warningInsights && warningInsights.length > 0 && (
                        <span className="flex alignItems--center u-marginRight--30 u-fontSize--small u-fontWeight--medium u-textColor--warning">
                          <Icon
                            icon="warning"
                            className="warning-color u-marginRight--5"
                            size={16}
                          />
                          {warningInsights.length} warning
                          {warningInsights.length > 1 ? "s" : ""} found
                        </span>
                      )}
                      {otherInsights && otherInsights.length > 0 && (
                        <span className="flex alignItems--center u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy">
                          <span className="icon u-bundleInsightOtherIcon u-marginRight--5" />
                          {otherInsights.length} informational and debugging
                          insight{otherInsights.length > 1 ? "s" : ""} found
                        </span>
                      )}
                    </div>
                  ) : (
                    noInsightsMessage
                  )}
                </div>
              </div>
              <div className="SupportBundleRow--Progress flex flex-auto alignItems--center justifyContent--flexEnd">
                {this.state.sendingBundleErrMsg && (
                  <p className="u-textColor--error u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-marginRight--10">
                    {this.state.sendingBundleErrMsg}
                  </p>
                )}
                {this.props.bundle.sharedAt ? (
                  <div className="sentToVendorWrapper flex alignItems--flexEnd u-paddingLeft--10 u-paddingRight--10 u-marginRight--10">
                    <Icon
                      icon="paper-airplane"
                      size={16}
                      className="u-marginRight--5"
                    />
                    <span className="u-fontWeight--bold u-fontSize--small u-color--mutedteal">
                      Sent to vendor on{" "}
                      {Utilities.dateFormat(bundle.sharedAt, "MM/DD/YYYY")}
                    </span>
                  </div>
                ) : this.state.sendingBundle ? (
                  <Loader size="30" className="u-marginRight--10" />
                ) : showSendSupportBundleLink && !loadingBundle ? (
                  <span
                    className="u-fontSize--small u-marginRight--10 u-linkColor u-fontWeight--medium u-textDecoration--underlineOnHover u-paddingRight--10"
                    onClick={() =>
                      this.sendBundleToVendor(this.props.bundle.slug)
                    }
                  >
                    <Icon
                      icon="paper-airplane"
                      size={16}
                      className="clickable"
                    />
                  </span>
                ) : null}
                {this.state.downloadBundleErrMsg && (
                  <p className="u-textColor--error u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-marginRight--10">
                    {this.state.downloadBundleErrMsg}
                  </p>
                )}
                {this.state.downloadingBundle ? (
                  <Loader size="30" />
                ) : this.props.loadingBundle ||
                  this.props.progressData?.collectorsCompleted > 0 ? (
                  <div
                    className="flex alignItems--center"
                    style={{ width: "350px" }}
                  >
                    <span className="u-fontWeight--bold u-fontSize--normal u-textColor--secondary u-marginRight--10">
                      {percentage.toString() + "%"}
                    </span>
                    {progressBar}
                    <span className="u-fontWeight--bold u-fontSize--normal u-textColor--secondary u-marginRight--10">
                      100%
                    </span>
                  </div>
                ) : (
                  <span
                    className="u-fontSize--small u-linkColor u-fontWeight--medium u-textDecoration--underlineOnHover"
                    onClick={() => this.downloadBundle(bundle)}
                  >
                    <Icon icon="download" size={16} className="clickable" />
                  </span>
                )}
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }
}

/* eslint-disable */
// @ts-ignore
export default withRouter(SupportBundleRow) as any;
/* eslint-enable*/
