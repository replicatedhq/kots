import React, { useState, useEffect } from 'react';
import classNames from "classnames";
import { Utilities } from "@src/utilities/utilities";
import "@src/scss/components/DebugInfo.scss";

const DebugInfo = () => {
  const [debugData, setDebugData] = useState(null);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    fetchDebugInfo();
    const intervalId = setInterval(fetchDebugInfo, 10000);
    return () => clearInterval(intervalId);
  }, []);

  async function fetchDebugInfo() {
    try {
      const response = await fetch(`${process.env.API_ENDPOINT}/debug`, {
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
      });
      const data = await response.json();
      setDebugData(data);
    } catch (error) {
      console.error("Error fetching debug info:", error);
    } finally {
      setLoading(false);
    }
  }

  function isStale(clientInfo) {
    const timestamps = [
      clientInfo.lastPingSent.time,
      clientInfo.lastPongRecv.time,
      clientInfo.lastPongRecv.time,
      clientInfo.lastPingRecv.time
    ];
    return timestamps.some(timestamp => new Date() - new Date(timestamp) > 60 * 1000);
  };

  if (loading) {
    return <div className="loading">Loading debug information...</div>;
  }

  if (!debugData?.wsClients) {
    return <div className="no-data">No debug information available.</div>;
  }

  return (
    <div className="u-padding--20">
      <h2 className="u-fontSize--large u-fontWeight--bold card-title u-marginBottom--10">Websocket Clients</h2>
      <div className="flex flexWrap--wrap">
        {Object.entries(debugData.wsClients).map(([nodeName, clientInfo]) => (
          <div key={nodeName} className="card-bg u-marginBottom--20 u-marginRight--20" style={{ maxWidth: "50%" }}>
            <p className="u-fontSize--normal u-fontWeight--medium u-marginBottom--10 card-title">
              <span className={classNames("node-status u-marginRight--5", { disconnected: isStale(clientInfo)})}/>
              {nodeName}
            </p>
            <div className="card-item u-marginTop--5 u-marginBottom--5">
              <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-padding--5">
                <strong>Connected At:</strong> {Utilities.dateFormat(clientInfo.connectedAt, "MM/DD/YY @ hh:mm a z")}
              </p>
            </div>
            <div className="card-item u-marginTop--5 u-marginBottom--5">
              <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-padding--5">
                <strong>Last Ping Sent:</strong> {Utilities.dateFormat(clientInfo.lastPingSent.time, "MM/DD/YY @ hh:mm:ss a z")} - (Message: <strong>{clientInfo.lastPingSent.message || "N/A"})</strong>
              </p>
            </div>
            <div className="card-item u-marginTop--5 u-marginBottom--5">
              <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-padding--5">
                <strong>Last Pong Received:</strong> {Utilities.dateFormat(clientInfo.lastPongRecv.time, "MM/DD/YY @ hh:mm:ss a z")} - (Message: <strong>{clientInfo.lastPongRecv.message || "N/A"})</strong>
              </p>
            </div>
            <div className="card-item u-marginTop--5 u-marginBottom--5">
              <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-padding--5">
                <strong>Last Ping Received:</strong> {Utilities.dateFormat(clientInfo.lastPingRecv.time, "MM/DD/YY @ hh:mm:ss a z")} (Message: <strong>{clientInfo.lastPingRecv.message || "N/A"})</strong>
              </p>
            </div>
            <div className="card-item u-marginTop--5 u-marginBottom--5">
              <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-padding--5">
                <strong>Last Pong Sent:</strong> {Utilities.dateFormat(clientInfo.lastPongSent.time, "MM/DD/YY @ hh:mm:ss a z")} - (Message: <strong>{clientInfo.lastPongSent.message || "N/A"})</strong>
              </p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
};

export default DebugInfo;
