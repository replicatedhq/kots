import React from "react";
import classNames from "classnames";
import { Link } from "react-router-dom";

export default function HelmChartSidebarItem(props) {
  const { className, watch } = props;
  const { helmName } = watch;
  const helmIcon = "";

  return (
    <div className={classNames('sidebar-link', className)}>
      <Link
        className="flex alignItems--center"
        to={`/watch/helm/${watch.id}`}>
          <span className="sidebar-link-icon" style={{ backgroundImage: `url(${helmIcon})` }}></span>
          <div className="flex-column">
            <p className="u-color--tuna u-fontWeight--bold u-marginBottom--10">{helmName}</p>
            <div className="flex alignItems--center">
              <div className="icon blueCircleMinus--icon" />
              <span className="u-marginLeft--5 u-fontSize--normal u-fontWeight--medium u-color--dustyGray">Pending Helm chart</span>
            </div>
          </div>
      </Link>
    </div>
  );
}
