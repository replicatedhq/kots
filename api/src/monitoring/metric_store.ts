import pg from "pg";
import rp from "request-promise";
import { StatusCodeError } from "request-promise/errors";
import { Params } from "../server/params";
import { logger } from "../server/logger";
import { ParamsStore } from "../params/params_store";
import { MetricGraph, MetricQuery } from "../kots_app/kots_app_spec";
import { MetricChart, Series, Metric, ValuePair } from "./";

const DefaultQueryDurationSeconds: number = 15 * 60; // 15 minutes
const DefaultGraphStepPoints: number = 80;

export class MetricStore {
  constructor(
    private readonly pool: pg.Pool,
    private readonly params: Params,
    private readonly paramsStore: ParamsStore,
  ) {}

  async getKotsAppMetricCharts(graphs: MetricGraph[]): Promise<MetricChart[]> {
    const prometheusAddress = (await this.paramsStore.getParam("PROMETHEUS_ADDRESS")) || this.params.prometheusAddress;
    if (!prometheusAddress) {
      return [];
    }

    const endTime = new Date().getTime() / 1000; // seconds
    const charts = await Promise.all(graphs.map(async (graph: MetricGraph): Promise<MetricChart | void> => {
      try {
        const queries: MetricQuery[] = [];
        if (graph.query) {
          queries.push({
            query: graph.query,
            legend: graph.legend,
          })
        }
        if (graph.queries) {
          graph.queries.forEach(query => queries.push(query));
        }
        const series = await Promise.all(queries.map(async (query: MetricQuery): Promise<Series[]> => {
          const duration = graph.durationSeconds || DefaultQueryDurationSeconds;
          const matrix = await prometheusQueryRange(
            prometheusAddress,
            query.query,
            (endTime - duration),
            endTime, duration / DefaultGraphStepPoints,
          );
          return matrix.map((sampleStream: SampleStream): Series => {
            const data = sampleStream.values.map((value): ValuePair => {
              return {
                timestamp: value[0],
                value: value[1],
              };
            })
            // convert this cause graphql...
            const metric = Object.entries(sampleStream.metric).map(([key, value]): Metric => {
              return {
                name: key,
                value: value,
              };
            });
            return {
              legendTemplate: query.legend || "",
              metric: metric,
              data: data,
            };
          })
        }));
        return {
          title: graph.title,
          tickFormat: graph.yAxisFormat || "",
          tickTemplate: graph.yAxisTemplate || "",
          series: ([]).concat.apply([], series),
        };
      } catch(err) {
        // render all graphs that we can, catch errors and return void
        logger.error(`Failed to render graph "${graph.title}": ${err}`);
        return;
      }
    }));
    // filter void entries
    return charts.filter((value) => !!value) as MetricChart[];
  }
}

interface SampleStream {
  metric: { [ key: string]: string };
	values: [number, number][];
}

async function prometheusQueryRange(address: string, query: string, start: number, end: number, step: number): Promise<SampleStream[]> {
  try {
    const response = await rp({
      method: "GET",
      uri: `${address}/api/v1/query_range`,
      qs: {
        query: query,
        start: start,
        end: end,
        step: step,
      },
      json: true,
    });
    if (response.data.resultType != "matrix") {
      throw new Error(`unexpected response retsult type ${response.data.resultType}`);
    }
    return response.data.result as SampleStream[];
  } catch(err) {
    if (!(err instanceof StatusCodeError)) {
      throw err;
    }
    throw new Error(err.error);
  }
}
