package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/pkg/errors"
	"github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/render"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
	kotsv1beta1 "github.com/replicatedhq/kotskinds/apis/kots/v1beta1"
)

type MetricChart struct {
	Title        string   `json:"title"`
	TickFormat   string   `json:"tickFormat"`
	TickTemplate string   `json:"tickTemplate"`
	Series       []Series `json:"series"`
}

type Series struct {
	LegendTemplate string      `json:"legendTemplate"`
	Metric         []Metric    `json:"metric"`
	Data           []ValuePair `json:"data"`
}

type Metric struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type ValuePair struct {
	Timestamp float64 `json:"timestamp"`
	Value     float64 `json:"value"`
}

type SampleStream struct {
	Metric map[string]string `json:"metric"`
	Values [][2]interface{}  `json:"values"`
}

var (
	DefaultQueryDurationSeconds uint = 15 * 60 // 15 minutes
	DefaultGraphStepPoints      uint = 80
	DefaultMetricGraphs              = []kotsv1beta1.MetricGraph{
		{
			Title: "Disk Usage",
			Queries: []kotsv1beta1.MetricQuery{
				{
					Query:  `node_filesystem_size_bytes{fstype=~"ext4|xfs|btrfs|zfs",instance!="",job=~"node-exporter|kubernetes-service-endpoints"} - node_filesystem_avail_bytes{fstype=~"ext4|xfs|btrfs|zfs",instance!="",job=~"node-exporter|kubernetes-service-endpoints"}`,
					Legend: "Used: {{ instance }}-{{ device }}",
				},
				{
					Query:  `node_filesystem_avail_bytes{fstype=~"ext4|xfs|btrfs|zfs",instance!="",job=~"node-exporter|kubernetes-service-endpoints"}`,
					Legend: "Available: {{ instance }}-{{ device }}",
				},
			},
			YAxisFormat:   "bytes",
			YAxisTemplate: "{{ value }} bytes",
		},
		{
			Title:  "CPU Usage",
			Query:  fmt.Sprintf(`sum(rate(container_cpu_usage_seconds_total{namespace="%s",container!="POD",pod!=""}[5m])) by (pod)`, util.PodNamespace),
			Legend: "{{ pod }}",
		},
		{
			Title:       "Memory Usage",
			Query:       fmt.Sprintf(`sum(container_memory_usage_bytes{namespace="%s",container!="POD",pod!=""}) by (pod)`, util.PodNamespace),
			Legend:      "{{ pod }}",
			YAxisFormat: "bytes",
		},
	}
)

// GetGraphs returns the rendered graphs for the given app.
// If there are no graphs or an error is encountered, the default set of graphs is returned.
func GetGraphs(app *types.App, sequence int64, kotsStore store.Store) ([]kotsv1beta1.MetricGraph, error) {
	graphs := DefaultMetricGraphs

	archiveDir, err := os.MkdirTemp("", "kotsadm")
	if err != nil {
		return graphs, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	err = kotsStore.GetAppVersionArchive(app.ID, sequence, archiveDir)
	if err != nil {
		return graphs, errors.Wrap(err, "failed to get app version archive")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(filepath.Join(archiveDir, "upstream"))
	if err != nil {
		return graphs, errors.Wrap(err, "failed to load kots kinds from path")
	}

	registrySettings, err := kotsStore.GetRegistryDetailsForApp(app.ID)
	if err != nil {
		return graphs, errors.Wrap(err, "failed to get registry settings for app")
	}

	templatedKotsApplication, err := kotsKinds.Marshal("kots.io", "v1beta1", "Application")
	if err != nil {
		return graphs, errors.Wrap(err, "failed to marshal kots application")
	}

	renderedKotsApplication, err := render.RenderFile(kotsKinds, registrySettings, app.Slug, sequence, app.IsAirgap, util.PodNamespace, []byte(templatedKotsApplication))
	if err != nil {
		return graphs, errors.Wrap(err, "failed to render kots application")
	}

	kotsApplication, err := kotsutil.LoadKotsAppFromContents(renderedKotsApplication)
	if err != nil {
		return graphs, errors.Wrap(err, "failed to load kots application from contents")
	}

	if len(kotsApplication.Spec.Graphs) > 0 {
		graphs = kotsApplication.Spec.Graphs
	}

	return graphs, nil
}

func GetMetricCharts(graphs []kotsv1beta1.MetricGraph, prometheusAddress string) []MetricChart {
	endTime := uint(time.Now().Unix())
	charts := []MetricChart{}
	for _, graph := range graphs {
		queries := []kotsv1beta1.MetricQuery{}

		if graph.Query != "" {
			query := kotsv1beta1.MetricQuery{
				Query:  graph.Query,
				Legend: graph.Legend,
			}
			queries = append(queries, query)
		}

		for _, query := range graph.Queries {
			queries = append(queries, query)
		}

		series := []Series{}
		for _, query := range queries {
			duration := DefaultQueryDurationSeconds
			if graph.DurationSeconds > 0 {
				duration = graph.DurationSeconds
			}

			matrix, err := prometheusQueryRange(prometheusAddress, query.Query, endTime-duration, endTime, duration/DefaultGraphStepPoints)
			if err != nil {
				logger.Error(errors.Wrap(err, "failed to prometheus query range"))
				continue // don't stop
			}

			for _, sampleStream := range matrix {
				data := []ValuePair{}
				for _, v := range sampleStream.Values {
					timestamp := v[0].(float64)
					value, _ := strconv.ParseFloat(v[1].(string), 64)
					valuePair := ValuePair{
						Timestamp: timestamp,
						Value:     value,
					}
					data = append(data, valuePair)
				}

				metric := []Metric{}
				for k, v := range sampleStream.Metric {
					m := Metric{
						Name:  k,
						Value: v,
					}
					metric = append(metric, m)
				}

				s := Series{
					LegendTemplate: query.Legend,
					Metric:         metric,
					Data:           data,
				}
				series = append(series, s)
			}
		}

		chart := MetricChart{
			Title:        graph.Title,
			TickFormat:   graph.YAxisFormat,
			TickTemplate: graph.YAxisTemplate,
			Series:       series,
		}
		charts = append(charts, chart)
	}

	return charts
}

func prometheusQueryRange(address string, query string, start uint, end uint, step uint) ([]SampleStream, error) {
	host := fmt.Sprintf("%s/api/v1/query_range", address)

	v := url.Values{}
	v.Set("query", query)
	v.Set("start", fmt.Sprintf("%d", start))
	v.Set("end", fmt.Sprintf("%d", end))
	v.Set("step", fmt.Sprintf("%d", step))

	uri := host + "?" + v.Encode()
	req, err := util.NewRequest("GET", uri, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create request")
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to do req")
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.Errorf("Unexpected status code %d", resp.StatusCode)
	}

	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read response body")
	}

	type ResponseData struct {
		Result     []SampleStream `json:"result"`
		ResultType string         `json:"resultType"`
	}
	type Response struct {
		Data ResponseData `json:"data"`
	}
	var response Response
	if err := json.Unmarshal(b, &response); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal response body")
	}

	if response.Data.ResultType != "matrix" {
		return nil, errors.Wrapf(err, "unexpected result type %s", response.Data.ResultType)
	}

	return response.Data.Result, nil
}
