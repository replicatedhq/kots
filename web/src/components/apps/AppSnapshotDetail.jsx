import React, { Component } from "react";
import { graphql, compose, withApollo } from "react-apollo";
import { Link, withRouter } from "react-router-dom";
import MonacoEditor from "react-monaco-editor";
import Modal from "react-modal";
import filter from "lodash/filter";
import ReactApexChart from "react-apexcharts";
import moment from "moment";
import Loader from "../shared/Loader";
import { snapshotDetail } from "../../queries/SnapshotQueries";
import ShowAllModal from "../modals/ShowAllModal";
import { Utilities } from "../../utilities/utilities";

class AppSnapshotDetail extends Component {
  state = {
    showOutputForPreScripts: false,
    preScriptOutput: "",
    selectedTab: "stdout",
    showAllVolumes: false,
    showAllPreSnapshotScripts: false,
    showAllPostSnapshotScripts: false,
    showAllWarnings: false,
    showAllErrors: false,

    options: {
      chart: {
        height: 110,
        type: "rangeBar",
        toolbar: {
          show: false
        }
      },
      plotOptions: {
        bar: {
          horizontal: true,
          distributed: true,
          dataLabels: {
            hideOverflowingLabels: false
          }
        }
      },
      xaxis: {
        type: "datetime",
        labels: {
          formatter: (value) => {
            return moment(value).format("h:mm:ss");
          }
        }
      },
      yaxis: {
        show: false
      },
      grid: {
        xaxis: {
          lines: {
            show: true
          }
        },
        yaxis: {
          lines: {
            show: false
          }
        },
      },
      tooltip: {
        custom: function ({ series, seriesIndex, dataPointIndex, w }) {
          return (
            '<div class="arrow_box">' +
              '<p class="u-color--tuna u-fontSize--normal u-fontWeight--medium">' +
                w.globals.labels[dataPointIndex] +
              "</p>" +
              "<br />" +
              '<span class="u-fontSize--normal u-fontWeight--normal u-color--dustyGray u-marginTop--10">' +
              "Started at " + moment(w.globals.seriesRangeStart[seriesIndex][dataPointIndex]).format("h:mm:ss") + "</span>" +
              "<br />" +
              '<span class="u-fontSize--normal u-fontWeight--normal u-color--dustyGray">' +
              "Finished at " + moment(w.globals.seriesRangeEnd[seriesIndex][dataPointIndex]).format("h:mm:ss") + "</span>" +
            "</div>"
          );
        }
      }
    }
  };

  preSnapshotScripts = () => {
    return filter(this.props.snapshotDetail?.snapshotDetail?.hooks, (hook) => {
      return hook.phase === "pre";
    });
  }

  postSnapshotScripts = () => {
    return filter(this.props.snapshotDetail?.snapshotDetail?.hooks, (hook) => {
      return hook.phase === "post";
    });
  }

  toggleShowAllPreScripts = () => {
    this.setState({ showAllPreSnapshotScripts: !this.state.showAllPreSnapshotScripts });
  }

  toggleShowAllPostScripts = () => {
    this.setState({ showAllPostSnapshotScripts: !this.state.showAllPostSnapshotScripts });
  }

  toggleShowAllVolumes = () => {
    this.setState({ showAllVolumes: !this.state.showAllVolumes });
  }

  toggleOutputForPreScripts = output => {
    if (this.state.toggleOutputForPreScripts) {
      this.setState({ showOutputForPreScripts: false, preScriptOutput: "" });
    } else {
      this.setState({ showOutputForPreScripts: true, preScriptOutput: output });
    }
  }

  toggleShowAllNamespaces = () => {
    this.setState({ showAllNamespaces: !this.state.showAllNamespaces });
  }

  toggleShowAllWarnings = () => {
    this.setState({ showAllWarnings: !this.state.showAllWarnings });
  }

  toggleShowAllErrors = () => {
    this.setState({ showAllErrors: !this.state.showAllErrors });
  }

  downloadLogs = () => {
    const name = this.props.snapshotDetail?.snapshotDetail?.name;
    const link = `${window.env.API_ENDPOINT}/snapshot/${name}/logs`;  

    fetch(link, {
      method: "GET",
      headers: {
        "Authorization": `${Utilities.getToken()}`,
      },
    });
  }

  renderOutputTabs = () => {
    const { selectedTab } = this.state;
    const tabs = ["stdout", "stderr"];
    return (
      <div className="flex action-tab-bar u-marginTop--10">
        {tabs.map(tab => (
          <div className={`tab-item blue ${tab === selectedTab && "is-active"}`} key={tab} onClick={() => this.setState({ selectedTab: tab })}>
            {tab}
          </div>
        ))}
      </div>
    );
  }

  renderShowAllVolumes = (volumes) => {
    return (
      volumes.map((volume) => (
        <div className="flex flex1 u-borderBottom--gray" key={volume.name}>
          <div className="flex1 u-paddingBottom--15 u-paddingTop--15 u-paddingLeft--10">
            <p className="flex1 u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--bold">{volume.name}</p>
            <p className="u-fontSize--normal u-color--doveGray u-fontWeight--bold u-lineHeight--normal u-marginRight--20">Size:
            <span className="u-fontWeight--normal u-color--dustyGray"> {volume.doneBytesHuman}/{volume.sizeBytesHuman} </span>
            </p>
          </div>
          <div className="flex flex1 justifyContent--flexEnd alignItems--center">
            <p className="u-fontSize--normal u-fontWeight--normal u-marginBottom--5"><span className={`status-indicator ${volume?.phase?.toLowerCase()} u-marginLeft--5`}>{volume.phase}</span></p>
          </div>
        </div>
      ))
    );
  }

  renderShowAllPrescripts = () => {
    return (
      this.preSnapshotScripts().map((hook, i) => (
        <div className="flex flex1 u-borderBottom--gray" key={`${hook.hookName}-${hook.phase}-${i}`}>
          <div className="flex1 u-paddingBottom--15 u-paddingTop--15 u-paddingLeft--10">
            <p className="flex1 u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--bold">{hook.hookName}</p>
            <div className="flex flex1 u-marginTop--5 alignItems--center">
              <span className="u-fontWeight--normal u-color--dustyGray u-marginRight--10"> {hook.command} </span>
              <span className="replicated-link u-fontSize--small" onClick={() => this.toggleOutputForPreScripts(hook)}> View output </span>
            </div>
          </div>
        </div>
      ))
    );
  }

  renderShowAllPostscripts = () => {
    return (
      this.postSnapshotScripts().map((hook, i) => (
        <div className="flex flex1 u-borderBottom--gray" key={`${hook.hookName}-${hook.phase}-${i}`}>
          <div className="flex1 u-paddingBottom--15 u-paddingTop--15 u-paddingLeft--10">
            <p className="flex1 u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--bold">{hook.hookName}</p>
            <div className="flex flex1 u-marginTop--5 alignItems--center">
              <span className="u-fontWeight--normal u-color--dustyGray u-marginRight--10"> {hook.command} </span>
              <span className="replicated-link u-fontSize--small" onClick={() => this.toggleOutputForPScripts(hook)}> View output </span>
            </div>
          </div>
        </div>
      ))
    );
  }

  renderShowAllNamespaces = (namespaces) => {
    return (
      namespaces.map((namespace) => (
        <div className="flex flex1 u-borderBottom--gray" key={namespace}>
          <div className="flex1">
            <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--bold u-paddingBottom--10 u-paddingTop--10 u-paddingLeft--10">{namespace}</p>
          </div>
        </div>
      ))
    );
  }

  renderShowAllWarnings = (warnings) => {
    return (
      warnings.map((warning, i) => (
        <div className="flex flex1 u-borderBottom--gray" key={`${warning.title}-${i}`}>
          <div className="flex1">
            <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--bold u-paddingBottom--10 u-paddingTop--10 u-paddingLeft--10">{warning.title}</p>
          </div>
        </div>
      ))
    );
  }

  renderShowAllErrors = (errors) => {
    return (
      errors.map((error, i) => (
        <div className="flex flex1 u-borderBottom--gray" key={`${error.title}-${i}`}>
          <div className="flex1 u-paddingBottom--10 u-paddingTop--10 u-paddingLeft--10">
            <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--bold">{error.title}</p>
            <p className="u-fontSize--normal u-fontWeight--normal u-color--dustyGray"> {error.message} </p>
          </div>
        </div>
      ))
    );
  }

  calculateVolumeTimeInterval = (volumes) => {
    const startedTimes = volumes.map((volume) => moment(volume.started).format("MMM D, YYYY h:mm A"));
    const finishedTimes = volumes.map((volume) => moment(volume.finished).format("MMM D, YYYY h:mm A"));
    const minStarted = startedTimes.length ? startedTimes.reduce((a, b) => { return a <= b ? a : b; }) : "";
    const maxFinished = finishedTimes.length ? finishedTimes.reduce((a, b) => { return a <= b ? b : a; }) : "";

    const diffHours = moment(maxFinished).diff(moment(minStarted), "hours")
    const diffMinutes = moment(maxFinished).diff(moment(minStarted), "minutes");

    return {
      "minStarted": minStarted,
      "maxFinished": maxFinished,
      "maxHourDifference": diffHours,
      "maxMinDifference": diffMinutes
    }
  }

  getSeriesData = (volumes) => {
    const colors = ["#32C5FF", "#44BB66", "#6236FF", "#F7B500", "#4999AD"];
    const series = [{ data: null }]
    if (!volumes) {
      return series;
    }
    const data = volumes.map((volume, i) => {
      return {
        x: volume.name,
        y: [new Date(volume.started).getTime(), new Date(volume.finished).getTime()],
        fillColor: colors[i]
      }
    });
    series[0].data = data;
    return series;
  }

  render() {
    const {
      showOutputForPreScripts,
      selectedTab,
      preScriptOutput,
      showAllVolumes,
      showAllPreSnapshotScripts,
      showAllPostSnapshotScripts,
      showAllNamespaces,
      showAllErrors,
      showAllWarnings } = this.state;
    const { app, snapshotDetail } = this.props;

    if (snapshotDetail?.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
        </div>)
    }

    const series = this.getSeriesData(this.props.snapshotDetail?.snapshotDetail?.volumes);

    return (
      <div className="container flex-column flex1 u-overflow--auto u-paddingTop--30 u-paddingBottom--20">
        <p className="u-marginBottom--30 u-fontSize--small u-color--tundora u-fontWeight--medium">
          <Link to={`/app/${app?.slug}/snapshots`} className="replicated-link">Snapshots</Link>
          <span className="u-color--dustyGray"> > </span>
          {snapshotDetail?.snapshotDetail?.name}
        </p>
        <div className="flex justifyContent--spaceBetween alignItems--center u-paddingBottom--30 u-borderBottom--gray">
          <div className="flex-column u-lineHeight--normal">
            <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna u-marginBottom--5">{snapshotDetail?.snapshotDetail?.name}</p>
            <p className="u-fontSize--normal u-fontWeight--normal u-color--dustyGray">Total size: <span className="u-fontWeight--bold u-color--doveGray">{snapshotDetail?.snapshotDetail?.volumeSizeHuman}</span></p>
          </div>
          <div className="flex-column u-lineHeight--normal u-textAlign--right">
            <p className="u-fontSize--normal u-fontWeight--normal u-marginBottom--5">Status: <span className={`status-indicator ${snapshotDetail?.snapshotDetail?.status.toLowerCase()} u-marginLeft--5`}>{snapshotDetail?.snapshotDetail?.status}</span></p>
            <div className="u-fontSize--small"><span className="u-marginRight--5 u-fontWeight--medium u-color--chestnut">{`${snapshotDetail?.snapshotDetail?.warnings ? snapshotDetail?.snapshotDetail?.errors.length : 0} errors`}</span><span className="replicated-link" onClick={() => this.downloadLogs()}>Download logs</span></div>
          </div>
        </div>

        {snapshotDetail?.snapshotDetail?.volumes?.length ?
          <div className="flex-column flex-auto u-marginTop--30 u-marginBottom--40">
            <p className="u-fontSize--larger u-fontWeight--bold u-color--tuna u-marginBottom--10">Snapshot timeline</p>
            <div className="flex1" id="chart">
              <ReactApexChart options={this.state.options} series={series} type="rangeBar" height={110} />
              <div className="flex flex1">
                <div className="flex flex1">
                  <p className="u-fontSize--normal u-fontWeight--normal u-color--dustyGray">
                    Started: <span className="u-fontWeight--bold u-color--doveGray"> {this.calculateVolumeTimeInterval(snapshotDetail?.snapshotDetail?.volumes).minStarted}</span>
                  </p>
                </div>
                <div className="flex flex1 justifyContent--center">
                  <p className="u-fontSize--small u-fontWeight--normal u-color--dustyGray">
                    Total capture time: <span className="u-fontWeight--bold u-color--doveGray">{`${this.calculateVolumeTimeInterval(snapshotDetail?.snapshotDetail?.volumes).maxHourDifference} hr `}</span>
                    <span className="u-fontWeight--bold u-color--doveGray">{`${this.calculateVolumeTimeInterval(snapshotDetail?.snapshotDetail?.volumes).maxMinDifference} min `}</span>
                  </p>
                </div>
                <div className="flex flex1 justifyContent--flexEnd">
                  <p className="u-fontSize--normal u-fontWeight--normal u-color--dustyGray">
                    Finished: <span className="u-fontWeight--bold u-color--doveGray"> {this.calculateVolumeTimeInterval(snapshotDetail?.snapshotDetail?.volumes).maxFinished} </span>
                  </p>
                </div>
              </div>
            </div>
          </div> : null}

        <div className="flex flex-auto u-marginBottom--30">
          <div className="flex-column flex1 u-marginRight--20">
            <div className="dashboard-card-wrapper flex1">
              <p className="u-fontSize--larger u-color--tuna u-fontWeight--bold u-lineHeight--bold u-paddingBottom--10 u-borderBottom--gray">Volumes</p>
              {snapshotDetail?.snapshotDetail?.volumes?.slice(0, 3).map((volume) => (
                <div className="flex flex1 u-borderBottom--gray" key={volume.name}>
                  <div className="flex1 u-paddingBottom--15 u-paddingTop--15 u-paddingLeft--10">
                    <p className="flex1 u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--bold">{volume.name}</p>
                    <p className="u-fontSize--normal u-color--doveGray u-fontWeight--bold u-lineHeight--normal u-marginRight--20">Size:
                    <span className="u-fontWeight--normal u-color--dustyGray"> {volume.doneBytesHuman}/{volume.sizeBytesHuman} </span>
                    </p>
                  </div>
                  <div className="flex flex1 justifyContent--flexEnd alignItems--center">
                    <p className="u-fontSize--normal u-fontWeight--normal u-marginBottom--5"><span className={`status-indicator ${volume?.phase?.toLowerCase()} u-marginLeft--5`}>{volume.phase}</span></p>
                  </div>
                </div>
              ))}
              {snapshotDetail?.snapshotDetail?.volumes?.length > 3 &&
                <div className="flex flex1 justifyContent--center">
                  <span className="replicated-link u-fontSize--normal u-paddingTop--20" onClick={() => this.toggleShowAllVolumes()}>Show all {snapshotDetail?.snapshotDetail?.volumes?.length} volumes</span>
                </div>
              }
            </div>
          </div>
          <div className="flex-column flex1 u-marginLeft--20">
            <div className="dashboard-card-wrapper flex1">
              <p className="u-fontSize--larger u-color--tuna u-fontWeight--bold u-lineHeight--bold u-paddingBottom--10 u-borderBottom--gray">Namespaces</p>
              {snapshotDetail?.snapshotDetail?.namespaces?.slice(0, 3).map((namespace) => (
                <div className="flex flex1 u-borderBottom--gray" key={namespace}>
                  <div className="flex1">
                    <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--bold u-paddingBottom--10 u-paddingTop--10 u-paddingLeft--10">{namespace}</p>
                  </div>
                </div>
              ))}
              {snapshotDetail?.snapshotDetail?.namespaces?.length > 3 &&
                <div className="flex flex1 justifyContent--center">
                  <span className="replicated-link u-fontSize--normal u-paddingTop--20" onClick={() => this.toggleShowAllNamespaces()}>Show all {snapshotDetail?.snapshotDetail?.namespaces?.length} namespaces</span>
                </div>
              }
            </div>
          </div>
        </div>

        <div className="flex flex-auto u-marginBottom--30">
          <div className="flex-column flex1 u-marginRight--20">
            <div className="dashboard-card-wrapper flex1">
              <p className="u-fontSize--larger u-color--tuna u-fontWeight--bold u-lineHeight--bold u-paddingBottom--10 u-borderBottom--gray">Pre-snapshot scripts</p>
              {this.preSnapshotScripts().slice(0, 3).map((hook, i) => (
                <div className="flex flex1 u-borderBottom--gray" key={`${hook.hookName}-${hook.phase}-${i}`}>
                  <div className="flex1 u-paddingBottom--15 u-paddingTop--15 u-paddingLeft--10">
                    <p className="flex1 u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--bold">{hook.hookName}</p>
                    <div className="flex flex1 u-marginTop--5 alignItems--center">
                      <span className="u-fontWeight--normal u-color--dustyGray u-marginRight--10"> {hook.command} </span>
                      <span className="replicated-link u-fontSize--small" onClick={() => this.toggleOutputForPreScripts(hook)}> View output </span>
                    </div>
                  </div>
                </div>
              ))}
              {this.preSnapshotScripts().length > 3 &&
                <div className="flex flex1 justifyContent--center">
                  <span className="replicated-link u-fontSize--normal u-paddingTop--20" onClick={() => this.toggleShowAllPreScripts()}>Show all {this.preSnapshotScripts().length} scripts</span>
                </div>
              }
            </div>
          </div>
          <div className="flex-column flex1 u-marginLeft--20">
            <div className="dashboard-card-wrapper flex1">
              <p className="u-fontSize--larger u-color--tuna u-fontWeight--bold u-lineHeight--bold u-paddingBottom--10 u-borderBottom--gray">Post-snapshot scripts</p>
              {this.postSnapshotScripts().slice(0, 3).map((hook, i) => (
                <div className="flex flex1 u-borderBottom--gray" key={`${hook.hookName}-${hook.phase}-${i}`}>
                  <div className="flex1 u-paddingBottom--15 u-paddingTop--15 u-paddingLeft--10">
                    <p className="flex1 u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--bold">{hook.hookName}</p>
                    <div className="flex flex1 u-marginTop--5 alignItems--center">
                      <span className="u-fontWeight--normal u-color--dustyGray u-marginRight--10"> {hook.command} </span>
                      <span className="replicated-link u-fontSize--small" onClick={() => this.toggleShowAllPostScripts(hook)}> View output </span>
                    </div>
                  </div>
                </div>
              ))}
              {this.postSnapshotScripts().length > 3 &&
                <div className="flex flex1 justifyContent--center">
                  <span className="replicated-link u-fontSize--normal u-paddingTop--20" onClick={() => this.toggleShowAllPostScripts()}>Show all {this.postSnapshotScripts().length} scripts</span>
                </div>
              }
            </div>
          </div>
        </div>

        <div className="flex flex-auto u-marginBottom--30">
          <div className="flex-column flex1 u-marginRight--20">
            <div className="dashboard-card-wrapper flex1">
              <p className="u-fontSize--larger u-color--tuna u-fontWeight--bold u-lineHeight--bold u-paddingBottom--10 u-marginBottom--10 u-borderBottom--gray">Warnings</p>
              {snapshotDetail?.snapshotDetail?.warnings?.slice(0, 3).map((warning, i) => (
                <div className="flex flex1 u-borderBottom--gray" key={`${warning.title}-${i}`}>
                  <div className="flex1">
                    <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--bold u-paddingBottom--10 u-paddingTop--10 u-paddingLeft--10">{warning.title}</p>
                  </div>
                </div>
              ))}
              {snapshotDetail?.snapshotDetail?.warnings?.length > 3 &&
                <div className="flex flex1 justifyContent--center">
                  <span className="replicated-link u-fontSize--normal u-paddingTop--20" onClick={() => this.toggleShowAllWarnings()}>Show all {snapshotDetail?.snapshotDetail?.warnings?.length} warnings</span>
                </div>
              }
            </div>
          </div>
          <div className="flex-column flex1 u-marginLeft--20">
            <div className="dashboard-card-wrapper flex1">
              <p className="u-fontSize--larger u-color--tuna u-fontWeight--bold u-lineHeight--bold u-paddingBottom--10 u-borderBottom--gray">Errors</p>
              {snapshotDetail?.snapshotDetail?.errors?.slice(0, 3).map((error, i) => (
                <div className="flex flex1 u-borderBottom--gray" key={`${error.title}-${i}`}>
                  <div className="flex1 u-paddingBottom--10 u-paddingTop--10 u-paddingLeft--10">
                    <p className="u-fontSize--large u-color--tuna u-fontWeight--bold u-lineHeight--bold">{error.title}</p>
                    <p className="u-fontSize--normal u-fontWeight--normal u-color--dustyGray"> {error.message} </p>
                  </div>
                </div>
              ))}
              {snapshotDetail?.snapshotDetail?.errors?.length > 3 &&
                <div className="flex flex1 justifyContent--center">
                  <span className="replicated-link u-fontSize--normal u-paddingTop--20" onClick={() => this.toggleShowAllErrors()}>Show all {snapshotDetail?.snapshotDetail?.errors?.length} errors</span>
                </div>
              }
            </div>
          </div>
        </div>

        {showOutputForPreScripts && preScriptOutput &&
          <Modal
            isOpen={showOutputForPreScripts}
            onRequestClose={() => this.toggleOutputForPreScripts()}
            shouldReturnFocusAfterClose={false}
            contentLabel="View logs"
            ariaHideApp={false}
            className="Modal logs-modal"
          >
            <div className="Modal-body flex flex1 flex-column">
              {!selectedTab ?
                <div className="flex-column flex1 alignItems--center justifyContent--center">
                  <Loader size="60" />
                </div>
                :
                <div className="flex-column flex1">
                  {this.renderOutputTabs()}
                  <div className="flex-column flex1 u-border--gray monaco-editor-wrapper">
                    <MonacoEditor
                      language="json"
                      value={preScriptOutput[selectedTab]}
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
                </div>
              }
              <div className="u-marginTop--20 flex">
                <button type="button" className="btn primary blue" onClick={() => this.toggleOutputForPreScripts()}>Ok, got it!</button>
              </div>
            </div>
          </Modal>
        }
        {showAllVolumes &&
          <ShowAllModal
            displayShowAllModal={showAllVolumes}
            toggleShowAllModal={this.toggleShowAllVolumes}
            dataToShow={this.renderShowAllVolumes(snapshotDetail?.snapshotDetail?.volumes)}
            name="Volumes"
          />
        }
        {showAllPreSnapshotScripts &&
          <ShowAllModal
            displayShowAllModal={showAllPreSnapshotScripts}
            toggleShowAllModal={this.toggleShowAllPreScripts}
            dataToShow={this.renderShowAllPrescripts()}
            name="Pre-snapshot scripts"
          />
        }
        {showAllPostSnapshotScripts &&
          <ShowAllModal
            displayShowAllModal={showAllPostSnapshotScripts}
            toggleShowAllModal={this.toggleShowAllPostScripts}
            dataToShow={this.renderShowAllPostscripts()}
            name="Post-snapshot scripts"
          />
        }
        {showAllNamespaces &&
          <ShowAllModal
            displayShowAllModal={showAllNamespaces}
            toggleShowAllModal={this.toggleShowAllNamespaces}
            dataToShow={this.renderShowAllNamespaces(snapshotDetail?.snapshotDetail?.namespaces)}
            name="Namespaces"
          />
        }
        {showAllWarnings &&
          <ShowAllModal
            displayShowAllModal={showAllWarnings}
            toggleShowAllModal={this.toggleShowAllWarnings}
            dataToShow={this.renderShowAllWarnings(snapshotDetail?.snapshotDetail?.warnings)}
            name="Warnings"
          />
        }
        {showAllErrors &&
          <ShowAllModal
            displayShowAllModal={showAllErrors}
            toggleShowAllModal={this.toggleShowAllErrors}
            dataToShow={this.renderShowAllErrors(snapshotDetail?.snapshotDetail?.errors)}
            name="Errors"
          />
        }
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  graphql(snapshotDetail, {
    name: "snapshotDetail",
    options: ({ match }) => {
      const slug = match.params.slug;
      const id = match.params.id;
      return {
        variables: { slug, id },
        fetchPolicy: "no-cache"
      }
    }
  })
)(AppSnapshotDetail);