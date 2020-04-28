import React, { Component } from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { withRouter } from "react-router-dom";
import Helmet from "react-helmet";
import { Line } from "rc-progress";
import Loader from "../shared/Loader";
import { Utilities } from "@src/utilities/utilities";
import { restoreDetail } from "../../queries/SnapshotQueries";
import { cancelRestore } from "../../mutations/SnapshotMutations";
import "../../scss/components/snapshots/AppSnapshots.scss";

class AppSnapshotRestore extends Component {
  state = {
    cancelingRestore: false,
    cancelRestoreErr: "",
    cancelRestoreErrorMsg: ""
  }

  componentDidMount() {
    this.props.restoreDetail.startPolling(2000);
  }

  componentDidUpdate(lastProps) {
    if (this.props.restoreDetail?.restoreDetail !== lastProps.restoreDetail?.restoreDetail && this.props.restoreDetail?.restoreDetail) {
      const phase = this.props.restoreDetail?.restoreDetail?.phase;
      if (phase !== "New" && phase !== "InProgress") {
        this.props.restoreDetail.stopPolling();
      }
    }
  }

  logOutUser = () => {
    const token = Utilities.getToken();
    if (token) {
      window.localStorage.removeItem("token");
    }
  }

  onCancelRestore = () => {
    const { app } = this.props;
    this.setState({ cancelingRestore: true, cancelRestoreErr: false, cancelRestoreErrorMsg: "" });
    this.props.cancelRestore(app.id)
      .then(() => {
        this.setState({ cancelingRestore: false });
        this.props.history.push(`/app/${this.props.app?.slug}/snapshots`);
      })
      .catch(err => {
        err.graphQLErrors.map(({ msg }) => {
          this.setState({
            cancelingRestore: false,
            cancelRestoreErr: true,
            cancelRestoreErrorMsg: msg,
          });
        })
      })
      .finally(() => {
        this.setState({ cancelingRestore: false });
      });
  }

  renderErrors = (errors) => {
    return (
      errors.map((error, i) => (
        <div className="RestoreError--wrapper flex justifyContent--space-between alignItems--center" key={`${error.title}-${i}`}>
          <div className="flex-auto icon error-small u-marginRight--10 u-marginLeft--10"> </div>
          <p className="u-fontSize--normal u-fontWeight--normal u-lineHeight--normal">{error.message}</p>
        </div>
      ))
    );
  }

  renderWarnings = (warnings) => {
    return (
      warnings.map((warning, i) => (
        <div className="RestoreWarning--wrapper flex justifyContent--space-between alignItems--center" key={`${warning.title}-${i}`}>
          <div className="flex-auto icon exclamationMark--icon u-marginRight--10 u-marginLeft--10"> </div>
          <p className="u-fontSize--normal u-fontWeight--normal u-lineHeight--normal">{warning.message}</p>
        </div>
      ))
    );
  }

  renderFailedRestoreView = (detail) => {
    if (detail?.warnings.length > 0 && detail?.errors?.length === 0) {
      return this.renderWarningsRestoreView(detail?.warnings);
    } else {
      return (
        <div className="FailedRestore--wrapper">
          <div className="flex flex-column alignItems--center">
            <span className="icon u-superWarning--large"></span>
            <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-lineHeight--normal u-marginTop--15 u-marginBottom--10">
              Application failed to restore </p>
            <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal">
              Your application failed to restore to  <span className="u-fontWeight--bold u-color--dustyGray"> {this.props.match.params.id} </span> because of errors. During the restore there were
            <span className="u-fontWeight--bold  u-color--tundora"> {detail.warnings?.length} warnings </span> and <span className="u-fontWeight--bold u-color--tundora"> {detail.errors?.length} errors</span>.</p>
          </div>
          <div className="u-marginTop--30">
            {this.renderErrors(detail?.errors)}
            {this.renderWarnings(detail?.warnings)}
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
            Application restored with warnings </p>
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
    const { cancelingRestore } = this.state;
    const { restoreDetail } = this.props;

    const hasNoErrorsOrWarnings = restoreDetail?.restoreDetail?.warnings?.length === 0 && restoreDetail?.restoreDetail?.errors?.length === 0;
    const restoreCompleted = restoreDetail?.restoreDetail?.phase === "Completed";
    const restoreFailing = restoreDetail?.restoreDetail?.phase === "PartiallyFailed" || restoreDetail?.restoreDetail?.phase === "Failed";

    if (restoreDetail?.loading) {
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
          <title>{`${this.props.app.name} Snapshots Restore`}</title>
        </Helmet>
        {!restoreCompleted && !restoreFailing ?
          <div className="flex1 flex-column alignItems--center">
            <p className="u-fontWeight--bold u-color--tuna u-fontSize--larger u-lineHeight--normal u-marginBottom--10"> Application restore in progress </p>
            <p className="u-fontSize--normal u-fontWeight--medium u-color--dustyGray u-lineHeight--normal"> After all volumes have been restored you will need to log back in to the admin console. </p>
            <div className="flex flex-column  u-marginTop--40">
              {restoreDetail?.restoreDetail?.volumes?.length === 0 && restoreDetail?.restoreDetail?.warnings?.length === 0 && restoreDetail?.restoreDetail?.errors?.length === 0 &&
                <div className="flex-column flex1 alignItems--center justifyContent--center">
                  <Loader size="60" />
                </div>
              }
              {restoreDetail?.restoreDetail?.volumes?.map((volume, i) => {
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
                      {volume.timeRemainingSeconds !== 0 ?
                        <div className="flex justifyContent--center u-fontSize--smaller u-fontWeight--medium u-color--silverSand u-marginTop--5"> {volume.timeRemainingSeconds ? remainingTime : null}</div>
                        :
                        <div className="flex justifyContent--center u-fontSize--smaller u-fontWeight--medium u-color--silverSand u-marginTop--5"> Complete </div>
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
          !hasNoErrorsOrWarnings ?
            this.renderFailedRestoreView(restoreDetail?.restoreDetail) : null
        }
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  graphql(restoreDetail, {
    name: "restoreDetail",
    options: ({ app, match }) => {
      const appId = app.id;
      const restoreName = match.params.id;
      return {
        variables: { appId, restoreName },
        fetchPolicy: "no-cache"
      }
    }
  }),
  graphql(cancelRestore, {
    props: ({ mutate }) => ({
      cancelRestore: (appId) => mutate({ variables: { appId } })
    })
  })
)(AppSnapshotRestore);
