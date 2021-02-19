import React, { Component } from "react";
import { withRouter } from "react-router-dom";
import Helmet from "react-helmet";
import { Line } from "rc-progress";
import Loader from "../shared/Loader";
import { Utilities } from "@src/utilities/utilities";
import { Repeater } from "@src/utilities/repeater";
import "../../scss/components/snapshots/AppSnapshots.scss";

class SnapshotRestore extends Component {
  state = {
    fetchRestoreDetailJob: new Repeater(),
    loadingRestoreDetail: true,
    restoreDetail: [],
    errorMessage: "",
    errorTitle: "",

    cancelingRestore: false,
    cancelRestoreErr: "",
    cancelRestoreErrorMsg: "",
  }

  componentDidMount() {
    this.state.fetchRestoreDetailJob.start(this.fetchRestoreDetail, 2000);
  }

  componentDidUpdate(lastProps) {
    const { match } = this.props;
    const appSlug = match.params.slug;
    if (match.params.id !== lastProps.match.params.id) {
      this.state.fetchRestoreDetailJob.start(this.fetchRestoreDetail, 2000);
    } else {
      const currentAppDetails = this.state.restoreDetail?.find(a => a.appSlug === appSlug);
      const phase = currentAppDetails?.restoreDetail?.phase;
      if (phase && phase !== "New" && phase !== "InProgress") {
        this.state.fetchRestoreDetailJob.stop();
      }
    }
  }

  componentWillUnmount() {
    this.state.fetchRestoreDetailJob.stop();
  }

  fetchRestoreDetail = async () => {
    const { match } = this.props;
    const restoreName = match.params.id;
    const appSlug = match.params.slug;

    this.setState({
      errorMessage: "",
      errorTitle: "",
    });

    try {
      const res = await fetch(`${window.env.API_ENDPOINT}/snapshot/${restoreName}/apps-restore-status`, {
        method: "POST",
        headers: {
          "Authorization": Utilities.getToken(),
        },
        body: JSON.stringify({
          appSlugs: [appSlug]
        })
      });
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }
        this.setState({
          loadingRestoreDetail: false,
          errorMessage: `Unexpected status code: ${res.status}`,
          errorTitle: "Failed to fetch restore details",
        });
        return;
      }
      const response = await res.json();

      const restoreDetail = response.statuses;

      this.setState({
        loadingRestoreDetail: false,
        restoreDetail: restoreDetail,
        errorMessage: "",
        errorTitle: "",
      });
    } catch (err) {
      console.log(err);
      this.setState({
        loadingRestoreDetail: false,
        errorMessage: err ? `${err.message}` : "Something went wrong, please try again.",
        errorTitle: "Failed to fetch restore details",
      });
    }
  }

  logOutUser = () => {
    const token = Utilities.getToken();
    if (token) {
      window.localStorage.removeItem("token");
    }
  }

  onCancelRestore = async () => {
    this.setState({ cancelingRestore: true, cancelRestoreErr: false, cancelRestoreErrorMsg: "" });
    try {
      await this.fetchCancelRestore();
      this.props.history.push("/snapshots");
    } catch (err) {
      this.setState({
        cancelRestoreErr: true,
        cancelRestoreErrorMsg: err ? `${err.message}` : "Something went wrong, please try again.",
      });
    }
    this.setState({ cancelingRestore: false });
  }

  fetchCancelRestore = async () => {
    const { match } = this.props;
    const appSlug = match.params.slug;
    const res = await fetch(`${window.env.API_ENDPOINT}/app/${appSlug}/snapshot/restore`, {
      method: "DELETE",
      headers: {
        "Authorization": Utilities.getToken(),
      }
    });
    if (!res.ok) {
      if (res.status === 401) {
        Utilities.logoutUser();
        return;
      }
      throw new Error(`Unexpected status code: ${res.status}`);
    }
  }

  renderErrors = (errors) => {
    return (
      errors?.map((error, i) => (
        <div className="RestoreError--wrapper flex justifyContent--space-between alignItems--center" key={`${error.title}-${i}`}>
          <div className="flex-auto icon error-small u-marginRight--10 u-marginLeft--10"> </div>
          <p className="u-fontSize--normal u-fontWeight--normal u-lineHeight--normal">{error.message}</p>
        </div>
      ))
    );
  }

  renderWarnings = (warnings) => {
    return (
      warnings?.map((warning, i) => (
        <div className="RestoreWarning--wrapper flex justifyContent--space-between alignItems--center" key={`${warning.title}-${i}`}>
          <div className="flex-auto icon exclamationMark--icon u-marginRight--10 u-marginLeft--10"> </div>
          <p className="u-fontSize--normal u-fontWeight--normal u-lineHeight--normal">{warning.message}</p>
        </div>
      ))
    );
  }

  renderFailedRestoreView = (detail) => {
    if (detail?.warnings?.length > 0 && (!detail?.errors || detail?.errors?.length === 0)) {
      return this.renderWarningsRestoreView(detail?.warnings);
    } else if (detail?.errors?.length > 0) {
      return (
        <div className="FailedRestore--wrapper">
          <div className="flex flex-column alignItems--center">
            <span className="icon u-superWarning--large"></span>
            <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-lineHeight--normal u-marginTop--15 u-marginBottom--10">
              Application failed to restore </p>
            {detail?.warnings?.length > 0 ?
              <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal">
                Your application failed to restore to
                <span className="u-fontWeight--bold u-color--dustyGray"> {this.props.match.params.id} </span> because of errors. During the restore there were
                <span className="u-fontWeight--bold  u-color--tundora"> {detail?.warnings?.length} {detail?.warnings?.length === 1 ? "warning" : "warnings"} </span> and
                <span className="u-fontWeight--bold u-color--tundora"> {detail?.errors?.length} {detail?.errors?.length === 1 ? "error" : "errors"}. </span>
              </p>
              :
              <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal">
                Your application failed to restore to
                <span className="u-fontWeight--bold u-color--dustyGray"> {this.props.match.params.id} </span> because of errors. During the restore there {detail?.errors?.length === 1 ? "was" : "were"}
                <span className="u-fontWeight--bold u-color--tundora"> {detail?.errors?.length} {detail?.errors?.length === 1 ? "error" : "errors"}. </span>
              </p>
            }
          </div>
          <div className="u-marginTop--30">
            {this.renderErrors(detail?.errors)}
            {detail?.warnings?.length > 0 && this.renderWarnings(detail?.warnings)}
          </div>
          <div className="flex alignItems--center justifyContent--center">
            <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal u-marginTop--30"> Contact your vendor for help troubleshooting this restore. </p>
          </div>
        </div>
      )
    } else {
      return (
        <div className="FailedRestore--wrapper">
          <div className="flex flex-column alignItems--center">
            <span className="icon u-superWarning--large"></span>
            <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-lineHeight--normal u-marginTop--15 u-marginBottom--10">
              Your partial restore failed </p>
            <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal">
              Your application failed to restore to  <span className="u-fontWeight--bold u-color--dustyGray"> {this.props.match.params.id} </span>
            </p>
          </div>
          <div className="flex alignItems--center justifyContent--center">
            <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal u-marginTop--30"> Contact your vendor for help troubleshooting this restore. </p>
          </div>
        </div>
      )
    }
  }

  renderWarningsRestoreView = (warnings) => {
    return (
      <div className="FailedRestore--wrapper">
        <div className="flex flex-column alignItems--center">
          <span className="icon yellowWarningIcon"></span>
          <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-lineHeight--normal u-marginTop--15 u-marginBottom--10">
            Partial restore was restored with warnings </p>
          <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal">
            Your application restored  to <span className="u-fontWeight--bold u-color--dustyGray"> {this.props.match.params.id} </span> but there were warnings that my affect the application. During the restore there were
          <span className="u-fontWeight--bold  u-color--tundora"> {warnings?.length} warnings </span>.</p>
        </div>
        <div className="u-marginTop--30">
          {this.renderWarnings(warnings)}
        </div>
        <div className="flex alignItems--center justifyContent--center">
          <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal u-marginTop--30"> Contact your vendor for help troubleshooting this restore. </p>
        </div>
      </div>
    )
  }

  render() {
    const { match } = this.props;
    const { cancelingRestore, restoreDetail, loadingRestoreDetail } = this.state;
    const appSlug = match.params.slug;

    const currentAppDetails = restoreDetail.find(a => a.appSlug === appSlug);

    const hasNoErrorsOrWarnings = (!currentAppDetails?.restoreDetail?.warning && !currentAppDetails?.restoreDetail?.errors) ||
      (currentAppDetails?.restoreDetail?.warnings?.length === 0 && currentAppDetails?.restoreDetail?.errors?.length === 0);
    const restoreCompleted = currentAppDetails?.restoreDetail?.phase === "Completed";
    const restoreFailing = currentAppDetails?.restoreDetail?.phase === "PartiallyFailed" || restoreDetail?.phase === "Failed";
    const restoreLoading = !currentAppDetails?.restoreDetail?.warnings && !currentAppDetails?.restoreDetail?.errors;

    if (loadingRestoreDetail) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      )
    }

    if (restoreCompleted && hasNoErrorsOrWarnings) {
      this.logOutUser();
      this.props.history.push("/restore-completed")
    }

    return (
      <div className="container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20 alignItems--center">
        <Helmet>
          <title>{`${this.props.match.params.slug} Snapshots Restore`}</title>
        </Helmet>
        {!restoreCompleted && !restoreFailing ?
          <div className="flex1 flex-column alignItems--center">
            <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-lineHeight--normal u-marginBottom--10"> Partial restore in progress </p>
            <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal"> After all volumes have been restored you will need to log back in to the admin console. </p>
            <div className="flex flex-column  u-marginTop--40">
              {restoreLoading &&
                <div className="flex-column flex1 alignItems--center justifyContent--center">
                  <Loader size="60" />
                </div>
              }
              {currentAppDetails?.restoreDetail?.volumes?.map((volume, i) => {
                const strokeColor = volume.completionPercent === 100 ? "#44BB66" : "#326DE6";
                const minutes = Math.floor(volume.timeRemainingSeconds / 60);
                const remainingTime = volume.timeRemainingSeconds < 60 ? `${volume.timeRemainingSeconds} seconds remaining` : `${minutes} minutes remaining`;
                const percentage = volume.completionPercent ? volume.completionPercent : 0;

                return (
                  <div className="flex flex1 u-marginTop--30 alignItems--center" key={`${volume.name}-${i}`}>
                    <div className="flex flex1">
                      <p className="u-fontSize--normal u-color--tuna u-fontWeight--bold u-lineHeight--bold u-marginRight--10">Restoring volume: {volume.name}</p>
                    </div>
                    <div className="flex flex1 flex-column justifyContent--center">
                      <Line percent={percentage} strokeWidth="3" strokeColor={strokeColor} />
                      {volume.completionPercent === 100 ?
                        <div className="flex justifyContent--center u-fontSize--smaller u-fontWeight--medium u-color--silverSand u-marginTop--5"> Complete </div>
                        : volume.remainingSecondsExist ?
                          <div className="flex justifyContent--center u-fontSize--smaller u-fontWeight--medium u-color--silverSand u-marginTop--5"> {volume.timeRemainingSeconds ? remainingTime : null}</div>
                          :
                          <div className="flex justifyContent--center u-fontSize--smaller u-fontWeight--medium u-color--silverSand u-marginTop--5"> In progress </div>
                      }
                    </div>
                    {volume.completionPercent === 100 ?
                      <span className="icon checkmark-icon u-marginLeft--10 u-marginBottom--15"></span>
                      :
                      <span className="u-fontSize-small u-fontWeight--medium u-color--silverSand u-marginLeft--10 u-marginBottom--15">{volume.completionPercent ? `${volume.completionPercent}%` : null}</span>
                    }
                  </div>
                );
              })}
            </div>
            <div className="flex alignItems--center justifyContent--center u-marginTop--40">
              <button className="btn secondary red" onClick={this.onCancelRestore} disabled={cancelingRestore}>{cancelingRestore ? "Canceling..." : "Cancel snapshot restore"}</button>
            </div>
          </div>
          :
          !hasNoErrorsOrWarnings || restoreFailing ?
            this.renderFailedRestoreView(currentAppDetails?.restoreDetail)
            : null
        }
      </div>
    );
  }
}

export default withRouter(SnapshotRestore);
