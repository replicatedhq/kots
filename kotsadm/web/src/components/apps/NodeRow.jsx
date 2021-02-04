import React from "react";
import classNames from "classnames";
import Loader from "../shared/Loader";
import { rbacRoles } from "../../constants/rbac";
import { getPercentageStatus, Utilities } from '../../utilities/utilities';

export default function NodeRow(props) {
  const { node } = props;

  let drainDeleteNode;
  if (props.drainNode && Utilities.sessionRolesHasOneOf([rbacRoles.CLUSTER_ADMIN])) {
    if (props.drainingNode) {
      drainDeleteNode = (
        <div className="flex flex-auto alignItems--center">
          <span className="u-marginRight--5">
            <Loader size="25" />
          </span>
          <span className="u-fontSize--normal u-color--tundora u-fontWeight--medium">Draining Node</span>
        </div>
      );
    } else if (props.drainNodeSuccessful) {
      drainDeleteNode = (
        <div className="flex flex-auto alignItems--center">
          <span className="u-marginRight--5 icon checkmark-icon" />
          <span className="u-fontSize--normal u-color--tundora u-fontWeight--medium">Node successfully drained</span>
        </div>
      );
    } else {
      drainDeleteNode = (
        <div className="flex-auto flex-column justifyContent--center">
          <button onClick={() => node?.isConnected ? props.drainNode(node?.name) : props.deleteNode(node?.name) } className="btn secondary red">{node?.isConnected ? "Drain node" : "Delete node"}</button>
        </div>
      );
    }
  }

  return (
    <div className="flex flex-auto NodeRow--wrapper">
      <div className="flex-column flex1">
        <div className="flex flex-auto alignItems--center u-fontWeight--bold u-color--tuna">
          <p className="u-fontSize--normal u-fontWeight--bold u-color--tuna">
            {node?.name}
          </p>
        </div>
        <div className="flex flex1 alignItems--center u-marginTop--10 NodeRow--items">
          <div className="flex-column flex1 u-marginRight--10">
            <p className="flex1 u-fontSize--small u-fontWeight--medium u-color--tuna">
              <span className={classNames("node-status", { "disconnected": !node?.isConnected })}></span>
              {node?.isConnected ? "Connected" : "Disconnected"}
            </p>
            <p className="u-marginTop--5 u-color--silverSand u-fontSize--small u-fontWeight--medium">&nbsp;</p>
          </div>
          <div className="flex-column flex1 u-marginRight--10">
            <p className={classNames("flex1 u-fontSize--small u-fontWeight--medium u-color--tuna", {
              "u-color--orange": node?.pods?.available !== -1 && getPercentageStatus(node?.pods?.available, node?.pods?.capacity) === "warning",
              "u-color--red": node?.pods?.available !== -1 && getPercentageStatus(node?.pods?.available, node?.pods?.capacity) === "danger",
            })}>
              <span className={classNames("icon kubernetesLogoSmall")} />
              {
                node?.pods?.available === -1 ?
                `${node?.pods?.capacity} pods` :
                `${node?.pods?.available} pods used`
              }
            </p>
            {node?.pods?.available !== -1 && <p className="u-marginTop--5 u-color--silverSand u-fontSize--small u-fontWeight--medium">of {node?.pods?.capacity} pods available</p>}
          </div>
          <div className="flex-column flex1 u-marginRight--10">
            <p className={classNames("flex1 u-fontSize--small u-fontWeight--medium u-color--tuna", {
              "u-color--orange": node?.cpu?.available !== -1 && getPercentageStatus(node?.cpu?.available, node?.cpu?.capacity) === "warning",
              "u-color--red": node?.cpu?.available !== -1 && getPercentageStatus(node?.cpu?.available, node?.cpu?.capacity) === "danger",
            })}>
              <span className={"icon analysis-os_cpu"} />
              {
                node?.cpu?.available === -1 ?
                `${node?.cpu?.capacity} ${node?.cpu?.available === "1" ? "core" : "cores"}` :
                `${node?.cpu?.available?.toFixed(1)} ${node?.cpu?.available === "1" ? "core used" : "cores used"}`
              }
            </p>
            {node?.pods?.available !== -1 && <p className="u-marginTop--5 u-color--silverSand u-fontSize--small u-fontWeight--medium">of {node?.cpu?.capacity} {node?.cpu?.available === "1" ? "core available" : "cores available"}</p>}
          </div>
          <div className="flex-column flex1 u-marginRight--10">
            <p className={classNames("flex1 u-fontSize--small u-fontWeight--medium u-color--tuna", {
              "u-color--orange": node?.memory?.available !== -1 && getPercentageStatus(node?.memory?.available, node?.memory?.capacity) === "warning",
              "u-color--red": node?.memory?.available !== -1 && getPercentageStatus(node?.memory?.available, node?.memory?.capacity) === "danger",
            })}>
              <span className={"icon analysis-os_memory"} />
              {
                node?.memory?.available === -1 ?
                `${node?.memory?.capacity?.toFixed(1)} GB` :
                `${node?.memory?.available?.toFixed(1)} used`
              }
            </p>
            {node?.pods?.available !== -1 && <p className="u-marginTop--5 u-color--silverSand u-fontSize--small u-fontWeight--medium">of {node?.memory?.capacity?.toFixed(1)} GB available</p>}
          </div>
        </div>
        <div className="flex flex1 alignItems--center u-marginTop--15 NodeRow--items">
          <div className="flex-column flex1 u-marginRight--10">
            <p className="flex1 u-fontSize--small u-fontWeight--medium u-color--tuna">
              <span className="icon versionHistoryIcon"></span>
              {node?.kubeletVersion}
            </p>
          </div>
          <div className="flex-column flex1 u-marginRight--10">
            <p className="flex1 u-fontSize--small u-fontWeight--medium u-color--tuna">
              <span className={classNames("icon", {
                "analysis-disk": !node?.conditions?.diskPressure,
                "analysis-disk_full": node?.conditions?.diskPressure,
              })} />
              {node?.conditions?.diskPressure ? "No Space on Device" : "No Disk Pressure"}
            </p>
          </div>
          <div className="flex-column flex1 u-marginRight--10">
            <p className="flex1 u-fontSize--small u-fontWeight--medium u-color--tuna">
              <span className={classNames("icon", {
                "checkmark-icon": !node?.conditions?.memoryPressure,
                "exclamationMark--icon": node?.conditions?.memoryPressure,
              })} />
              {node?.conditions?.memoryPressure ? "No Space on Memory" : "No Memory Pressure"}
            </p>
          </div>
          <div className="flex-column flex1 u-marginRight--10">
            <p className="flex1 u-fontSize--small u-fontWeight--medium u-color--tuna">
              <span className={classNames("icon", {
                "checkmark-icon": !node?.conditions?.pidPressure,
                "exclamationMark--icon": node?.conditions?.pidPressure,
              })} />
              {node?.conditions?.pidPressure ? "Pressure on CPU" : "No CPU Pressure"}
            </p>
          </div>
        </div>
        <div className="u-marginTop--15">
          <p className="u-color--dustyGray u-fontSize--small u-fontWeight--normal">For more details run <span className="inline-code">kubectl describe node {node?.hostname}</span></p>
        </div>
      </div>
      {drainDeleteNode}
    </div>
  )

}
