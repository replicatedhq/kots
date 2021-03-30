import dayjs from "dayjs";
import React, { Component } from "react";
import Helmet from "react-helmet";
import { withRouter } from "react-router-dom";
import size from "lodash/size";
import get from "lodash/get";
import Loader from "../shared/Loader";
import DashboardCard from "./DashboardCard";
import ConfigureGraphsModal from "../shared/modals/ConfigureGraphsModal";
import UpdateCheckerModal from "@src/components/modals/UpdateCheckerModal";
import SnapshotDifferencesModal from "@src/components/modals/SnapshotDifferencesModal";
import Modal from "react-modal";
import { Repeater } from "../../utilities/repeater";
import { Utilities } from "../../utilities/utilities";
import { AirgapUploader } from "../../utilities/airgapUploader";

import { XYPlot, XAxis, YAxis, HorizontalGridLines, VerticalGridLines, LineSeries, DiscreteColorLegend, Crosshair } from "react-vis";

import { getValueFormat } from "@grafana/ui"
import Handlebars from "handlebars";

import "../../scss/components/watches/Dashboard.scss";
import "../../../node_modules/react-vis/dist/style";

const COMMON_ERRORS = {
  "HTTP 401": "Registry credentials are invalid",
  "invalid username/password": "Registry credentials are invalid",
  "no such host": "No such host"
};

class Dashboard extends Component {

  state = {
    appName: "",
    iconUri: "",
    currentVersion: {},
    downstream: [],
    links: [],
    checkingForUpdates: false,
    checkingUpdateMessage: "Checking for updates",
    errorCheckingUpdate: false,
    appLicense: null,
    showConfigureGraphs: false,
    promValue: "",
    savingPromValue: false,
    savingPromError: "",
    activeChart: null,
    crosshairValues: [],
    updateChecker: new Repeater(),
    uploadingAirgapFile: false,
    airgapUploadError: null,
    viewAirgapUploadError: false,
    viewAirgapUpdateError: false,
    airgapUpdateError: "",
    startSnapshotErrorMsg: "",
    showUpdateCheckerModal: false,
    dashboard: {
      appStatus: null,
      metrics: [],
      prometheusAddress: "",
    },
    getAppDashboardJob: new Repeater(),
    gettingAppLicenseErrMsg: "",
    startSnapshotOptions: [
      { option: "partial", name: "Start a Partial snapshot" },
      { option: "full", name: "Start a Full snapshot" },
      { option: "learn", name: "Learn about the difference" }
    ],
    selectedSnapshotOption: { option: "full", name: "Start a Full snapshot" },
    snapshotDifferencesModal: false
  }

  toggleConfigureGraphs = () => {
    const { showConfigureGraphs } = this.state;
    this.setState({
      showConfigureGraphs: !showConfigureGraphs
    });
  }

  updatePromValue = () => {
    this.setState({ savingPromValue: true, savingPromError: "" });

    fetch(`${window.env.API_ENDPOINT}/prometheus`, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      body: JSON.stringify({
        value: this.state.promValue,
      }),
      method: "POST",
    })
      .then(async (res) => {
        if (!res.ok) {
          if (res.status === 401) {
            Utilities.logoutUser();
            return;
          }
          try {
            const response = await res.json();
            if (response?.error) {
              throw new Error(response?.error);
            }
          } catch (_) {
            // ignore
          }
          throw new Error(`Unexpected status code ${res.status}`);
        }
        await this.getAppDashboard();
        this.toggleConfigureGraphs();
        this.setState({ savingPromValue: false, savingPromError: "" });
      })
      .catch((err) => {
        console.log(err);
        this.setState({ savingPromValue: false, savingPromError: err?.message });
      });
  }

  onPromValueChange = (e) => {
    const { value } = e.target;
    this.setState({
      promValue: value
    });
  }

  setWatchState = (app) => {
    this.setState({
      appName: app.name,
      iconUri: app.iconUri,
      currentVersion: app.downstreams[0]?.currentVersion,
      downstream: app.downstreams[0],
      links: app.downstreams[0]?.links
    });
  }

  componentDidUpdate(lastProps) {
    const { app } = this.props;
    if (app !== lastProps.app && app) {
      this.setWatchState(app)
    }
  }

  getAppLicense = async (app) => {
    await fetch(`${window.env.API_ENDPOINT}/app/${app.slug}/license`, {
      method: "GET",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    }).then(async (res) => {
      if (!res.ok) {
        this.setState({ gettingAppLicenseErrMsg: body.error });
        return;
      }
      const body = await res.json();
      if (body === null) {
        this.setState({ appLicense: {}, gettingAppLicenseErrMsg: "" });
      } else {
        this.setState({ appLicense: body, gettingAppLicenseErrMsg: "" });
      }
    }).catch((err) => {
      console.log(err)
      this.setState({ gettingAppLicenseErrMsg: err ? `Error while getting the license: ${err.message}` : "Something went wrong, please try again." })
    });
  }

  componentDidMount() {
    const { app } = this.props;

    if (app?.isAirgap && !this.state.airgapUploader) {
      this.getAirgapConfig()
    }

    this.state.updateChecker.start(this.updateStatus, 1000);
    this.state.getAppDashboardJob.start(this.getAppDashboard, 2000);

    if (app) {
      this.setWatchState(app);
      this.getAppLicense(app);
    }
  }

  componentWillUnmount() {
    this.state.updateChecker.stop();
    this.state.getAppDashboardJob.stop();
  }

  getAirgapConfig = async () => {
    const { app } = this.props;
    const configUrl = `${window.env.API_ENDPOINT}/app/${app.slug}/airgap/config`;
    let simultaneousUploads = 3;
    try {
      let res = await fetch(configUrl, {
        method: "GET",
        headers: {
          "Content-Type": "application/json",
          "Authorization": Utilities.getToken(),
        }
      });
      if (res.ok) {
        const response = await res.json();
        simultaneousUploads = response.simultaneousUploads;
      }
    } catch {
      // no-op
    }

    this.setState({
      airgapUploader: new AirgapUploader(true, app.slug, this.onDropBundle, simultaneousUploads),
    });
}

  getAppDashboard = () => {
    return new Promise((resolve, reject) => {
      fetch(`${window.env.API_ENDPOINT}/app/${this.props.app?.slug}/cluster/${this.props.cluster?.id}/dashboard`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      })
        .then(async (res) => {
          if (!res.ok && res.status === 401) {
            Utilities.logoutUser();
            return;
          }
          const response = await res.json();
          this.setState({
            dashboard: {
              appStatus: response.appStatus,
              prometheusAddress: response.prometheusAddress,
              metrics: response.metrics,
            },
          });
          resolve();
        })
        .catch((err) => {
          console.log(err);
          reject(err);
        });
    });
  }

  onCheckForUpdates = async () => {
    const { app } = this.props;

    this.setState({
      checkingForUpdates: true,
      checkingForUpdateError: false,
    });

    fetch(`${window.env.API_ENDPOINT}/app/${app.slug}/updatecheck`, {
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      },
      method: "POST",
    })
      .then(async (res) => {
        this.state.updateChecker.start(this.updateStatus, 1000);
      })
      .catch((err) => {
        this.setState({ errorCheckingUpdate: true });
      });
  }

  hideUpdateCheckerModal = () => {
    this.setState({
      showUpdateCheckerModal: false
    });
  }

  showUpdateCheckerModal = () => {
    this.setState({
      showUpdateCheckerModal: true
    });
  }

  updateStatus = () => {
    const { app } = this.props;

    return new Promise((resolve, reject) => {
      fetch(`${window.env.API_ENDPOINT}/app/${app?.slug}/task/updatedownload`, {
        headers: {
          "Authorization": Utilities.getToken(),
          "Content-Type": "application/json",
        },
        method: "GET",
      })
        .then(async (res) => {
          const response = await res.json();

          if (response.status !== "running" && !this.props.isBundleUploading) {
            this.state.updateChecker.stop();

            this.setState({
              checkingForUpdates: false,
              checkingUpdateMessage: response.currentMessage,
              checkingForUpdateError: response.status === "failed"
            });

            if (this.props.updateCallback) {
              this.props.updateCallback();
            }
          } else {
            this.setState({
              checkingForUpdates: true,
              checkingUpdateMessage: response.currentMessage,
            });
          }
          resolve();
        }).catch((err) => {
          console.log("failed to get rewrite status", err);
          reject();
        });
    });
  }


  onDropBundle = async () => {
    this.setState({
      uploadingAirgapFile: true,
      checkingForUpdates: true,
      airgapUploadError: null,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
    });

    this.props.toggleIsBundleUploading(true);

    const params = {
      appId: this.props.app?.id,
    };
    this.state.airgapUploader.upload(params, this.onUploadProgress, this.onUploadError, this.onUploadComplete);
  }

  onUploadProgress = (progress, size, resuming = false) => {
    this.setState({
      uploadProgress: progress,
      uploadSize: size,
      uploadResuming: resuming,
    });
  }

  onUploadError = message => {
    this.setState({
      uploadingAirgapFile: false,
      checkingForUpdates: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
      airgapUploadError: message || "Error uploading bundle, please try again"
    });
    this.props.toggleIsBundleUploading(false);
  }

  onUploadComplete = () => {
    this.state.updateChecker.start(this.updateStatus, 1000);
    this.setState({
      uploadingAirgapFile: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
    });
    this.props.toggleIsBundleUploading(false);
  }

  onProgressError = async (airgapUploadError) => {
    Object.entries(COMMON_ERRORS).forEach(([errorString, message]) => {
      if (airgapUploadError.includes(errorString)) {
        airgapUploadError = message;
      }
    });
    this.setState({
      uploadingAirgapFile: false,
      airgapUploadError,
      checkingForUpdates: false,
      uploadProgress: 0,
      uploadSize: 0,
      uploadResuming: false,
    });
  }

  getLegendItems = (chart) => {
    return chart.series.map((series) => {
      const metrics = {};
      series.metric.forEach((metric) => {
        metrics[metric.name] = metric.value;
      });
      if (series.legendTemplate) {
        try {
          const template = Handlebars.compile(series.legendTemplate);
          return template(metrics);
        } catch (err) {
          console.error("Failed to compile legend template", err);
        }
      }
      return metrics.length > 0 ? metrics[Object.keys(metrics)[0]] : "";
    });
  }

  toggleViewAirgapUploadError = () => {
    this.setState({ viewAirgapUploadError: !this.state.viewAirgapUploadError });
  }

  toggleViewAirgapUpdateError = (err) => {
    this.setState({
      viewAirgapUpdateError: !this.state.viewAirgapUpdateError,
      airgapUpdateError: !this.state.viewAirgapUpdateError ? err : ""
    });
  }

  getValue = (chart, value) => {
    let yAxisTickFormat = null;
    if (chart.tickFormat) {
      const valueFormatter = getValueFormat(chart.tickFormat);
      yAxisTickFormat = (v) => `${valueFormatter(v)}`;
      return yAxisTickFormat(value);
    } else if (chart.tickTemplate) {
      try {
        const template = Handlebars.compile(chart.tickTemplate);
        yAxisTickFormat = (v) => `${template({ values: v })}`;
        return yAxisTickFormat(value);
      } catch (err) {
        console.error("Failed to compile y axis tick template", err);
      }
    } else {
      return value.toFixed(5);
    }
  }

  renderGraph = (chart) => {
    const axisStyle = {
      title: { fontSize: "12px", fontWeight: 500, fill: "#4A4A4A" },
      ticks: { fontSize: "12px", fontWeight: 400, fill: "#4A4A4A" }
    }
    const legendItems = this.getLegendItems(chart);
    const series = chart.series.map((series, idx) => {
      const data = series.data.map((valuePair) => {
        return { x: valuePair.timestamp, y: valuePair.value };
      });

      return (
        <LineSeries
          key={idx}
          data={data}
          onNearestX={(value, { index }) => this.setState({
            crosshairValues: chart.series.map(s => ({ x: s.data[index].timestamp, y: s.data[index].value, pod: s.metric[0].value })),
            activeChart: chart
          })}
        />
      );
    });

    let yAxisTickFormat = null;
    if (chart.tickFormat) {
      const valueFormatter = getValueFormat(chart.tickFormat);
      yAxisTickFormat = (v) => `${valueFormatter(v)}`;
    } else if (chart.tickTemplate) {
      try {
        const template = Handlebars.compile(chart.tickTemplate);
        yAxisTickFormat = (v) => `${template({ values: v })}`;
      } catch (err) {
        console.error("Failed to compile y axis tick template", err);
      }
    }

    return (
      <div className="dashboard-card graph flex-column flex1 flex u-marginTop--20" key={chart.title}>
        <XYPlot width={460} height={180} onMouseLeave={() => this.setState({ crosshairValues: [] })} margin={{ left: 60 }}>
          <VerticalGridLines />
          <HorizontalGridLines />
          <XAxis tickFormat={v => `${dayjs.unix(v).format("H:mm")}`} style={axisStyle} />
          <YAxis width={60} tickFormat={yAxisTickFormat} style={axisStyle} />
          {series}
          {this.state.crosshairValues?.length > 0 && this.state.activeChart === chart &&
            <Crosshair values={this.state.crosshairValues}>
              <div className="flex flex-column" style={{ background: "black", width: "250px" }}>
                <p className="u-fontWeight--bold u-textAlign--center"> {dayjs.unix(this.state.crosshairValues[0].x).format("LLL")} </p>
                <br />
                {this.state.crosshairValues.map((c, i) => {
                  return (
                    <div className="flex-auto flex flexWrap--wrap u-padding--5" key={i}>
                      <div className="flex flex1">
                        <p className="u-fontWeight--normal">{c.pod}:</p>
                      </div>
                      <div className="flex flex1">
                        <span className="u-fontWeight--bold u-marginLeft--10">{this.getValue(chart, c.y)}</span>
                      </div>
                    </div>
                  )
                })}
              </div>
            </Crosshair>
          }
        </XYPlot>
        {legendItems ? <DiscreteColorLegend className="legends" height={120} items={legendItems} /> : null}
        <div className="u-marginTop--10 u-paddingBottom--10 u-textAlign--center">
          <p className="u-fontSize--normal u-fontWeight--bold u-color--tundora u-lineHeight--normal">{chart.title}</p>
        </div>
      </div>
    );
  }

  startASnapshot = (option) => {
    const { app } = this.props;
    this.setState({
      startingSnapshot: true,
      startSnapshotErr: false,
      startSnapshotErrorMsg: "",
    });

    let url = option === "full" ?
      `${window.env.API_ENDPOINT}/snapshot/backup`
      : `${window.env.API_ENDPOINT}/app/${app.slug}/snapshot/backup`;

    fetch(url, {
      method: "POST",
      headers: {
        "Authorization": Utilities.getToken(),
        "Content-Type": "application/json",
      }
    })
      .then(async (result) => {
        if (!result.ok && result.status === 409) {
          const res = await result.json();
          if (res.kotsadmRequiresVeleroAccess) {
            this.setState({
              startingSnapshot: false
            });
            this.props.history.replace("/snapshots/settings");
            return;
          }
        }

        if (result.ok) {
          this.setState({
            startingSnapshot: false
          });
          this.props.ping();
          option === "full" ?
            this.props.history.push("/snapshots")
            : this.props.history.push(`/snapshots/partial/${app.slug}`)
        } else {
          const body = await result.json();
          this.setState({
            startingSnapshot: false,
            startSnapshotErr: true,
            startSnapshotErrorMsg: body.error,
          });
        }
      })
      .catch(err => {
        console.log(err);
        this.setState({
          startSnapshotErrorMsg: err ? err.message : "Something went wrong, please try again."
        });
      })
  }

  onSnapshotOptionChange = (selectedSnapshotOption) => {
    this.setState({ selectedSnapshotOption }, () => {
      if (selectedSnapshotOption.option === "learn") {
        this.setState({ snapshotDifferencesModal: true });
      } else {
        this.startASnapshot(selectedSnapshotOption.option);
      }
    });
  }

  toggleSnaphotDifferencesModal = () => {
    this.setState({ snapshotDifferencesModal: !this.state.snapshotDifferencesModal });
  }

  onSnapshotOptionClick = () => {
    const { selectedSnapshotOption } = this.state;
    this.startASnapshot(selectedSnapshotOption.option);
  }

  render() {
    const {
      appName,
      iconUri,
      currentVersion,
      downstream,
      links,
      checkingForUpdates,
      checkingUpdateMessage,
      errorCheckingUpdate,
      uploadingAirgapFile,
      airgapUploadError,
      appLicense,
      showConfigureGraphs,
      promValue,
      savingPromValue,
      savingPromError,
    } = this.state;

    const { app, isBundleUploading, isVeleroInstalled } = this.props;

    let checkingUpdateText = checkingUpdateMessage;
    try {
      const jsonMessage = JSON.parse(checkingUpdateText);
      const type = get(jsonMessage, "type");
      if (type === "progressReport") {
        checkingUpdateText = jsonMessage.compatibilityMessage;
        // TODO: handle image upload progress here
      }
    } catch {
      // empty
    }

    if (!app) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    return (
      <div className="flex-column flex1 u-position--relative u-overflow--auto u-padding--20">
        <Helmet>
          <title>{appName}</title>
        </Helmet>
        <div className="Dashboard flex flex-auto justifyContent--center alignSelf--center alignItems--center">
          <div className="flex1 flex-column">
            <div className="flex flex1">
              <div className="flex flex1 alignItems--center">
                <div className="flex flex-auto">
                  <div
                    style={{ backgroundImage: `url(${iconUri})` }}
                    className="Dashboard--appIcon u-position--relative">
                  </div>
                </div>
                <p className="u-fontSize--30 u-color--tuna u-fontWeight--bold u-marginLeft--20">{appName}</p>
              </div>
            </div>
            <div className="u-marginTop--30 u-paddingTop--10 flex-auto flex flexWrap--wrap u-width--full alignItems--center justifyContent--center">
              <DashboardCard
                cardName="Application"
                application={true}
                cardIcon="applicationIcon"
                appStatus={this.state.dashboard?.appStatus?.state}
                url={this.props.match.url}
                links={links}
                app={app}
              />
              <DashboardCard
                cardName={`Version ${currentVersion?.versionLabel ? currentVersion?.versionLabel : ""}`}
                cardIcon="versionIcon"
                versionHistory={true}
                currentVersion={currentVersion}
                downstream={downstream}
                app={app}
                url={this.props.match.url}
                checkingForUpdates={checkingForUpdates}
                checkingUpdateText={checkingUpdateText}
                errorCheckingUpdate={errorCheckingUpdate}
                onDropBundle={this.onDropBundle}
                airgapUploader={this.state.airgapUploader}
                uploadingAirgapFile={uploadingAirgapFile}
                airgapUploadError={airgapUploadError}
                uploadProgress={this.state.uploadProgress}
                uploadSize={this.state.uploadSize}
                uploadResuming={this.state.uploadResuming}
                onProgressError={this.onProgressError}
                onCheckForUpdates={() => this.onCheckForUpdates()}
                onUploadNewVersion={() => this.onUploadNewVersion()}
                isBundleUploading={isBundleUploading}
                checkingForUpdateError={this.state.checkingForUpdateError}
                viewAirgapUploadError={() => this.toggleViewAirgapUploadError()}
                viewAirgapUpdateError={(err) => this.toggleViewAirgapUpdateError(err)}
                showUpdateCheckerModal={this.showUpdateCheckerModal}
              />
              {app.allowSnapshots && isVeleroInstalled ?
                <div className="small-dashboard-wrapper flex-column flex">
                  <DashboardCard
                    cardName="Snapshots"
                    cardIcon="snapshotIcon"
                    url={this.props.match.url}
                    app={app}
                    isSnapshotAllowed={app.allowSnapshots && isVeleroInstalled}
                    isVeleroInstalled={isVeleroInstalled}
                    startASnapshot={this.startASnapshot}
                    startSnapshotOptions={this.state.startSnapshotOptions}
                    startSnapshotErr={this.state.startSnapshotErr}
                    startSnapshotErrorMsg={this.state.startSnapshotErrorMsg}
                    snapshotInProgressApps={this.props.snapshotInProgressApps}
                    selectedSnapshotOption={this.state.selectedSnapshotOption}
                    onSnapshotOptionChange={this.onSnapshotOptionChange}
                    onSnapshotOptionClick={this.onSnapshotOptionClick}
                  />
                  <DashboardCard
                    cardName="License"
                    cardIcon={size(appLicense) > 0 ? "licenseIcon" : "grayedLicenseIcon"}
                    license={true}
                    isSnapshotAllowed={app.allowSnapshots && isVeleroInstalled}
                    url={this.props.match.url}
                    appLicense={appLicense}
                    gettingAppLicenseErrMsg={this.state.gettingAppLicenseErrMsg}
                  />
                </div>
                :
                <DashboardCard
                  cardName="License"
                  cardIcon={size(appLicense) > 0 ? "licenseIcon" : "grayedLicenseIcon"}
                  license={true}
                  url={this.props.match.url}
                  appLicense={appLicense}
                  gettingAppLicenseErrMsg={this.state.gettingAppLicenseErrMsg}
                />
              }
            </div>
            <div className="u-marginTop--30 flex flex1">
              {this.state.dashboard?.prometheusAddress ?
                <div>
                  <div className="flex flex1 justifyContent--flexEnd">
                    <span className="card-link" onClick={this.toggleConfigureGraphs}> Configure Prometheus Address </span>
                  </div>
                  <div className="flex-auto flex flexWrap--wrap u-width--full">
                    {this.state.dashboard?.metrics.map(this.renderGraph)}
                  </div>
                </div>
                :
                <div className="flex-auto flex flexWrap--wrap u-width--full u-position--relative">
                  <div className="dashboard-card emptyGraph flex-column flex1 flex">
                    <div className="flex flex1 justifyContent--center alignItems--center alignSelf--center">
                      <span className="icon graphIcon"></span>
                    </div>
                  </div>
                  <div className="dashboard-card emptyGraph flex-column flex1 flex">
                    <div className="flex flex1 justifyContent--center alignItems--center alignSelf--center">
                      <span className="icon graphPieIcon"></span>
                    </div>
                  </div>
                  <div className="dashboard-card absolute-button  flex flex1 alignItems--center justifyContent--center alignSelf--center">
                    <button className="btn secondary blue" onClick={this.toggleConfigureGraphs}> Configure graphs </button>
                  </div>
                </div>
              }
            </div>
          </div>
        </div>
        <ConfigureGraphsModal
          showConfigureGraphs={showConfigureGraphs}
          toggleConfigureGraphs={this.toggleConfigureGraphs}
          updatePromValue={this.updatePromValue}
          promValue={promValue}
          savingPromValue={savingPromValue}
          savingPromError={savingPromError}
          onPromValueChange={this.onPromValueChange}
        />
        {this.state.viewAirgapUploadError &&
          <Modal
            isOpen={this.state.viewAirgapUploadError}
            onRequestClose={this.toggleViewAirgapUploadError}
            contentLabel="Error uploading airgap bundle"
            ariaHideApp={false}
            className="Modal"
          >
            <div className="Modal-body">
              <p className="u-fontSize--large u-fontWeight--bold u-color--tuna">Error uploading airgap buundle</p>
              <div className="ExpandedError--wrapper u-marginTop--10 u-marginBottom--10">
                <p className="u-fontSize--normal u-color--chestnut">{this.state.airgapUploadError}</p>
              </div>
              <button type="button" className="btn primary u-marginTop--15" onClick={this.toggleViewAirgapUploadError}>Ok, got it!</button>
            </div>
          </Modal>
        }
        {this.state.viewAirgapUpdateError &&
          <Modal
            isOpen={this.state.viewAirgapUpdateError}
            onRequestClose={this.toggleViewAirgapUpdateError}
            contentLabel="Error updating airgap version"
            ariaHideApp={false}
            className="Modal"
          >
            <div className="Modal-body">
              <p className="u-fontSize--large u-fontWeight--bold u-color--tuna">Error updating version</p>
              <div className="ExpandedError--wrapper u-marginTop--10 u-marginBottom--10">
                <p className="u-fontSize--normal u-color--chestnut">{this.state.airgapUpdateError}</p>
              </div>
              <button type="button" className="btn primary u-marginTop--15" onClick={this.toggleViewAirgapUpdateError}>Ok, got it!</button>
            </div>
          </Modal>
        }

        {this.state.showUpdateCheckerModal &&
          <UpdateCheckerModal
            isOpen={this.state.showUpdateCheckerModal}
            onRequestClose={this.hideUpdateCheckerModal}
            updateCheckerSpec={app.updateCheckerSpec}
            appSlug={app.slug}
            gitopsEnabled={downstream?.gitops?.enabled}
            onUpdateCheckerSpecSubmitted={() => {
              this.hideUpdateCheckerModal();
              this.props.refreshAppData();
            }}
          />
        }
        {this.state.snapshotDifferencesModal &&
          <SnapshotDifferencesModal
            snapshotDifferencesModal={this.state.snapshotDifferencesModal}
            toggleSnapshotDifferencesModal={this.toggleSnaphotDifferencesModal}
          />}
      </div>
    );
  }
}

export default withRouter(Dashboard);
