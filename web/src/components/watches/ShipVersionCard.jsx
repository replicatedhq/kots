import * as React from "react";
import { Utilities } from "../../utilities/utilities";
import "../../scss/components/watches/VersionCard.scss";
import Loader from "../shared/Loader";


class ShipVersionCard extends React.Component {

  handleCheckbox = (isChecked) => {
    this.props.onCardChecked(this.props.versionHistory.sequence, isChecked)
  }

  handleMakeCurrent = () => {
    this.props.makeCurrentVersion(this.props.versionHistory.sequence)
  }

  render() {
    const { selected, versionHistory, isChecked, isPending } = this.props;

    if (!versionHistory || versionHistory.loading) {
      return (
        <div className="flex-column flex1 alignItems--center justifyContent--center">
          <Loader size="60" />
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
                  <p className="flex u-fontSize--normal u-color--dustyGray u-fontWeight--medium alignSelf--center u-marginLeft--10">{Utilities.dateFormat(versionHistory.createdOn, "MMMM D, YYYY")}</p>
                </div>
                {isPending &&
                  <div className="flex flex1 content-section actions-section justifyContent--flexEnd">
                    <div className="flex-column flex-auto icon-wrapper flex-verticalCenter">
                      <button className="btn secondary smallPadding" onClick={() => this.handleMakeCurrent()}>Make current</button>
                    </div>
                  </div>
                }
              </div>
            </div>
            :
            <label htmlFor={versionHistory.sequence} className={`${isChecked ? `verison-card-selected` : `verison-card`} select-view flex-column flex-verticalCenter flex-auto u-cursor--pointer`}>
              <div className="flex flex-auto">
                <div className="flex flex1 content-section flex-auto">
                  <input
                    type="checkbox"
                    className="alignSelf--center u-marginRight--5"
                    id={versionHistory.sequence}
                    name="diffReleases"
                    checked={isChecked}
                    onChange={() => { this.handleCheckbox(!isChecked) }}  />
                  <p className="flex u-fontSize--larger u-fontWeight--bold u-color--tundora u-marginRight--10 alignSelf--center">{versionHistory.title} </p>
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

export default ShipVersionCard;



