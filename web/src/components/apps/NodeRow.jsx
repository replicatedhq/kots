import React from "react";
import classNames from "classnames";
import Loader from "../shared/Loader";
import { rbacRoles } from "../../constants/rbac";
import { getPercentageStatus, Utilities } from '../../utilities/utilities';

export default function NodeRow(props) {
  const { node } = props;

  let drainDeleteNode;
  if (props.drainNode && Utilities.sessionRolesHasOneOf([rbacRoles.CLUSTER_ADMIN])) {
    if (!props.drainNodeSuccessful && props.drainingNodeName && props.drainingNodeName === node?.name) {
      drainDeleteNode = (
        <div className="flex flex-auto alignItems--center">
          <span className="u-marginRight--5">
            <Loader size="25" />
          </span>
          <span className="u-fontSize--normal u-textColor--secondary u-fontWeight--medium">Draining Node</span>
        </div>
      );
    } else if (props.drainNodeSuccessful && props.drainingNodeName === node?.name) {
      drainDeleteNode = (
        <div className="flex flex-auto alignItems--center">
          <span className="u-marginRight--5 icon checkmark-icon" />
          <span className="u-fontSize--normal u-textColor--secondary u-fontWeight--medium">Node successfully drained</span>
        </div>
      );
    } else {
      drainDeleteNode = (
        <div className="flex-auto flex-column justifyContent--center">
          <button onClick={() => node?.canDelete ? props.deleteNode(node?.name) : props.drainNode(node?.name)} className="btn secondary red">{node?.canDelete ? "Delete node" : "Drain node"}</button>
        </div>
      );
    }
  }

  return (
    <div className="flex flex-auto NodeRow--wrapper">
      <div className="flex-column flex1">
        <div className="flex flex-auto alignItems--center u-fontWeight--bold u-textColor--primary">
          <p className="u-fontSize--normal u-fontWeight--bold u-textColor--primary">
            {node?.name}
          </p>
          {node?.isPrimaryNode && <span className="nodeTag flex-auto alignItems--center u-fontWeight--medium u-marginLeft--10">Primary node</span>}
        </div>
        <div className="flex flex1 alignItems--center u-marginTop--10 NodeRow--items">
          <div className="flex-column flex1 u-marginRight--10">
            <p className="flex1 u-fontSize--small u-fontWeight--medium u-textColor--primary">
              <span className={classNames("node-status", { "disconnected": !node?.isConnected })}></span>
              {node?.isConnected ? "Connected" : "Disconnected"}
            </p>
            <p className="u-marginTop--5 u-textColor--info u-fontSize--small u-fontWeight--medium">&nbsp;</p>
          </div>
          <div className="flex-column flex1 u-marginRight--10">
            <p className={classNames("flex1 u-fontSize--small u-fontWeight--medium u-textColor--primary", {
              "u-textColor--warning": node?.pods?.available !== -1 && getPercentageStatus(node?.pods?.available, node?.pods?.capacity) === "warning",
              "u-textColor--error": node?.pods?.available !== -1 && getPercentageStatus(node?.pods?.available, node?.pods?.capacity) === "danger",
            })}>
              <span className={classNames("icon kubernetesLogoSmall")} />
              {
                node?.pods?.available === -1 ?
                `${node?.pods?.capacity} pods` :
                `${node?.pods?.available === 0 ? "0" : (node?.pods?.capacity - node?.pods?.available)} pods used`
              }
            </p>
            {node?.pods?.available !== -1 && <p className="u-marginTop--5 u-textColor--info u-fontSize--small u-fontWeight--medium">of {node?.pods?.capacity} pods total</p>}
          </div>
          <div className="flex-column flex1 u-marginRight--10">
            <p className={classNames("flex1 u-fontSize--small u-fontWeight--medium u-textColor--primary", {
              "u-textColor--warning": node?.cpu?.available !== -1 && getPercentageStatus(node?.cpu?.available, node?.cpu?.capacity) === "warning",
              "u-textColor--error": node?.cpu?.available !== -1 && getPercentageStatus(node?.cpu?.available, node?.cpu?.capacity) === "danger",
            })}>
              <span className={"icon analysis-os_cpu"} />
              {
                node?.cpu?.available === -1 ?
                `${node?.cpu?.capacity} ${node?.cpu?.available === "1" ? "core" : "cores"}` :
                `${node?.cpu?.available === 0 ? "0" : (node?.cpu?.capacity - node?.cpu?.available).toFixed(1)} ${node?.cpu?.available === "1" ? "core used" : "cores used"}`
              }
            </p>
            {node?.pods?.available !== -1 && <p className="u-marginTop--5 u-textColor--info u-fontSize--small u-fontWeight--medium">of {node?.cpu?.capacity} {node?.cpu?.available === "1" ? "core total" : "cores total"}</p>}
          </div>
          <div className="flex-column flex1 u-marginRight--10">
            <p className={classNames("flex1 u-fontSize--small u-fontWeight--medium u-textColor--primary", {
              "u-textColor--warning": node?.memory?.available !== -1 && getPercentageStatus(node?.memory?.available, node?.memory?.capacity) === "warning",
              "u-textColor--error": node?.memory?.available !== -1 && getPercentageStatus(node?.memory?.available, node?.memory?.capacity) === "danger",
            })}>
              <span className={"icon analysis-os_memory"} />
              {
                node?.memory?.available === -1 ?
                `${node?.memory?.capacity?.toFixed(1)} GB` :
                `${node?.memory?.available === 0 ? "0" : (node?.memory?.capacity - node?.memory?.available).toFixed(1)} GB used`
              }
            </p>
            {node?.pods?.available !== -1 && <p className="u-marginTop--5 u-textColor--info u-fontSize--small u-fontWeight--medium">of {node?.memory?.capacity?.toFixed(1)} GB total</p>}
          </div>
        </div>
        <div className="flex flex1 alignItems--center u-marginTop--15 NodeRow--items">
          <div className="flex-column flex1 u-marginRight--10">
            <p className="flex1 u-fontSize--small u-fontWeight--medium u-textColor--primary">
              <span className="icon versionHistoryIcon"></span>
              {node?.kubeletVersion}
            </p>
          </div>
          <div className="flex-column flex1 u-marginRight--10">
            <p className="flex1 u-fontSize--small u-fontWeight--medium u-textColor--primary">
              <span className={classNames("icon", {
                "analysis-disk": !node?.conditions?.diskPressure,
                "analysis-disk_full": node?.conditions?.diskPressure,
              })} />
              {node?.conditions?.diskPressure ? "No Space on Device" : "No Disk Pressure"}
            </p>
          </div>
          <div className="flex-column flex1 u-marginRight--10">
            <p className="flex1 u-fontSize--small u-fontWeight--medium u-textColor--primary">
              <span className={classNames("icon", {
                "checkmark-icon": !node?.conditions?.pidPressure,
                "exclamationMark--icon": node?.conditions?.pidPressure,
              })} />
              {node?.conditions?.pidPressure ? "Pressure on CPU" : "No CPU Pressure"}
            </p>
          </div>
          <div className="flex-column flex1 u-marginRight--10">
            <p className="flex1 u-fontSize--small u-fontWeight--medium u-textColor--primary">
              <span className={classNames("icon", {
                "checkmark-icon": !node?.conditions?.memoryPressure,
                "exclamationMark--icon": node?.conditions?.memoryPressure,
              })} />
              {node?.conditions?.memoryPressure ? "No Space on Memory" : "No Memory Pressure"}
            </p>
          </div>
        </div>
        <div className="u-marginTop--15">
          <p className="u-textColor--bodyCopy u-fontSize--small u-fontWeight--normal">For more details run <span className="inline-code">kubectl describe node {node?.name}</span></p>
        </div>
      </div>
      {drainDeleteNode}
    </div>
  )

}
