import classNames from "classnames";
import { Link } from "react-router-dom";
import Icon from "@src/components/Icon";

export default function HelmChartSidebarItem(props) {
  const { className, helmChart } = props;
  const { helmName } = helmChart;
  const helmIcon = "";

  return (
    <div className={classNames("sidebar-link", className)}>
      <Link
        className="flex alignItems--center"
        to={`/watch/helm/${helmChart.id}`}
      >
        <span
          className="sidebar-link-icon"
          style={{ backgroundImage: `url(${helmIcon})` }}
        />
        <div className="flex-column">
          <p className="u-textColor--primary u-fontWeight--bold u-marginBottom--10">
            {helmName}
          </p>
          <div className="flex alignItems--center">
            <Icon icon="no-activity-circle-filled" size={16} />
            <span className="u-marginLeft--5 u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy">
              Pending Helm chart
            </span>
          </div>
        </div>
      </Link>
    </div>
  );
}
