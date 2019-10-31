import React from "react";
import classNames from "classnames";

import { getPercentageStatus } from '../../utilities/utilities';

export default function NodeRow(props) {
  const { node } = props;

  return (
    <div className="flex flex-auto NodeRow--wrapper">
      <div className="flex-column flex1">
        <div className="flex flex-auto alignItems--center u-fontWeight--bold u-color--tuna">
          <p className="u-fontSize--normal u-fontWeight--bold u-color--tuna">
            {node?.name}
          </p>
        </div>
        <div className="flex flex1 alignItems--center u-marginTop--10 NodeRow--items">
          <p className="flex1 u-fontSize--small u-fontWeight--medium u-color--tuna u-marginRight--10">
            <span className={classNames("node-status", { "disconnected": !node?.isConnected })}></span>
            {node?.isConnected ? "Connected" : "Disconnected"}
          </p>
          <p className="flex1 u-fontSize--small u-fontWeight--medium u-color--tuna u-marginRight--10">
            <span className="icon versionHistoryIcon"></span>
            {node?.kubeletVersion}
          </p>
          <p className={classNames("flex1 u-fontSize--small u-fontWeight--medium u-color--tuna u-marginRight--10", {
            "u-color--orange": getPercentageStatus(node?.cpu?.available, node?.cpu?.capacity) === "warning",
            "u-color--red": getPercentageStatus(node?.cpu?.available, node?.cpu?.capacity) === "danger",
          })}>
            <span className={"icon analysis-os_cpu"} />
            {`${node?.cpu?.available?.toFixed(1)} / ${node?.cpu?.capacity} ${node?.cpu?.available === "1" ? "core available" : "cores available"}`}
          </p>
          <p className={classNames("flex1 u-fontSize--small u-fontWeight--medium u-color--tuna", {
            "u-color--orange": getPercentageStatus(node?.memory?.available, node?.memory?.capacity) === "warning",
            "u-color--red": getPercentageStatus(node?.memory?.available, node?.memory?.capacity) === "danger",
          })}>
            <span className={"icon analysis-os_memory"} />
            {`${node?.memory?.available?.toFixed(1)} / ${node?.memory?.capacity?.toFixed(1)} GB available`}
          </p>
        </div>
        <div className="flex flex1 alignItems--center u-marginTop--10 NodeRow--items">
          <p className={classNames("flex1 u-fontSize--small u-fontWeight--medium u-color--tuna u-marginRight--10", {
            "u-color--orange": getPercentageStatus(node?.pods?.available, node?.pods?.capacity) === "warning",
            "u-color--red": getPercentageStatus(node?.pods?.available, node?.pods?.capacity) === "danger",
          })}>
            <span className={classNames("icon kubernetesLogoSmall")} />
            {`${node?.pods?.available} / ${node?.pods?.capacity} pods available`}
          </p>
          <p className="flex1 u-fontSize--small u-fontWeight--medium u-color--tuna u-marginRight--10">
            <span className={classNames("icon", {
              "analysis-disk": !node?.conditions?.diskPressure,
              "analysis-disk_full": node?.conditions?.diskPressure,
            })} />
            {node?.conditions?.diskPressure ? "No Space on Device" : "No Disk Pressure"}
          </p>
          <p className="flex1 u-fontSize--small u-fontWeight--medium u-color--tuna u-marginRight--10">
            <span className={classNames("icon", {
              "checkmark-icon": !node?.conditions?.memoryPressure,
              "exclamationMark--icon": node?.conditions?.memoryPressure,
            })} />
            {node?.conditions?.memoryPressure ? "No Space on Memory" : "No Memory Pressure"}
          </p>
          <p className="flex1 u-fontSize--small u-fontWeight--medium u-color--tuna">
            <span className={classNames("icon", {
              "checkmark-icon": !node?.conditions?.pidPressure,
              "exclamationMark--icon": node?.conditions?.pidPressure,
            })} />
            {node?.conditions?.pidPressure ? "Pressure on CPU" : "No CPU Pressure"}
          </p>
        </div>
        <div className="u-marginTop--10">
          <p className="u-color--dustyGray u-fontSize--small u-fontWeight--normal">For more details run <span className="inline-code">kubectl describe node {node?.hostname}</span></p>
        </div>
      </div>
      <div className="flex-auto flex-column justifyContent--center">
        <button onClick={() => node?.isConnected ? props.drainNode(node?.name) : props.deleteNode(node?.name) } className="btn secondary red">{node?.isConnected ? "Drain node" : "Delete node"}</button>
      </div>
    </div>
  )

}
