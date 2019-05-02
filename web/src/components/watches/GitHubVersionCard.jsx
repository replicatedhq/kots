import * as React from "react";
import { Utilities } from "../../utilities/utilities";
import "../../scss/components/watches/VersionCard.scss";
import Loader from "../shared/Loader";


class GitHubVersionCard extends React.Component {
  constructor() {
    super();
  }

  handleCheckbox(isChecked) {
    this.props.onCardChecked(this.props.versionHistory.sequence, isChecked)
  }



  render() {
    const { selected, versionHistory, isChecked, pullRequestRootUrl } = this.props;

    if (!versionHistory || versionHistory.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" color="#44bb66" />
        </div>
      )
    }

    return (
      <div className="timeline flex">
        <div className="flex-auto u-marginRight--10">
          <div className="line"></div>
          <div className="dot"></div>
          <div className="line"></div>
        </div>
        <div className="VerisonCard--Row alignSelf--center">
          {!selected ?
            <div className="verison-card  flex-column flex-verticalCenter flex-auto">
              <div className="flex flex-auto">
                <div className="flex flex1 content-section flex-auto">
                  <p className="flex u-fontSize--larger u-fontWeight--bold u-color--tundora u-marginRight--10 alignSelf--center">{versionHistory.title} </p>
                  <span className={`${versionHistory.status}`}>{Utilities.toTitleCase(versionHistory.status)}</span>
                  <p className="flex u-fontSize--normal u-color--dustyGray u-fontWeight--medium alignSelf--center u-marginLeft--10">{Utilities.dateFormat(versionHistory.createdOn, "MMMM D, YYYY")}</p>
                </div>
                <div className="flex flex1 content-section actions-section justifyContent--flexEnd">
                  {/* <div className="flex-column flex-auto icon-wrapper flex-verticalCenter u-marginRight--5">
                    <button className="btn secondary smallPadding">Re-open</button>
                  </div> */}
                  <div className="flex-column flex-auto icon-wrapper flex-verticalCenter u-marginRight--5">
                    <span className={`u-marginLeft--5 icon integration-card-icon-github`}></span>
                  </div>
                  <div className="flex-column flex-auto icon-wrapper flex-verticalCenter">
                    <a href={`https://github.com/${pullRequestRootUrl}/pull/${versionHistory.pullrequestNumber}`} target="_blank" rel="noopener noreferrer" className="replicated-link u-fontSize--meidum">#{versionHistory.pullrequestNumber}</a>
                  </div>
                </div>
              </div>
            </div>
            :
            <label htmlFor={versionHistory.sequence} className={`${isChecked ? `verison-card-selected` : `verison-card`} select-view flex-column flex-verticalCenter flex-auto u-cursor--pointer`}>
              <div className="flex flex-auto">
                <div className="flex flex1 content-section flex-auto">
                  <input
                    type="checkbox"
                    className="alignSelf--center u-marginRight--"
                    id={versionHistory.sequence}
                    name="diffReleases"
                    checked={isChecked}
                    onChange={() => { this.handleCheckbox(!isChecked) }}  />
                  <p className="flex u-fontSize--larger u-fontWeight--bold u-color--tundora u-marginRight--10 alignSelf--center">{versionHistory.title} </p>
                  <span className={`${versionHistory.status}`}>{Utilities.toTitleCase(versionHistory.status)}</span>
                  <p className="flex u-fontSize--normal u-color--dustyGray u-fontWeight--medium alignSelf--center u-marginLeft--10">{Utilities.dateFormat(versionHistory.createdOn, "MMMM D, YYYY")}</p>
                </div>
              </div>
            </label>
          }
        </div>
      </div>
    );
  }
}

export default GitHubVersionCard;



