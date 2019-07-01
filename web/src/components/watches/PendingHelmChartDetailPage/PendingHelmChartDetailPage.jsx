import React from "react";
import { withRouter } from "react-router-dom";
import { compose } from "react-apollo";

import Loader from "@src/components/shared/Loader";
import PaperIcon from "@src/components/shared/PaperIcon";

export function PendingHelmChartDetailPage(props) {
  const { chart } = props;

  // Sometimes the chart isn't quite loaded yet, so throw in a loader
  // just in case we're waiting on the `loading` prop in WatchDetailPage
  // This will probably have to change...
  if (!chart) {
    return (
      <div className="flex1 flex-column alignItems--center justifyContent--center">
        <Loader size="60" />
      </div>
    );
  }

  return (
    <div className="DetailPageApplication--wrapper flex-column flex1 container alignItems--center u-overflow--auto u-paddingTop--20 u-paddingBottom--20">
      <div className="DetailPageApplication flex flex-auto flex-column">
        <div className="flex1 flex-column u-paddingRight--30">
          <div className="flex">
            <div className="flex flex-auto">
              <div
                style={{ backgroundImage: `url(${chart.helmIcon})` }}
                className="DetailPageApplication--appIcon u-position--relative">
                <PaperIcon
                  className="u-position--absolute"
                  iconClass="helm"
                  height="30px"
                  width="30px"
                />
              </div>
            </div>
            <div className="flex-column flex1 justifyContent--center u-marginLeft--10 u-paddingLeft--5">
              <p className="u-fontSize--30 u-color--tuna u-fontWeight--bold">
                {chart.helmName}
                <span className="u-fontSize--normal u-paddingLeft--20" to="#">View Chart</span>
              </p>

              <div className="u-marginTop--10 flex-column">
                <div className="flex-auto">
                  <button className="btn secondary u-marginRight--20">See values.yaml</button>
                  <button className="btn secondary">Get chart YAML</button>
                </div>
              </div>
            </div>
          </div>
          <div className="u-marginTop--30">
            <div className="flex midstream-banner justifyContent--spaceBetween">
              <div className="flex3 u-paddingRight--20">
                <p className="u-fontSize--small u-textAlign--left u-fontWeight--medium u-lineHeight--normal u-color--nevada">
                  This is a pending watch from an upstream Helm chart. You should unfork it so you can better manage updates directly from the upstream and set up automatic deployments.
                </p>
              </div>
              <div className="flex1 flex-column u-textAlign--left u-paddingLeft--20">
                <button className="btn secondary red u-marginBottom--10">
                  Ignore application
                </button>
                <span className="u-fontSize--normal" to="#">Learn more</span>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div className="flex flex-auto full-width-container justifyContent--spaceBetween u-marginTop--40 u-textAlign--center">
        <div className="flex-auto flex-column">
          <div className="flex-column">
            <div className="unfork-icon-wrapper">
              <span className="flex justifyContent--center">
                <span className="icon folder-unfork" />
              </span>
            </div>
            <p className="u-color--tuna u-fontWeight--bold u-fontSize--large u-marginTop--20">Unfork the upstream {chart.helmName} chart</p>
            <p className="u-fontSize--normal u-lineHeight--medium u-marginTop--10">Ship can migrate your forked components to be automatically updated from upstream.</p>
          </div>
          <div className="flex-column alignItems--center justifyContent--flexEnd">
            <button className="btn primary green u-marginTop--10">
              Unfork {chart.helmName}
            </button>
          </div>
        </div>
        <div className="flex-auto flex-column">
          <div className="flex-column">
            <p className="u-color--tuna u-fontWeight--bold u-fontSize--large">Current version x.x.x</p>
          </div>
          <div className="flex-column alignItems--center justifyContent--flexEnd">
            <button className="btn primary green u-marginTop--10">
              Unfork {chart.helmName}
            </button>
          </div>
        </div>
        <div className="flex-auto flex-column u-filter--grayscale">
          <div className="flex-column">
            <div className="unfork-icon-wrapper flex-column justifyContent--center alignItems--center u-marginBottom--20">
              <div className="flex alignItems--center">
                <span className="icon ship-complete-icon-gh"></span>
                <span className="deployment-or-text">OR</span>
                <span className="icon ship-medium-size"></span>
              </div>
            </div>
            <p className="u-color--tuna u-fontWeight--bold u-fontSize--large ">Enable automatic updates</p>
            <p className="u-fontSize--normal u-lineHeight--medium u-marginTop--10">
              Convert {chart.helmName} into a watch managed by a gitops workflow or ship deployment.
            </p>
          </div>
          <div className="flex-column alignItems--center justifyContent--flexEnd">
            <button className="btn primary green u-marginTop--10">
              Unfork to enable auto updates
            </button>
          </div>
        </div>
      </div>
    </div>
  );
}

export default compose(
  withRouter
)(PendingHelmChartDetailPage);
