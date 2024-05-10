import { Component } from "react";
import { withRouter } from "@src/utilities/react-router-utilities";
import MonacoEditor from "@monaco-editor/react";
import Modal from "react-modal";
import filter from "lodash/filter";
import isEmpty from "lodash/isEmpty";
import ReactApexChart from "react-apexcharts";
import dayjs from "dayjs";
import minMax from "dayjs/plugin/minMax";
import duration from "dayjs/plugin/duration";
import Loader from "../shared/Loader";
import ShowAllModal from "../modals/ShowAllModal";
import ViewSnapshotLogsModal from "../modals/ViewSnapshotLogsModal";
import ErrorModal from "../modals/ErrorModal";
import { Utilities } from "../../utilities/utilities";

dayjs.extend(minMax);
dayjs.extend(duration);

let colorIndex = 0;
let mapColors = {};

class SnapshotDetails extends Component {
  state = {
    showScriptsOutput: false,
    scriptOutput: "",
    selectedTab: "stdout",
    showAllVolumes: false,
    selectedScriptTab: "Pre-snapshot scripts",
    showAllPreSnapshotScripts: false,
    showAllPostSnapshotScripts: false,
    selectedErrorsWarningTab: "Errors",
    showAllWarnings: false,
    showAllErrors: false,
    series: [],
    toggleViewLogsModal: false,
    snapshotLogs: "",
    loadingSnapshotLogs: false,
    snapshotLogsErr: false,
    snapshotLogsErrMsg: "",

    loading: true,
    snapshotDetails: {},
    errorMessage: "",
    errorTitle: "",

    options: {
      chart: {
        height: 140,
        type: "rangeBar",
        toolbar: {
          show: false,
        },
      },
      plotOptions: {
        bar: {
          horizontal: true,
          distributed: true,
          dataLabels: {
            hideOverflowingLabels: false,
          },
        },
      },
      xaxis: {
        type: "datetime",
        labels: {
          formatter: (value) => {
            return dayjs(value).format("h:mm:ss");
          },
        },
      },
      yaxis: {
        show: false,
      },
      grid: {
        xaxis: {
          lines: {
            show: true,
          },
        },
        yaxis: {
          lines: {
            show: false,
          },
        },
      },
      tooltip: {
        custom: function ({ series, seriesIndex, dataPointIndex, w }) {
          return (
            '<div class="arrow_box">' +
            '<p class="u-textColor--primary u-fontSize--normal u-fontWeight--medium">' +
            w.globals.labels[dataPointIndex] +
            "</p>" +
            '<span class="u-fontSize--small u-fontWeight--normal u-textColor--bodyCopy u-marginTop--10">' +
            w.globals.seriesZ[seriesIndex][dataPointIndex] +
            "</span>" +
            "<br />" +
            "<br />" +
            '<span class="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy u-marginTop--10">' +
            "Started at " +
            dayjs(
              w.globals.seriesRangeStart[seriesIndex][dataPointIndex]
            ).format("h:mm:ss") +
            "</span>" +
            "<br />" +
            '<span class="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy">' +
            "Finished at " +
            dayjs(w.globals.seriesRangeEnd[seriesIndex][dataPointIndex]).format(
              "h:mm:ss"
            ) +
            "</span>" +
            "</div>"
          );
        },
      },
    },
  };

  componentDidMount() {
    this.fetchSnapshotDetails();
  }

  componentDidUpdate(lastProps) {
    const { params } = this.props;
    if (params.id !== lastProps.params.id) {
      this.fetchSnapshotDetails();
    }
  }

  fetchSnapshotDetails = async () => {
    const { params } = this.props;
    const snapshotName = params.id;

    this.setState({
      errorMessage: "",
      errorTitle: "",
    });

    try {
      const res = await fetch(
        `${process.env.API_ENDPOINT}/snapshot/${snapshotName}`,
        {
          method: "GET",
          credentials: "include",
        }
      );
      if (!res.ok) {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }
        this.setState({
          loading: false,
          errorMessage: `Unexpected status code: ${res.status}`,
          errorTitle: "Failed to fetch snapshot details",
        });
        return;
      }
      const response = await res.json();

      const snapshotDetails = response.backupDetail;

      let series = [];
      if (!isEmpty(snapshotDetails?.volumes)) {
        if (snapshotDetails?.hooks && !isEmpty(snapshotDetails?.hooks)) {
          series = this.getSeriesData(
            [...snapshotDetails?.volumes, ...snapshotDetails?.hooks].sort(
              (a, b) => new Date(a.started) - new Date(b.started)
            )
          );
        } else {
          series = this.getSeriesData(
            (snapshotDetails?.volumes).sort(
              (a, b) => new Date(a.started) - new Date(b.started)
            )
          );
        }
      } else if (snapshotDetails?.hooks && !isEmpty(snapshotDetails?.hooks)) {
        series = this.getSeriesData(
          (snapshotDetails?.hooks).sort(
            (a, b) => new Date(a.started) - new Date(b.started)
          )
        );
      }

      this.setState({
        loading: false,
        snapshotDetails: snapshotDetails,
        series: series,
        errorMessage: "",
        errorTitle: "",
      });
    } catch (err) {
      console.log(err);
      this.setState({
        loading: false,
        errorMessage: err
          ? `${err.message}`
          : "Something went wrong, please try again.",
        errorTitle: "Failed to fetch snapshot details",
      });
    }
  };

  preSnapshotScripts = () => {
    return filter(this.state.snapshotDetails?.hooks, (hook) => {
      return hook.phase === "pre";
    });
  };

  postSnapshotScripts = () => {
    return filter(this.state.snapshotDetails?.hooks, (hook) => {
      return hook.phase === "post";
    });
  };

  toggleShowAllPreScripts = () => {
    this.setState({
      showAllPreSnapshotScripts: !this.state.showAllPreSnapshotScripts,
    });
  };

  toggleShowAllPostScripts = () => {
    this.setState({
      showAllPostSnapshotScripts: !this.state.showAllPostSnapshotScripts,
    });
  };

  toggleShowAllVolumes = () => {
    this.setState({ showAllVolumes: !this.state.showAllVolumes });
  };

  toggleScriptsOutput = (output) => {
    if (this.state.toggleScriptsOutput) {
      this.setState({ showScriptsOutput: false, scriptOutput: "" });
    } else {
      this.setState({ showScriptsOutput: true, scriptOutput: output });
    }
  };

  toggleShowAllWarnings = () => {
    this.setState({ showAllWarnings: !this.state.showAllWarnings });
  };

  toggleShowAllErrors = () => {
    this.setState({ showAllErrors: !this.state.showAllErrors });
  };

  viewLogs = () => {
    this.setState(
      {
        toggleViewLogsModal: !this.state.toggleViewLogsModal,
      },
      () => {
        this.setState({ loadingSnapshotLogs: true });
        const name = this.state.snapshotDetails?.name;
        const url = `${process.env.API_ENDPOINT}/snapshot/${name}/logs`;
        fetch(url, {
          credentials: "include",
          method: "GET",
        })
          .then(async (result) => {
            const logs = await result.text();
            if (!result.ok) {
              this.setState({
                loadingSnapshotLogs: false,
                snapshotLogsErr: true,
                snapshotLogsErrMsg:
                  "An error occurred while viewing snapshot logs. Please try again",
              });
            } else {
              this.setState({
                snapshotLogs: logs,
                snapshotLogsErr: false,
                snapshotLogsErrMsg: "",
                loadingSnapshotLogs: false,
              });
            }
          })
          .catch((err) => {
            this.setState({
              loadingSnapshotLogs: false,
              snapshotLogsErr: true,
              snapshotLogsErrMsg: err,
            });
          });
      }
    );
  };

  renderOutputTabs = () => {
    const { selectedTab } = this.state;
    const tabs = ["stdout", "stderr"];
    return (
      <div className="flex action-tab-bar u-marginTop--10">
        {tabs.map((tab) => (
          <div
            className={`tab-item ${tab === selectedTab && "is-active"}`}
            key={tab}
            onClick={() => this.setState({ selectedTab: tab })}
          >
            {tab}
          </div>
        ))}
      </div>
    );
  };

  renderShowAllVolumes = (volumes) => {
    return volumes.map((volume) => {
      const diffMinutes = dayjs(volume?.finishedAt).diff(
        dayjs(volume?.startedAt),
        "minutes"
      );
      return (
        <div
          className="flex flex1 u-borderBottom--gray alignItems--center"
          key={volume.name}
        >
          <div className="flex1 u-paddingBottom--15 u-paddingTop--15 u-paddingLeft--10">
            <p className="flex1 u-fontSize--large u-textColor--primary u-fontWeight--bold u-lineHeight--bold u-marginBottom--8">
              {volume.podNamespace}
              {"/"}
              {volume.podName} {volume.podVolumeName}
              <span className="u-fontWeight--normal u-textColor--bodyCopy">
                {" "}
                {volume.name}
              </span>
            </p>
            <p className="u-fontSize--normal u-textColor--accent u-fontWeight--bold u-lineHeight--normal u-marginRight--20">
              Size:
              <span className="u-fontWeight--normal u-textColor--bodyCopy">
                {" "}
                {volume.doneBytesHuman}/{volume.sizeBytesHuman}{" "}
              </span>
            </p>
          </div>
          <div className="flex flex-column justifyContent--flexEnd">
            <p className="u-fontSize--small u-fontWeight--normal alignSelf--flexEnd u-marginBottom--8">
              <span
                className={`status-indicator ${volume?.phase?.toLowerCase()} u-marginLeft--5`}
              >
                {volume.phase}
              </span>
            </p>
            <p className="u-fontSize--small u-fontWeight--normal">
              {" "}
              Finished in{" "}
              {diffMinutes === 0
                ? "less than a minute"
                : `${diffMinutes} minutes`}{" "}
            </p>
          </div>
        </div>
      );
    });
  };

  renderScriptsTabs = () => {
    const { selectedScriptTab } = this.state;
    const tabs = ["Pre-snapshot scripts", "Post-snapshot scripts"];
    return (
      <div className="flex action-tab-bar u-marginTop--10">
        {tabs.map((tab) => (
          <div
            className={`tab-item ${tab === selectedScriptTab && "is-active"}`}
            key={tab}
            onClick={() => this.setState({ selectedScriptTab: tab })}
          >
            {this.props.isEmbeddedCluster
              ? tab.replace("snapshot", "backup")
              : tab}
          </div>
        ))}
      </div>
    );
  };

  renderErrorsWarningsTabs = () => {
    const { snapshotDetails, selectedErrorsWarningTab } = this.state;
    const tabs = ["Errors", "Warnings"];
    return (
      <div className="flex action-tab-bar u-marginTop--10">
        {tabs.map((tab) => (
          <div
            className={`tab-item ${
              tab === selectedErrorsWarningTab && "is-active"
            }`}
            key={tab}
            onClick={() => this.setState({ selectedErrorsWarningTab: tab })}
          >
            {tab}
            {tab === "Errors" ? (
              <span className="errors u-marginLeft--5">
                {" "}
                {snapshotDetails?.errors?.length}{" "}
              </span>
            ) : (
              <span className="warnings u-marginLeft--5">
                {" "}
                {!snapshotDetails?.warnings
                  ? "0"
                  : snapshotDetails?.warnings?.length}{" "}
              </span>
            )}
          </div>
        ))}
      </div>
    );
  };

  renderShowAllScripts = (hooks) => {
    return hooks.map((hook, i) => {
      const diffMinutes = dayjs(hook?.finishedAt).diff(
        dayjs(hook?.startedAt),
        "minutes"
      );
      return (
        <div
          className="flex flex1 u-borderBottom--gray alignItems--center"
          key={`${hook.name}-${hook.phase}-${i}`}
        >
          <div className="flex flex1 u-paddingBottom--15 u-paddingTop--15 u-paddingLeft--10">
            <div className="flex flex-column">
              <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold u-lineHeight--bold u-marginBottom--8">
                {hook.name}{" "}
                <span className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginLeft--5">
                  Pod: {hook.podName}{" "}
                </span>{" "}
              </p>
              <span className="u-fontSize--small u-fontWeight--normal u-textColor--bodyCopy u-marginRight--10">
                {" "}
                {hook.command}{" "}
              </span>
            </div>
          </div>
          <div className="flex flex-column justifyContent--flexEnd">
            <p className="u-fontSize--small u-fontWeight--normal alignSelf--flexEnd u-marginBottom--8">
              <span
                className={`status-indicator ${
                  hook.errors ? "failed" : "completed"
                } u-marginLeft--5`}
              >
                {hook.errors ? "Failed" : "Completed"}
              </span>
            </p>
            {!hook.errors && (
              <p className="u-fontSize--small u-fontWeight--normal u-marginBottom--8">
                {" "}
                Finished in{" "}
                {diffMinutes === 0
                  ? "less than a minute"
                  : `${diffMinutes} minutes`}{" "}
              </p>
            )}
            {hook.stderr !== "" ||
              (hook.stdout !== "" && (
                <span
                  className="link u-fontSize--small alignSelf--flexEnd"
                  onClick={() => this.toggleScriptsOutput(hook)}
                >
                  {" "}
                  View output{" "}
                </span>
              ))}
          </div>
        </div>
      );
    });
  };

  renderShowAllWarnings = (warnings) => {
    return warnings.map((warning, i) => (
      <div
        className="flex flex1 u-borderBottom--gray"
        key={`${warning.title}-${i}`}
      >
        <div className="flex1">
          <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold u-lineHeight--bold u-paddingBottom--10 u-paddingTop--10 u-paddingLeft--10">
            {warning.title}
          </p>
        </div>
      </div>
    ));
  };

  renderShowAllErrors = (errors) => {
    return errors.map((error, i) => (
      <div
        className="flex flex1 u-borderBottom--gray"
        key={`${error.title}-${i}`}
      >
        <div className="flex1 u-paddingBottom--10 u-paddingTop--10 u-paddingLeft--10">
          <p className="u-fontSize--large u-textColor--error u-fontWeight--bold u-lineHeight--bold u-marginBottom--8">
            {error.title}
          </p>
          <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy">
            {" "}
            {error.message}{" "}
          </p>
        </div>
      </div>
    ));
  };

  calculateTimeInterval = (data) => {
    const startedTimes = data.map((d) => dayjs(d.startedAt));
    const finishedTimes = data.map((d) => dayjs(d.finishedAt));
    const minStarted = startedTimes?.length ? dayjs.min(startedTimes) : "";
    const maxFinished = finishedTimes?.length ? dayjs.max(finishedTimes) : "";

    const duration = dayjs.duration(maxFinished.diff(minStarted));
    const diffHours = parseInt(duration.asHours());
    const diffMinutes = parseInt(duration.asMinutes()) % 60;
    const diffDays = parseInt(duration.asDays());

    const timeObj = {
      minStarted: minStarted.format("MM/DD/YY @ hh:mm a z"),
      maxFinished: maxFinished.format("MM/DD/YY @ hh:mm a z"),
      maxHourDifference: diffHours,
      maxMinDifference: diffMinutes,
      maxDaysDifference: diffDays,
    };

    return timeObj;
  };

  assignColorToPath = (podName) => {
    const colors = [
      "#32C5FF",
      "#44BB66",
      "#6236FF",
      "#F7B500",
      "#4999AD",
      "#ED2D2D",
      "#6236FF",
      "#48C9B0",
      "#A569BD",
      "#D35400",
    ];

    if (mapColors[podName]) {
      return mapColors[podName];
    } else {
      mapColors[podName] = colors[colorIndex];
      colorIndex = (colorIndex + 1) % colors.length;
      return mapColors[podName];
    }
  };

  getSeriesData = (seriesData) => {
    const series = [{ data: null }];
    if (!seriesData) {
      return series;
    }

    const data = seriesData.map((d, i) => {
      let finishedTime;
      if (d.startedAt === d.finishedAt) {
        finishedTime = new Date(
          dayjs(d.finishedAt).add(1, "seconds")
        ).getTime();
      } else {
        finishedTime = new Date(d.finishedAt).getTime();
      }

      return {
        x: d.containerName ? `${d.name} (${d.podName})-${i}` : `${d.name}`,
        y: [new Date(d.startedAt).getTime(), finishedTime],
        z: d.containerName ? `${d.phase}-snapshot-script` : "Volume",
        fillColor: d.containerName
          ? this.assignColorToPath(d.name)
          : this.assignColorToPath(d.name),
      };
    });
    series[0].data = data;
    return series;
  };

  renderTimeInterval = () => {
    let data;
    if (!isEmpty(this.state.snapshotDetails?.volumes)) {
      if (!isEmpty(this.state.snapshotDetails?.hooks)) {
        data = [
          ...this.state.snapshotDetails?.volumes,
          ...this.state.snapshotDetails?.hooks,
        ];
      } else {
        data = this.state.snapshotDetails?.volumes;
      }
    } else if (!isEmpty(this.state.snapshotDetails?.hooks)) {
      data = this.state.snapshotDetails?.hooks;
    }
    return (
      <div className="flex flex1">
        <div className="flex flex1">
          <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy">
            Started:{" "}
            <span className="u-fontWeight--bold u-textColor--accent">
              {" "}
              {this.calculateTimeInterval(data).minStarted}
            </span>
          </p>
        </div>
        <div className="flex flex1 justifyContent--center">
          {this.calculateTimeInterval(data).maxHourDifference === 0 &&
          this.calculateTimeInterval(data).maxMinDifference === 0 ? (
            <p className="u-fontSize--small u-fontWeight--normal u-textColor--bodyCopy">
              Total capture time:{" "}
              <span className="u-fontWeight--bold u-textColor--accent">
                less than a minute
              </span>
            </p>
          ) : this.calculateTimeInterval(data).maxDaysDifference > 0 ? (
            <p className="u-fontSize--small u-fontWeight--normal u-textColor--bodyCopy">
              Total capture time:{" "}
              <span className="u-fontWeight--bold u-textColor--accent">{`${
                this.calculateTimeInterval(data).maxDaysDifference
              } days `}</span>
              <span className="u-fontWeight--bold u-textColor--accent">{`${
                this.calculateTimeInterval(data).maxHourDifference
              } hr `}</span>
              <span className="u-fontWeight--bold u-textColor--accent">{`${
                this.calculateTimeInterval(data).maxMinDifference
              } min `}</span>
            </p>
          ) : (
            <p className="u-fontSize--small u-fontWeight--normal u-textColor--bodyCopy">
              Total capture time:{" "}
              <span className="u-fontWeight--bold u-textColor--accent">{`${
                this.calculateTimeInterval(data).maxHourDifference
              } hr `}</span>
              <span className="u-fontWeight--bold u-textColor--accent">{`${
                this.calculateTimeInterval(data).maxMinDifference
              } min `}</span>
            </p>
          )}
        </div>
        <div className="flex flex1 justifyContent--flexEnd">
          <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy">
            Finished:{" "}
            <span className="u-fontWeight--bold u-textColor--accent">
              {" "}
              {this.calculateTimeInterval(data).maxFinished}{" "}
            </span>
          </p>
        </div>
      </div>
    );
  };

  render() {
    const { isEmbeddedCluster } = this.props;

    const {
      loading,
      showScriptsOutput,
      selectedTab,
      selectedScriptTab,
      scriptOutput,
      showAllVolumes,
      showAllPreSnapshotScripts,
      showAllPostSnapshotScripts,
      selectedErrorsWarningTab,
      showAllErrors,
      showAllWarnings,
      snapshotDetails,
      series,
      errorMessage,
      errorTitle,
    } = this.state;

    let featureName = "snapshot";
    if (isEmbeddedCluster) {
      featureName = "backup";
    }

    if (loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>
      );
    }

    return (
      <div className="container flex-column u-overflow--auto u-paddingTop--30 u-paddingBottom--20">
        <p className="u-marginBottom--30 u-fontSize--small u-textColor--accent u-fontWeight--medium">
          <span className="link" onClick={() => this.props.navigate(-1)}>
            {Utilities.toTitleCase(featureName)}s
          </span>
          <span className="u-textColor--bodyCopy"> &gt; </span>
          {snapshotDetails?.name}
        </p>
        <div className="card-bg">
          <div className="flex justifyContent--spaceBetween alignItems--center u-paddingBottom--15">
            <div className="flex-column u-lineHeight--normal">
              <p className="u-fontSize--larger u-fontWeight--bold u-textColor--primary u-marginBottom--5">
                {snapshotDetails?.name}
              </p>
              <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy">
                Total size:{" "}
                <span className="u-fontWeight--bold u-textColor--accent">
                  {snapshotDetails?.volumeSizeHuman}
                </span>
              </p>
            </div>
            <div className="flex-column u-lineHeight--normal u-textAlign--right">
              <p className="u-fontSize--normal u-fontWeight--normal u-marginBottom--5">
                Status:{" "}
                <span
                  className={`status-indicator ${snapshotDetails?.status?.toLowerCase()} u-marginLeft--5`}
                >
                  {Utilities.snapshotStatusToDisplayName(
                    snapshotDetails?.status
                  )}
                </span>
              </p>
              <div className="u-fontSize--small">
                {snapshotDetails?.status !== "InProgress" && (
                  <span className="link" onClick={() => this.viewLogs()}>
                    View logs
                  </span>
                )}
              </div>
            </div>
          </div>

          {snapshotDetails?.status === "InProgress" ? (
            <div className="flex flex-column alignItems--center u-marginTop--60">
              <span className="icon blueWarningIcon" />
              <p className="u-fontSize--larger u-fontWeight--bold u-textColor--primary u-marginTop--20">
                {" "}
                This {featureName} has not completed yet, check back soon{" "}
              </p>
            </div>
          ) : (
            <div>
              {!isEmpty(snapshotDetails?.volumes) ||
              !isEmpty(this.preSnapshotScripts()) ||
              !isEmpty(this.postSnapshotScripts()) ? (
                <div className="flex-column flex-auto card-item u-padding--15 u-marginBottom--30">
                  <p className="u-fontSize--larger u-fontWeight--bold u-textColor--primary u-marginBottom--10">
                    {Utilities.toTitleCase(featureName)} timeline
                  </p>
                  <div className="flex1" id="chart">
                    <ReactApexChart
                      options={this.state.options}
                      series={series}
                      type="rangeBar"
                      height={140}
                    />
                    {this.renderTimeInterval()}
                  </div>
                </div>
              ) : null}

              <div className="flex flex-auto u-marginBottom--30">
                <div className="flex flex1" style={{ gap: "15px" }}>
                  <div className="card-item u-padding--15 flex1">
                    <div className="flex flex1 alignItems--center u-paddingBottom--10 u-borderBottom--gray">
                      <p className="u-fontSize--larger u-textColor--primary u-fontWeight--bold u-lineHeight--bold">
                        Volumes
                      </p>
                      {snapshotDetails?.volumes?.length > 3 ? (
                        <div className="flex flex1 justifyContent--flexEnd">
                          <span
                            className="link u-fontSize--small"
                            onClick={() => this.toggleShowAllVolumes()}
                          >
                            Show all {snapshotDetails?.volumes?.length} volumes
                          </span>
                        </div>
                      ) : null}
                    </div>
                    {!isEmpty(snapshotDetails?.volumes) ? (
                      this.renderShowAllVolumes(
                        snapshotDetails?.volumes?.slice(0, 3)
                      )
                    ) : (
                      <div className="flex flex1 u-paddingTop--20 alignItems--center justifyContent--center">
                        <p className="u-fontSize--large u-fontWeight--normal u-textColor--bodyCopy">
                          {" "}
                          No volumes to display{" "}
                        </p>
                      </div>
                    )}
                  </div>
                  <div className="card-item u-padding--15 flex1">
                    <div className="flex flex-column u-paddingBottom--10 u-borderBottom--gray">
                      <div className="flex flex1">
                        <p className="u-fontSize--larger u-textColor--primary u-fontWeight--bold u-lineHeight--bold u-paddingBottom--10 flex flex1">
                          Scripts
                        </p>
                        {this.preSnapshotScripts()?.length > 3 &&
                        selectedScriptTab === "Pre-snapshot scripts" ? (
                          <div className="flex flex1 justifyContent--flexEnd">
                            <span
                              className="link u-fontSize--small"
                              onClick={() => this.toggleShowAllPreScripts()}
                            >
                              Show all {this.preSnapshotScripts()?.length}{" "}
                              pre-scripts
                            </span>
                          </div>
                        ) : null}
                        {this.postSnapshotScripts()?.length > 3 &&
                        selectedScriptTab === "Post-snapshot scripts" ? (
                          <div className="flex flex1 justifyContent--flexEnd">
                            <span
                              className="link u-fontSize--small"
                              onClick={() => this.toggleShowAllPostScripts()}
                            >
                              Show all {this.postSnapshotScripts()?.length}{" "}
                              post-scripts
                            </span>
                          </div>
                        ) : null}
                      </div>
                      <div className="flex-column flex1">
                        {this.renderScriptsTabs()}
                      </div>
                    </div>
                    <div>
                      {selectedScriptTab === "Pre-snapshot scripts" ? (
                        !isEmpty(this.preSnapshotScripts()) ? (
                          this.renderShowAllScripts(
                            this.preSnapshotScripts().slice(0, 3)
                          )
                        ) : (
                          <div className="flex flex1 u-paddingTop--20 alignItems--center justifyContent--center">
                            <p className="u-fontSize--large u-fontWeight--normal u-textColor--bodyCopy">
                              {" "}
                              No pre-{featureName} scripts to display{" "}
                            </p>
                          </div>
                        )
                      ) : selectedScriptTab === "Post-snapshot scripts" &&
                        !isEmpty(this.postSnapshotScripts()) ? (
                        this.renderShowAllScripts(
                          this.postSnapshotScripts().slice(0, 3)
                        )
                      ) : (
                        <div className="flex flex1 u-paddingTop--20 alignItems--center justifyContent--center">
                          <p className="u-fontSize--large u-fontWeight--normal u-textColor--bodyCopy">
                            {" "}
                            No post-{featureName} scripts to display{" "}
                          </p>
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              </div>

              {(!isEmpty(snapshotDetails?.errors) ||
                !isEmpty(snapshotDetails?.warnings)) && (
                <div className="flex flex-auto u-marginBottom--30">
                  <div className="flex-column flex1">
                    <div className="card-item u-padding--15 flex1">
                      <div className="flex flex-column u-paddingBottom--10 u-borderBottom--gray">
                        <div className="flex flex1">
                          <p className="u-fontSize--larger u-textColor--primary u-fontWeight--bold u-lineHeight--bold u-paddingBottom--10 flex flex1">
                            Errors and warnings
                          </p>
                          {snapshotDetails?.errors?.length > 3 &&
                          selectedErrorsWarningTab === "Errors" ? (
                            <div className="flex flex1 justifyContent--flexEnd">
                              <span
                                className="link u-fontSize--small"
                                onClick={() => this.toggleShowAllErrors()}
                              >
                                Show all {snapshotDetails?.errors?.length}{" "}
                                errors{" "}
                              </span>
                            </div>
                          ) : null}
                          {snapshotDetails?.warnings?.length > 3 &&
                          selectedErrorsWarningTab === "Warnings" ? (
                            <div className="flex flex1 justifyContent--flexEnd">
                              <span
                                className="link u-fontSize--small"
                                onClick={() => this.toggleShowAllWarnings()}
                              >
                                Show all {snapshotDetails?.warnings?.length}{" "}
                                warnings{" "}
                              </span>
                            </div>
                          ) : null}
                        </div>
                        <div className="flex-column flex1">
                          {this.renderErrorsWarningsTabs()}
                        </div>
                      </div>
                      <div>
                        {selectedErrorsWarningTab === "Errors" ? (
                          !isEmpty(snapshotDetails?.errors) ? (
                            this.renderShowAllErrors(
                              snapshotDetails?.errors.slice(0, 3)
                            )
                          ) : (
                            <div className="flex flex1 u-paddingTop--20 alignItems--center justifyContent--center">
                              <p className="u-fontSize--large u-fontWeight--normal u-textColor--bodyCopy">
                                {" "}
                                No errors to display{" "}
                              </p>
                            </div>
                          )
                        ) : selectedErrorsWarningTab === "Warnings" &&
                          !isEmpty(snapshotDetails?.warnings) ? (
                          this.renderShowAllWarnings(
                            snapshotDetails?.warnings?.slice(0, 3)
                          )
                        ) : (
                          <div className="flex flex1 u-paddingTop--20 alignItems--center justifyContent--center">
                            <p className="u-fontSize--large u-fontWeight--normal u-textColor--bodyCopy">
                              {" "}
                              No warnings to display{" "}
                            </p>
                          </div>
                        )}
                      </div>
                    </div>
                  </div>
                </div>
              )}
            </div>
          )}
        </div>
        {showScriptsOutput && scriptOutput && (
          <Modal
            isOpen={showScriptsOutput}
            onRequestClose={() => this.toggleScriptsOutput()}
            shouldReturnFocusAfterClose={false}
            contentLabel="View logs"
            ariaHideApp={false}
            className="Modal logs-modal"
          >
            <div className="Modal-body flex flex1 flex-column">
              {!selectedTab ? (
                <div className="flex-column flex1 alignItems--center justifyContent--center">
                  <Loader size="60" />
                </div>
              ) : (
                <div className="flex-column flex1">
                  {this.renderOutputTabs()}
                  <div className="flex-column flex1 u-border--gray monaco-editor-wrapper">
                    <MonacoEditor
                      language="json"
                      value={scriptOutput[selectedTab]}
                      options={{
                        readOnly: true,
                        contextmenu: false,
                        minimap: {
                          enabled: false,
                        },
                        scrollBeyondLastLine: false,
                      }}
                    />
                  </div>
                </div>
              )}
              <div className="u-marginTop--20 flex">
                <button
                  type="button"
                  className="btn primary blue"
                  onClick={() => this.toggleScriptsOutput()}
                >
                  Ok, got it!
                </button>
              </div>
            </div>
          </Modal>
        )}
        {showAllVolumes && (
          <ShowAllModal
            displayShowAllModal={showAllVolumes}
            toggleShowAllModal={this.toggleShowAllVolumes}
            dataToShow={this.renderShowAllVolumes(snapshotDetails?.volumes)}
            name="Volumes"
          />
        )}
        {showAllPreSnapshotScripts && (
          <ShowAllModal
            displayShowAllModal={showAllPreSnapshotScripts}
            toggleShowAllModal={this.toggleShowAllPreScripts}
            dataToShow={this.renderShowAllScripts(this.preSnapshotScripts())}
            name={`Pre-${featureName} scripts`}
          />
        )}
        {showAllPostSnapshotScripts && (
          <ShowAllModal
            displayShowAllModal={showAllPostSnapshotScripts}
            toggleShowAllModal={this.toggleShowAllPostScripts}
            dataToShow={this.renderShowAllScripts(this.postSnapshotScripts())}
            name={`Post-${featureName} scripts`}
          />
        )}
        {showAllWarnings && (
          <ShowAllModal
            displayShowAllModal={showAllWarnings}
            toggleShowAllModal={this.toggleShowAllWarnings}
            dataToShow={this.renderShowAllWarnings(snapshotDetails?.warnings)}
            name="Warnings"
          />
        )}
        {showAllErrors && (
          <ShowAllModal
            displayShowAllModal={showAllErrors}
            toggleShowAllModal={this.toggleShowAllErrors}
            dataToShow={this.renderShowAllErrors(snapshotDetails?.errors)}
            name="Errors"
          />
        )}
        {this.state.toggleViewLogsModal && (
          <ViewSnapshotLogsModal
            displayShowSnapshotLogsModal={this.state.toggleViewLogsModal}
            toggleViewLogsModal={this.viewLogs}
            logs={this.state.snapshotLogs}
            snapshotDetails={snapshotDetails}
            loadingSnapshotLogs={this.state.loadingSnapshotLogs}
            snapshotLogsErr={this.state.snapshotLogsErr}
            snapshotLogsErrMsg={this.state.snapshotLogsErrMsg}
          />
        )}

        {errorMessage && (
          <ErrorModal
            errorModal={this.state.displayErrorModal}
            toggleErrorModal={this.toggleErrorModal}
            errMsg={errorMessage}
            tryAgain={() => this.fetchSnapshotDetails()}
            err={errorTitle}
            loading={this.state.loadingApp}
          />
        )}
      </div>
    );
  }
}

export default withRouter(SnapshotDetails);
