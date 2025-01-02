import { ChangeEvent, Component } from "react";
import dayjs from "dayjs";
import localizedFormat from "dayjs/plugin/localizedFormat";
import Handlebars from "handlebars";
import Modal from "react-modal";
import { getValueFormat } from "@grafana/data";
import {
  XYPlot,
  XAxis,
  YAxis,
  HorizontalGridLines,
  VerticalGridLines,
  LineSeries,
  DiscreteColorLegend,
  Crosshair,
  // @ts-ignore
} from "react-vis";
import { Utilities } from "@src/utilities/utilities";
import { Repeater } from "@src/utilities/repeater";
import ConfigureGraphs from "@src/components/shared/ConfigureGraphs";
import "@src/scss/components/watches/DashboardCard.scss";
import "@src/scss/components/apps/AppLicense.scss";
import Icon from "@src/components/Icon";
import { Chart } from "@types";

dayjs.extend(localizedFormat);

type Props = {
  prometheusAddress: string;
  metrics: Chart[];
};

type State = {
  activeChart: Chart | null;
  crosshairValues: { x: number; y: number; pod: string }[];
  getAppDashboardJob: Repeater;
  promValue: string;
  savingPromError: string;
  savingPromValue: boolean;
  showConfigureGraphs: boolean;
};
export default class DashboardGraphsCard extends Component<Props, State> {
  constructor(props: Props) {
    super(props);
    this.state = {
      activeChart: null,
      crosshairValues: [],
      getAppDashboardJob: new Repeater(),
      promValue: "",
      savingPromError: "",
      savingPromValue: false,
      showConfigureGraphs: false,
    };
  }

  toggleConfigureGraphs = () => {
    const { showConfigureGraphs } = this.state;
    this.setState({
      showConfigureGraphs: !showConfigureGraphs,
    });
  };

  updatePromValue = () => {
    this.setState({ savingPromValue: true, savingPromError: "" });

    fetch(`${process.env.API_ENDPOINT}/prometheus`, {
      headers: {
        "Content-Type": "application/json",
      },
      credentials: "include",
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
        this.toggleConfigureGraphs();
        this.setState({ savingPromValue: false, savingPromError: "" });
      })
      .catch((err) => {
        console.log(err);
        this.setState({
          savingPromValue: false,
          savingPromError: err?.message,
        });
      });
  };

  onPromValueChange = (e: ChangeEvent<HTMLInputElement>) => {
    const { value } = e.target;
    this.setState({
      promValue: value,
    });
  };

  getLegendItems = (chart: Chart) => {
    return chart.series.map((series) => {
      const metrics: {
        [name: string]: string | number;
      } = {
        name: "",
        length: 0,
      };
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
      return Object.keys(metrics).length > 0
        ? metrics[Object.keys(metrics)[0]]
        : "";
    });
  };

  getValue = (chart: Chart, value: number) => {
    let yAxisTickFormat = null;
    if (chart.tickFormat) {
      const valueFormatter = getValueFormat(chart.tickFormat);
      yAxisTickFormat = (v: number) =>
        // TODO: fix typecheck
        //  Math.round expects number, but valueFormatter returns string
        // @ts-ignore
        `${Math.round(valueFormatter(v).text)} ${valueFormatter(v).suffix}`;
      return yAxisTickFormat(value);
    } else if (chart.tickTemplate) {
      try {
        const template = Handlebars.compile(chart.tickTemplate);
        yAxisTickFormat = (v: number) => `${template({ values: v })}`;
        return yAxisTickFormat(value);
      } catch (err) {
        console.error("Failed to compile y axis tick template", err);
      }
    } else {
      return value.toFixed(5);
    }
  };

  renderGraph = (chart: Chart) => {
    const axisStyle = {
      title: { fontSize: "12px", fontWeight: 500, fill: "#4A4A4A" },
      ticks: { fontSize: "12px", fontWeight: 400, fill: "#4A4A4A" },
    };
    const legendItems = this.getLegendItems(chart);
    const series = chart.series.map((sr, idx) => {
      const data = sr.data.map(
        (valuePair: { timestamp: number; value: number }) => {
          return { x: valuePair.timestamp, y: valuePair.value };
        }
      );

      return (
        <LineSeries
          key={idx}
          data={data}
          // TODO: Fix typing for onNearestX, not sure what the types are
          // eslint-disable-next-line
          onNearestX={(_value: any, { index }: any) =>
            this.setState({
              crosshairValues: chart.series.map((s) => ({
                x: s.data[index].timestamp,
                y: s.data[index].value,
                pod: s.metric[0].value,
              })),
              activeChart: chart,
            })
          }
        />
      );
    });

    let yAxisTickFormat = null;
    if (chart.tickFormat) {
      const valueFormatter = getValueFormat(chart.tickFormat);
      yAxisTickFormat = (v: string) =>
        `${Math.round(
          // TODO: Fix valueFormatter typing
          // Math.round expects number, but valueFormatter returns string
          // @ts-ignore
          valueFormatter(v).text
          // @ts-ignore
        )} ${valueFormatter(v).suffix}`;
    } else if (chart.tickTemplate) {
      try {
        const template = Handlebars.compile(chart.tickTemplate);
        yAxisTickFormat = (v: number) => `${template({ values: v })}`;
      } catch (err) {
        console.error("Failed to compile y axis tick template", err);
      }
    }

    return (
      <div
        className="dashboard-card graph GraphCard-content--wrapper flex-column"
        key={chart.title}
      >
        <XYPlot
          width={344}
          height={180}
          onMouseLeave={() => this.setState({ crosshairValues: [] })}
          margin={{ left: 60 }}
        >
          <VerticalGridLines />
          <HorizontalGridLines />
          <XAxis
            tickFormat={(v: number) => `${dayjs.unix(v).format("H:mm")}`}
            style={axisStyle}
          />
          <YAxis width={60} tickFormat={yAxisTickFormat} style={axisStyle} />
          {series}
          {this.state.crosshairValues?.length > 0 &&
            this.state.activeChart === chart && (
              <Crosshair values={this.state.crosshairValues}>
                <div
                  className="flex flex-column"
                  style={{ background: "black", width: "250px" }}
                >
                  <p className="u-fontWeight--bold u-textAlign--center">
                    {" "}
                    {dayjs
                      .unix(this.state.crosshairValues[0].x)
                      .format("LLL")}{" "}
                  </p>
                  <br />
                  {this.state.crosshairValues.map((c, i) => {
                    return (
                      <div
                        className="flex-auto flex flexWrap--wrap u-padding--5"
                        key={i}
                      >
                        <div className="flex flex1">
                          <p className="u-fontWeight--normal">{c.pod}:</p>
                        </div>
                        <div className="flex flex1">
                          <span className="u-fontWeight--bold u-marginLeft--10">
                            {this.getValue(chart, c.y)}
                          </span>
                        </div>
                      </div>
                    );
                  })}
                </div>
              </Crosshair>
            )}
        </XYPlot>
        {legendItems ? (
          <DiscreteColorLegend
            className="legends"
            height={120}
            items={legendItems}
          />
        ) : null}
        <div className="u-marginTop--10 u-paddingBottom--10 u-textAlign--center">
          <p className="u-fontSize--normal u-fontWeight--bold u-textColor--secondary u-lineHeight--normal">
            {chart.title}
          </p>
        </div>
      </div>
    );
  };

  render() {
    const { prometheusAddress, metrics } = this.props;
    const { promValue, showConfigureGraphs, savingPromError, savingPromValue } =
      this.state;

    return (
      <div
        className={`${
          !prometheusAddress ? "inverse-card" : ""
        } card-bg flex-column flex1`}
      >
        <div className="flex justifyContent--spaceBetween alignItems--center">
          <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold">
            Monitoring
          </p>
          <div className="flex alignItems--center">
            <Icon
              icon="settings-gear-outline"
              size={16}
              className="clickable u-marginRight--5"
            />
            <span
              className="link u-fontSize--small"
              onClick={this.toggleConfigureGraphs}
            >
              Configure Prometheus Address
            </span>
          </div>
        </div>
        {prometheusAddress ? (
          <div className="Graphs-wrapper">{metrics.map(this.renderGraph)}</div>
        ) : (
          <div className="flex flex1 justifyContent--center u-paddingTop--50 u-paddingBottom--50 u-position--relative">
            <ConfigureGraphs
              updatePromValue={this.updatePromValue}
              promValue={promValue}
              savingPromValue={savingPromValue}
              savingPromError={savingPromError}
              onPromValueChange={this.onPromValueChange}
              placeholder={prometheusAddress}
            />
          </div>
        )}
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
            placeholder={prometheusAddress}
          />
        </Modal>
      </div>
    );
  }
}
