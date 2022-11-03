package version

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/pkg/errors"
	kotsv1beta1 "github.com/replicatedhq/kots/kotskinds/apis/kots/v1beta1"
	"github.com/replicatedhq/kots/pkg/app/types"
	"github.com/replicatedhq/kots/pkg/kotsutil"
	kotsutiltypes "github.com/replicatedhq/kots/pkg/kotsutil/types"
	"github.com/replicatedhq/kots/pkg/logger"
	"github.com/replicatedhq/kots/pkg/store"
	"github.com/replicatedhq/kots/pkg/util"
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
					Query:  `sum(node_filesystem_size_bytes{job=~"node-exporter|kubernetes-service-endpoints",fstype!="",instance!=""} - node_filesystem_avail_bytes{job=~"node-exporter|kubernetes-service-endpoints",fstype!="",instance!=""}) by (instance)`,
					Legend: "Used: {{ instance }}",
				},
				{
					Query:  `sum(node_filesystem_avail_bytes{job=~"node-exporter|kubernetes-service-endpoints",fstype!="",instance!=""}) by (instance)`,
					Legend: "Available: {{ instance }}",
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

	archiveDir, err := ioutil.TempDir("", "kotsadm")
	if err != nil {
		return graphs, errors.Wrap(err, "failed to create temp dir")
	}
	defer os.RemoveAll(archiveDir)

	err = kotsStore.GetAppVersionArchive(app.ID, sequence, archiveDir)
	if err != nil {
		return graphs, errors.Wrap(err, "failed to get app version archive")
	}

	registrySettings, err := store.GetStore().GetRegistryDetailsForApp(app.ID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get app registry info")
	}

	kotsKinds, err := kotsutil.LoadKotsKindsFromPath(kotsutiltypes.LoadKotsKindsFromPathOptions{
		FromDir:          archiveDir,
		RegistrySettings: registrySettings,
		AppSlug:          app.Slug,
		Sequence:         sequence,
		IsAirgap:         app.IsAirgap,
		Namespace:        util.AppNamespace(),
	})
	if err != nil {
		return graphs, errors.Wrap(err, "failed to load kots kinds from path")
	}

	if len(kotsKinds.KotsApplication.Spec.Graphs) > 0 {
		graphs = kotsKinds.KotsApplication.Spec.Graphs
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

	b, err := ioutil.ReadAll(resp.Body)
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
