import React, { useEffect, useContext } from "react";
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
import { useNavigate } from "react-router-dom";
import { ToastContext } from "@src/context/ToastContext";

let percentage: number;

type Props = {
  bundle: SupportBundle;
  isAirgap: boolean;
  isCustomer: boolean;
  isSupportBundleUploadSupported: boolean;
  loadingBundle: boolean;
  progressData: SupportBundleProgress;
  refetchBundleList: () => void;
  //deleteBundleFromList: (id: string) => void;
  watchSlug: string;
  className: string;
} & withRouterType;

type State = {
  downloadBundleErrMsg?: string;
  downloadingBundle: boolean;
  errorInsights?: SupportBundleInsight[];
  otherInsights?: SupportBundleInsight[];
  sendingBundle: boolean;
  sendingBundleErrMsg?: string;
  warningInsights?: SupportBundleInsight[];
  timeoutId?: ReturnType<typeof setTimeout>;
};

export const SupportBundleRow = (props: Props) => {
  const navigate = useNavigate();
  const {
    setIsToastVisible,
    isCancelled,
    setIsCancelled,
    setDeleteBundleId,
    setToastMessage,
    setToastType,
    setToastChild,
  } = useContext(ToastContext);

  const [state, setState] = React.useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      downloadingBundle: false,
      sendingBundle: false,
    }
  );

  const renderSharedContext = () => {
    const { bundle } = props;
    if (!bundle) {
      return null;
    }
  };

  const buildInsights = () => {
    const { bundle } = props;
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
    setState({
      errorInsights,
      warningInsights,
      otherInsights,
    });
  };

  useEffect(() => {
    if (props.bundle) {
      buildInsights();
    }
  }, []);

  const handleBundleClick = (bundle: SupportBundle) => {
    const { watchSlug } = props;
    navigate(`/app/${watchSlug}/troubleshoot/analyze/${bundle.slug}`);
  };

  const downloadBundle = async (bundle: SupportBundle) => {
    setState({ downloadingBundle: true, downloadBundleErrMsg: "" });
    fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/supportbundle/${bundle.id}/download`,
      {
        method: "GET",
        credentials: "include",
      }
    )
      .then(async (result) => {
        if (!result.ok) {
          setState({
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

        setState({ downloadingBundle: false, downloadBundleErrMsg: "" });
      })
      .catch((err) => {
        console.log(err);
        setState({
          downloadingBundle: false,
          downloadBundleErrMsg: err
            ? `Unable to download bundle: ${err.message}`
            : "Something went wrong, please try again.",
        });
      });
  };

  useEffect(() => {
    if (isCancelled && state.timeoutId) {
      clearTimeout(state.timeoutId);
      setIsToastVisible(false);
      setDeleteBundleId("");
      setIsCancelled(false);
    }
  }, [isCancelled]);

  const deleteBundle = (bundle: SupportBundle) => {
    const { match } = props;
    const delayFetch = 7000;
    const bundleCollectionDate = dayjs(bundle?.createdAt)?.format(
      "MMMM D, YYYY @ h:mm a"
    );
    setToastMessage(`Deleting bundle collected on ${bundleCollectionDate}.`);
    setToastType("warning");
    setIsToastVisible(true);
    setDeleteBundleId(bundle.id);
    setToastChild(
      <span
        onClick={() => setIsCancelled(true)}
        className="tw-underline tw-cursor-pointer"
      >
        undo
      </span>
    );

    let id = setTimeout(async () => {
      const res = await fetch(
        `${process.env.API_ENDPOINT}/troubleshoot/app/${match.params.slug}/supportbundle/${bundle.id}`,
        {
          method: "DELETE",
          credentials: "include",
        }
      );
      if (res.ok) {
        //await props.deleteBundleFromList(bundle.id);
        setIsToastVisible(false);
        props.refetchBundleList();
        clearInterval(id);
      } else {
        console.log(res);
        setToastMessage("Unable to delete bundle, please try again.");
        setToastType("error");
        setToastChild(null);
        setDeleteBundleId("");
        setTimeout(() => {
          setIsToastVisible(false);
        }, 5000);
        clearInterval(id);
      }
    }, delayFetch);

    setState({ timeoutId: id });
  };

  const sendBundleToVendor = async (bundleSlug: string) => {
    setState({
      sendingBundle: true,
      sendingBundleErrMsg: "",
      downloadBundleErrMsg: "",
    });
    fetch(
      `${process.env.API_ENDPOINT}/troubleshoot/app/${props.match.params.slug}/supportbundle/${bundleSlug}/share`,
      {
        method: "POST",
        credentials: "include",
      }
    )
      .then(async (result) => {
        if (!result.ok) {
          setState({
            sendingBundle: false,
            sendingBundleErrMsg: `Unable to send bundle to vendor: Status ${result.status}, please try again.`,
          });
          return;
        }
        await props.refetchBundleList();
        setState({ sendingBundle: false, sendingBundleErrMsg: "" });
      })
      .catch((err) => {
        console.log(err);
        setState({
          sendingBundle: false,
          sendingBundleErrMsg: err
            ? `Unable to send bundle to vendor: ${err.message}`
            : "Something went wrong, please try again.",
        });
      });
  };

  const moveBar = (progressData: SupportBundleProgress) => {
    const elem = document.getElementById("supportBundleStatusBar");
    const calcPercent = Math.round(
      (progressData.collectorsCompleted / progressData.collectorCount) * 100
    );
    percentage = calcPercent > 98 ? 98 : calcPercent;
    if (elem) {
      elem.style.width = percentage.toString() + "%";
    }
  };

  const {
    bundle,
    isSupportBundleUploadSupported,
    isAirgap,
    progressData,
    loadingBundle,
  } = props;
  const { errorInsights, warningInsights, otherInsights } = state;

  const showSendSupportBundleLink = isSupportBundleUploadSupported && !isAirgap;

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
    moveBar(progressData);
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
        <div className={`bundle-row-wrapper card-item ${props.className}`}>
          <div className="bundle-row flex flex1">
            <div
              className="flex flex1 flex-column"
              onClick={() => handleBundleClick(bundle)}
            >
              <div className="flex">
                {!props.isCustomer ? (
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
                      {renderSharedContext()}
                    </span>
                  </div>
                )}
              </div>
              <div className="flex u-marginTop--15">
                {props.loadingBundle ? (
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
              {state.sendingBundleErrMsg && (
                <p className="u-textColor--error u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-marginRight--10">
                  {state.sendingBundleErrMsg}
                </p>
              )}
              {props.bundle.sharedAt ? (
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
              ) : state.sendingBundle ? (
                <Loader size="30" className="u-marginRight--10" />
              ) : showSendSupportBundleLink && !loadingBundle ? (
                <span
                  className="u-fontSize--small u-marginRight--10 link u-textDecoration--underlineOnHover u-paddingRight--10"
                  onClick={() => sendBundleToVendor(props.bundle.slug)}
                >
                  <Icon icon="paper-airplane" size={16} className="clickable" />
                </span>
              ) : null}
              {state.downloadBundleErrMsg && (
                <p className="u-textColor--error u-fontSize--normal u-fontWeight--medium u-lineHeight--normal u-marginRight--10">
                  {state.downloadBundleErrMsg}
                </p>
              )}
              {state.downloadingBundle ? (
                <Loader size="30" />
              ) : props.loadingBundle ||
                props.progressData?.collectorsCompleted > 0 ? (
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
                  className="u-fontSize--small link u-textDecoration--underlineOnHover"
                  onClick={() => downloadBundle(bundle)}
                >
                  <Icon icon="download" size={16} className="clickable" />
                </span>
              )}
              <span
                className="u-fontSize--small link u-textDecoration--underlineOnHover"
                onClick={() => deleteBundle(bundle)}
              >
                <Icon
                  icon="trash"
                  size={16}
                  className={"tw-ml-2 error-color clickable"}
                />
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};

/* eslint-disable */
// @ts-ignore
export default withRouter(SupportBundleRow) as any;
/* eslint-enable*/
