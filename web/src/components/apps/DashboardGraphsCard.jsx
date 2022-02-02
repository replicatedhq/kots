import React from "react";
import dayjs from "dayjs";
import Handlebars from "handlebars/runtime";
import Modal from "react-modal";
import { getValueFormat } from "@grafana/ui"
import { XYPlot, XAxis, YAxis, HorizontalGridLines, VerticalGridLines, LineSeries, DiscreteColorLegend, Crosshair } from "react-vis";
import { Utilities } from "../../utilities/utilities";
import { Repeater } from "../../utilities/repeater";
import ConfigureGraphs from "../shared/ConfigureGraphs";
import "../../scss/components/watches/DashboardCard.scss";
import "@src/scss/components/apps/AppLicense.scss";

export default class DashboardGraphsCard extends React.Component {

  state = {
    showConfigureGraphs: false,
    promValue: "",
    savingPromValue: false,
    savingPromError: "",
    getAppDashboardJob: new Repeater(),
  }

  toggleConfigureGraphs = () => {
    const { showConfigureGraphs } = this.state;
    this.setState({
      showConfigureGraphs: !showConfigureGraphs
    });
  }

  getAppDashboard = () => {
    return new Promise((resolve, reject) => {
      fetch(`${process.env.API_ENDPOINT}/app/${this.props.appSlug}/cluster/${this.props.clusterId}/dashboard`, {
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

  updatePromValue = () => {
    this.setState({ savingPromValue: true, savingPromError: "" });

    fetch(`${process.env.API_ENDPOINT}/prometheus`, {
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
      <div className="dashboard-card graph LicenseCard-content--wrapper flex-column flex1" key={chart.title}>
        <XYPlot width={360} height={180} onMouseLeave={() => this.setState({ crosshairValues: [] })} margin={{ left: 60 }}>
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
          <p className="u-fontSize--normal u-fontWeight--bold u-textColor--secondary u-lineHeight--normal">{chart.title}</p>
        </div>
      </div>
    );
  }

  componentDidMount() {
    if (this.props.prometheusAddress) {
      this.state.getAppDashboardJob.start(this.getAppDashboard, 2000);
    }
  }

  componentWillUnmount() {
    this.state.getAppDashboardJob.stop();
  }

  render() {
    const { prometheusAddress, metrics } = this.props;
    const { promValue, showConfigureGraphs, savingPromError, savingPromValue } = this.state;

    return (
      <div className={`${!prometheusAddress ? "inverse-card" : ""} dashboard-card flex-column flex1`}>
        <div className="flex justifyContent--spaceBetween alignItems--center">
          <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold">Monitoring</p>
          <div className="flex alignItems--center">
            <span className="icon clickable dashboard-card-settings-icon u-marginRight--5" />
            <span className="replicated-link u-fontSize--small" onClick={this.toggleConfigureGraphs}>Configure Prometheus Address</span>
          </div>
        </div>
        {prometheusAddress ?
          <div className="u-marginTop--10">
            <div className="flex flex1">
              {metrics.map(this.renderGraph)}
            </div>
          </div>
          :
          <div className="flex flex1 justifyContent--center u-paddingTop--50 u-paddingBottom--50 u-position--relative">
            <ConfigureGraphs
              updatePromValue={this.updatePromValue}
              promValue={promValue}
              savingPromValue={savingPromValue}
              savingPromError={savingPromError}
              onPromValueChange={this.onPromValueChange}
            />
          </div>
        }
        <Modal
          isOpen={showConfigureGraphs}
          onRequestClose={this.toggleConfigureGraphs}
          shouldReturnFocusAfterClose={false}
          contentLabel="Configure prometheus value"
          ariaHideApp={false}
          className="Modal"
        >
          <ConfigureGraphs
            toggleConfigureGraphs={this.toggleConfigureGraphs}
            updatePromValue={this.updatePromValue}
            promValue={promValue}
            savingPromValue={savingPromValue}
            savingPromError={savingPromError}
            onPromValueChange={this.onPromValueChange}
          />
        </Modal>
      </div>
    );
  }
}
