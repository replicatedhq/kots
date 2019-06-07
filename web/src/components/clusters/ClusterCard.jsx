import * as React from "react";
import PropTypes from "prop-types";
import truncateMiddle from "truncate-middle";
import "../../scss/components/clusters/ClusterCard.scss";

export default class ClusterCard extends React.Component {
  static propTypes = {
    item: PropTypes.object.isRequired,
  }

  state = {
    enabled: 1
  }

  handleEnableToggle = (e) => {
    const enabled = e.target.checked ? 1 : 0;
    this.setState({ enabled });
  }

  toggleInstallShipModal = () => {
    this.props.toggleInstallShipModal();
  }

  render() {
    const { item, handleManageClick, toggleDeleteClusterModal } = this.props;
    const type = item.gitOpsRef ? "git" : "ship";
    const gitPath = item.gitOpsRef ? `${item.gitOpsRef.owner}/${item.gitOpsRef.repo}/${item.gitOpsRef.branch}` : "";
    const upToDate = true;
    return (
      <div className="deployed-cluster flex-column flex1">
        <div className="flex-column">
          <div className="flex u-marginBottom--5">
            <span className={`normal u-marginRight--5 icon clusterType ${type}`}></span>
            <div className="flex1 justifyContent--center">
              <p className="u-fontWeight--bold u-fontSize--large u-color--tundora">{item.title}</p>
              <p className="u-fontWeight--medium u-fontSize--small u-color--dustyGray u-marginTop--5">{type === "git" ? truncateMiddle(gitPath, 22, 22, "...") : "Deployed with Ship"}</p>
            </div>
          </div>
          <div className="u-marginTop--10 u-marginBottom--5 flex flex1">
            <span className={`icon status-icon ${item.totalApplicationCount === 0 ? "blueCircleMinus--icon" : "checkmark-icon"} flex-auto u-marginRight--5`}></span>
            <div className="flex1">
              <p className="u-fontWeight--medium u-fontSize--normal u-lineHeight--normal u-color--tundora">{item.totalApplicationCount} application{item.totalApplicationCount !== 1 && "s"} deployed to cluster</p>
              <a href="/watches" className="u-fontSize--small replicated-link">View applications</a>
            </div>
          </div>
          {upToDate ? // TODO: Get real data to show here
            <div className="u-marginTop--10 u-paddingTop--10 u-marginBottom--5 u-borderTop--gray flex flex1">
              <span className="icon status-icon checkmark-icon flex-auto u-marginRight--5"></span>
              <div className="flex1">
                <p className="u-fontWeight--medium u-fontSize--normal u-lineHeight--normal u-color--chateauGreen">All applications are up to date</p>
                <p className="u-fontWeight--medium u-fontSize--small u-color--dustyGray">Check back often to keep apps up to date</p>
              </div>
            </div>
            :
            <div className="u-marginTop--10 u-paddingTop--10 u-marginBottom--5 u-borderTop--gray flex flex1">
              <span className="icon status-icon exclamationMark-icon flex-auto u-marginRight--5"></span>
              <div className="flex1">
                <p className="u-fontWeight--medium u-fontSize--normal u-lineHeight--normal u-color--orange">?? applications are out of date</p>
                <a href="/watches" className="u-fontSize--small replicated-link">View out of date applications</a>
              </div>
            </div>
          }
          <div className="flex u-marginTop--10 u-marginBottom--5 u-paddingTop--5">
            {type === "ship" ?
              <button className="btn secondary small" onClick={this.toggleInstallShipModal}>Install cluster</button>
            :
              <div style={{height: "25px"}} />
            }
          </div>
          <div className="flex flex1 alignItems--flexEnd">
            <div className="flex u-marginTop--10 u-borderTop--gray u-width--full">
              <div className="flex1 flex card-action-wrapper u-cursor--pointer">
                <span className="flex1 u-color--red card-action u-fontSize--small u-fontWeight--medium u-textAlign--center" onClick={() => { toggleDeleteClusterModal(item) }}>Delete cluster</span>
              </div>
              <div className="flex1 flex card-action-wrapper u-cursor--pointer">
                <span onClick={handleManageClick} className="flex1 u-color--astral card-action u-fontSize--small u-fontWeight--medium u-textAlign--center">Manage cluster</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }
}
