import * as React from "react";
import { withRouter } from "react-router-dom";
import ReactTooltip from "react-tooltip"
import Loader from "../shared/Loader";
import dayjs from "dayjs";
import filter from "lodash/filter";
import sortBy from "lodash/sortBy";
import isEmpty from "lodash/isEmpty";
import { Utilities } from "../../utilities/utilities";
// import { VendorUtilities } from "../../utilities/VendorUtilities";

class SupportBundleRow extends React.Component {

  renderSharedContext = () => {
    const { bundle } = this.props;
    if (!bundle) { return null; }
    // const notSameTeam = bundle.teamId !== VendorUtilities.getTeamId();
    // const isSameTeam = bundle.teamId === VendorUtilities.getTeamId();
    // const sharedIds = bundle.teamShareIds || [];
    // const isShared = sharedIds.length;
    // let shareContext;

    // if (notSameTeam) {
    //   shareContext = <span className="u-marginLeft--normal u-fontSize--normal u-color--chateauGreen">Shared by <span className="u-fontWeight--bold">{bundle.teamName}</span></span>
    // } else if (isSameTeam && isShared) {
    //   shareContext = <span className="u-marginLeft--normal u-fontSize--normal u-fontWeight--medium u-color--tundora">Shared with Replicated</span>
    // }
    // return shareContext;
  }

  handleBundleClick = (bundle) => {
    const { appType, watchSlug } = this.props;
    this.props.history.push(`/${appType === "watch" ? "watch" : "app"}/${watchSlug}/troubleshoot/analyze/${bundle.slug}`)
  }

  downloadBundle = async (bundle) => {
    const bundleId = bundle.id;
    const hiddenIFrameID = "hiddenDownloader";
    let iframe = document.getElementById(hiddenIFrameID);
    const url = `${window.env.TROUBLESHOOT_ENDPOINT}/supportbundle/${bundleId}/download?token=${Utilities.getToken()}`;
    if (iframe === null) {
      iframe = document.createElement("iframe");
      iframe.id = hiddenIFrameID;
      iframe.style.display = "none";
      document.body.appendChild(iframe);
    }
    iframe.src = url;
  }

  render() {
    const { bundle } = this.props;

    if (!bundle) {
      return null;
    }

    let noInsightsMessage;
    if (bundle && isEmpty(bundle?.analysis?.insights?.length)) {
      if (bundle.status === "uploaded" || bundle.status === "analyzing") {
        noInsightsMessage = (
          <div className="flex">
            <Loader size="14" color="#44bb66" />
            <p className="u-fontSize--small u-fontWeight--medium u-marginLeft--5 u-color--doveGray">We are still analyzing your bundle</p>
          </div>
        )
      } else {
        noInsightsMessage = <p className="u-fontSize--small u-fontWeight--medium u-color--doveGray">Unable to surface insights for this bundle</p>
      }
    }
    return (
      <div className="SupportBundle--Row u-position--relative">
        <div>
          <div className="bundle-row-wrapper">
            <div className="bundle-row flex flex1">
              <div className="flex flex1 flex-column" onClick={() => this.handleBundleClick(bundle)}>
              <div className="flex">
                <div className="flex">
                  {!this.props.isCustomer && bundle.customer ?
                    <div className="flex-column flex1 flex-verticalCenter">
                      <span className="u-fontSize--large u-color--tuna u-fontWeight--medium u-cursor--pointer">
                        <span>Collected on <span className="u-fontWeight--bold">{dayjs(bundle.createdAt).format("MMMM D, YYYY")}</span></span>
                      </span>
                    </div>
                    :
                    <div className="flex-column flex1 flex-verticalCenter">
                      <span>
                        <span className="u-fontSize--large u-cursor--pointer u-color--tuna u-fontWeight--medium">Collected on <span className="u-fontWeight--medium">{dayjs(bundle.createdAt).format("MMMM D, YYYY")}</span></span>
                        {this.renderSharedContext()}
                      </span>
                    </div>
                  }
                </div>
              </div>
              <div className="flex u-marginTop--10">
                {bundle?.analysis?.insights?.length ?
                  <div className="flex flex1 u-marginRight--5">
                    {sortBy(filter(bundle?.analysis?.insights, (i) => i.level !== "debug"), ["desiredPosition"]).map((insight, i) => (
                      <div key={i} className="analysis-icon-wrapper">
                        {insight.icon_key ?
                          <span className={`icon clickable analysis-${insight.icon_key}`} data-tip={`${bundle.id}-${i}-${insight.key}`} data-for={`${bundle.id}-${i}-${insight.key}`}></span>
                          : insight.icon ?
                            <span className="u-cursor--pointer" style={{ backgroundImage: `url(${insight.icon})` }} data-tip={`${bundle.id}-${i}-${insight.key}`} data-for={`${bundle.id}-${i}-${insight.key}`}></span>
                            : null
                        }
                        <ReactTooltip id={`${bundle.id}-${i}-${insight.key}`} effect="solid" className="replicated-tooltip">
                          <span>{insight.detail}</span>
                        </ReactTooltip>
                      </div>
                    ))}
                  </div>
                  :
                  noInsightsMessage
                }
              </div>
              </div>
              <div className="flex flex-auto alignItems--center justifyContent--flexEnd">
                <span className="u-fontSize--small u-color--astral u-fontWeight--medium u-textDecoration--underlineOnHover u-marginRight--normal" onClick={() => this.downloadBundle(bundle)}>Download bundle</span>
              </div>
            </div>
          </div>
        </div>
      </div >
    );
  }
}

export default withRouter(SupportBundleRow);
