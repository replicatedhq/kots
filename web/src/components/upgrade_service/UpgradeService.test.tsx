/**
 * @jest-environment jest-fixed-jsdom
 */

import { http, HttpResponse } from "msw";
import { setupServer, SetupServerApi } from "msw/node";
import { render } from "@testing-library/react";
import { UpgradeService } from "./UpgradeService";
import { MemoryRouter, Route, Routes } from "react-router-dom";

describe("UpgradeService", () => {
  const api = "http://test-api";
  const slug = "my-test-app";

  it("Loading screen is present", async () => {
    const { getByText } = render(
      <MemoryRouter initialEntries={[`/upgrade-service/app/${slug}`]}>
        <Routes>
          <Route path="/upgrade-service/*" element={<UpgradeService />} />
        </Routes>
      </MemoryRouter>
    );

    expect(getByText("Checking required steps...")).toBeDefined();
  });

  describe("Initial state request", () => {
    let server: SetupServerApi;

    // Override the API url used by the query
    beforeAll(() => {
      process.env.API_ENDPOINT = api;
    });

    afterAll(() => {
      process.env.API_ENDPOINT = undefined;
    });

    afterEach(() => {
      // Remove any handlers added
      // in individual tests (runtime handlers).
      server.resetHandlers();
      server.close();
    });

    it("We get routed to the config section if the initial request succeeds and the app is configurable", async () => {
      server = setupServer(
        http.get(`${api}/upgrade-service/app/${slug}`, () => {
          return HttpResponse.json({
            isConfigurable: true,
            hasPreflight: false,
          });
        })
      );
      server.listen();

      const { findByTestId } = render(
        <MemoryRouter initialEntries={[`/upgrade-service/app/${slug}`]}>
          <Routes>
            <Route path="/upgrade-service/*" element={<UpgradeService />} />
          </Routes>
        </MemoryRouter>
      );

      await findByTestId("config-area");
    });

    it("We get routed to the preflight section if the initial request succeeds and the app is not configurable", async () => {
      server = setupServer(
        http.get(`${api}/upgrade-service/app/${slug}`, () => {
          return HttpResponse.json({
            isConfigurable: false,
            hasPreflight: true,
          });
        })
      );
      server.listen();

      const { findByTestId, getByText } = render(
        <MemoryRouter initialEntries={[`/upgrade-service/app/${slug}`]}>
          <Routes>
            <Route path="/upgrade-service/*" element={<UpgradeService />} />
          </Routes>
        </MemoryRouter>
      );

      await findByTestId("preflight-check-area");

      expect(getByText("Back: Config")).toBeDisabled();
    });

    it("We get routed to the confirm and deploy section if the initial request succeeds and the app is not configurable and doesn't have preflights", async () => {
      server = setupServer(
        http.get(`${api}/upgrade-service/app/${slug}`, () => {
          return HttpResponse.json({
            isConfigurable: false,
            hasPreflight: false,
          });
        })
      );
      server.listen();

      const { findByTestId, getByText } = render(
        <MemoryRouter initialEntries={[`/upgrade-service/app/${slug}`]}>
          <Routes>
            <Route path="/upgrade-service/*" element={<UpgradeService />} />
          </Routes>
        </MemoryRouter>
      );

      await findByTestId("preflight-check-area");

      expect(getByText("Back: Config")).toBeDisabled();
    });

    it("We show an error if the get info request fails", async () => {
      server = setupServer(
        http.get(`${api}/upgrade-service/app/${slug}`, () => {
          return new HttpResponse("Not found", { status: 404 });
        })
      );
      server.listen();

      const { findByText } = render(
        <MemoryRouter initialEntries={[`/upgrade-service/app/${slug}`]}>
          <Routes>
            <Route path="/upgrade-service/*" element={<UpgradeService />} />
          </Routes>
        </MemoryRouter>
      );

      await findByText("Encountered an error");
    });
  });
});
