import React from "react";
import { shallow } from "enzyme";
import { MemoryRouter } from "react-router-dom";

import { WatchDetailPage } from "./WatchDetailPage";

describe("<WatchDetailPage> tests", () => {
  it("Renders without crashing", () => {
    const wrapper = shallow(
      <WatchDetailPage
        match={{
          params: {
            owner: "Bobby",
            slug: "datadog"
          }
        }}
        history={{
          location: {
            pathname: "watch/Bobby/datadog"
          }
        }}
        rootDidInitialWatchFetch={true}
        listWatches={[]}
        refetchListWatches={() => {}}
        getWatchQuery={{ loading: false }}
        getHelmChartQuery={{ loading: false }}
      />
    );

    expect(wrapper).toBeTruthy();
  });

  it("Renders a Loader if the root listWatches haven't been fetched", () => {
    const wrapper = shallow(
      <WatchDetailPage
        match={{
          params: {
            owner: "Bobby",
            slug: "datadog"
          }
        }}
        history={{
          location: {
            pathname: "watch/Bobby/datadog"
          }
        }}
        rootDidInitialWatchFetch={false}
        listWatches={[]}
        refetchListWatches={() => { }}
        getWatchQuery={{ loading: false }}
        getHelmChartQuery={{ loading: false }}
      />
    );
    expect(wrapper.find('Loader')).toHaveLength(1);
  });

  it("Shows a loader for helm chart queries", () => {
    const wrapper = shallow(
      <WatchDetailPage
        match={{
          params: {
            owner: "helm",
            id: "12345"
          }
        }}
        history={{
          location: {
            pathname: "watch/helm/12345"
          }
        }}
        rootDidInitialWatchFetch={false}
        listWatches={[]}
        refetchListWatches={() => { }}
        getWatchQuery={{ loading: false }}
        getHelmChartQuery={{ loading: true }}
      />
    );

    expect(wrapper.find("Loader")).toHaveLength(1);
  });

  it("Shows a loader for watch queries", () => {
    const wrapper = shallow(
      <WatchDetailPage
        match={{
          params: {
            owner: "joe",
            slug: "grafana"
          }
        }}
        history={{
          location: {
            pathname: "watch/joe/grafana"
          }
        }}
        rootDidInitialWatchFetch={true}
        listWatches={[]}
        refetchListWatches={() => { }}
        getWatchQuery={{ loading: true }}
        getHelmChartQuery={{ loading: false }}
      />
    );
    expect(wrapper.find("Loader")).toHaveLength(1);
  });

  it("Doesn't show a loader if it doesn't need to.", () => {
    const wrapper = shallow(
      <MemoryRouter>
        <WatchDetailPage
          match={{
            params: {
              owner: "joe",
              slug: "grafana"
            }
          }}
          history={{
            location: {
              pathname: "watch/joe/grafana"
            }
          }}
          rootDidInitialWatchFetch={true}
          listWatches={[]}
          refetchListWatches={() => { }}
          getWatchQuery={{ loading: false }}
          getHelmChartQuery={{ loading: false }}
        />
      </MemoryRouter>
    );

    expect(wrapper.find("Loader")).toHaveLength(0);
  });
});
