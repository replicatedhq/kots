import { useState } from "react";
import { act, render, screen } from "@testing-library/react";
import UpgradeStatusModal from "./UpgradeStatusModal";
import { Utilities } from "../../utilities/utilities";

jest.mock("../../utilities/utilities", () => ({
  Utilities: { logoutUser: jest.fn(), humanReadableClusterState: (value: string) => value },
}));
jest.mock("@components/shared/Loader", () => () => null);
jest.mock("react-router-dom", () => ({ Link: () => null }));

const refetchStatus = jest.fn();

function UpdatingCluster() {
  const [disconnected, setDisconnected] = useState(true);
  return <UpgradeStatusModal
    status="upgrading-cluster"
    message="Upgrading addons"
    appSlug="test-app"
    refetchStatus={refetchStatus}
    closeModal={jest.fn()}
    connectionTerminated={disconnected}
    setTerminatedState={setDisconnected}
  />;
}

describe("upgrade polling during API outages", () => {
  const originalFetch = global.fetch;
  beforeEach(() => {
    jest.useFakeTimers();
    jest.clearAllMocks();
    global.fetch = jest.fn();
  });
  afterEach(() => {
    jest.useRealTimers();
    global.fetch = originalFetch;
  });

  it("keeps polling after repeated 500s and resumes upgrade polling on recovery", async () => {
    (global.fetch as jest.Mock)
      .mockResolvedValueOnce({ ok: false, status: 500 })
      .mockResolvedValueOnce({ ok: false, status: 500 })
      .mockResolvedValue({ ok: true, status: 200 });
    render(<UpdatingCluster />);

    for (let attempt = 1; attempt <= 2; attempt++) {
      await act(async () => { await jest.advanceTimersByTimeAsync(10000); });
      expect(global.fetch).toHaveBeenCalledTimes(attempt);
      expect(screen.getByText("Cluster update in progress")).toBeInTheDocument();
      expect(Utilities.logoutUser).not.toHaveBeenCalled();
      expect(refetchStatus).not.toHaveBeenCalled();
    }
    await act(async () => { await jest.advanceTimersByTimeAsync(10000); });
    expect(global.fetch).toHaveBeenCalledTimes(3);
    await act(async () => { await jest.advanceTimersByTimeAsync(10000); });
    expect(refetchStatus).toHaveBeenCalledWith("test-app");
    expect(Utilities.logoutUser).not.toHaveBeenCalled();
  });

  it("logs out on an actual 401", async () => {
    (global.fetch as jest.Mock).mockResolvedValue({ ok: false, status: 401 });
    render(<UpdatingCluster />);
    await act(async () => { await jest.advanceTimersByTimeAsync(10000); });
    expect(Utilities.logoutUser).toHaveBeenCalledTimes(1);
  });
});
