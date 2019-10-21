import React from "react";
import classNames from "classnames";

export default function NodeRow(props) {
  const { node } = props;
  const nodeIsConnected = node.status === "Connected";

  return (
    <div className="flex flex-auto NodeRow--wrapper">
      <div className="flex-column flex1">
        <div className="flex flex-auto alignItems--center u-fontWeight--bold u-color--tuna">
          <p className="u-fontSize--normal u-fontWeight--bold u-color--tuna">
            {node.hostname}
          </p>
        </div>
        <div className="flex flex1 alignItems--center u-marginTop--10 NodeRow--items">
          <p className="flex1 u-fontSize--small u-fontWeight--medium u-color--tuna u-marginRight--30">
            <span className={classNames("node-status", node.status.toLowerCase())}></span>
            {node.status}
          </p>
          <p className="flex1 u-fontSize--small u-fontWeight--medium u-color--tuna u-marginRight--30">
            <span className="icon versionHistoryIcon"></span>
            {node.version}
          </p>
          <p className="flex1 u-fontSize--small u-fontWeight--medium u-color--tuna u-marginRight--30">
            <span className="icon analysis-os_cpu"></span>
            {`${node.cores} ${node.cores === 1 ? "core" : "cores"}`}
          </p>
          <p className="flex1 u-fontSize--small u-fontWeight--medium u-color--tuna">
            <span className="icon analysis-os_memory"></span>
            {node.ram}gb
          </p>
        </div>
        <div className="u-marginTop--10">
          <p className="u-color--dustyGray u-fontSize--small u-fontWeight--normal">For more details run <span className="inline-code">kubectl describe node {node.hostname}</span></p>
        </div>
      </div>
      <div className="flex-auto flex-column justifyContent--center">
        <button onClick={() => nodeIsConnected ? props.drainNode(node.id) : props.deleteNode(node.id) } className="btn secondary red">{nodeIsConnected ? "Drain node" : "Delete node"}</button>
      </div>
    </div>
  )

}