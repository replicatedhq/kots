
import React from "react";
import { Link } from "react-router-dom";
import ReactTooltip from "react-tooltip"
import dayjs from "dayjs";
import isSameOrAfter from "dayjs/plugin/isSameOrAfter";
dayjs.extend(isSameOrAfter);

import { Utilities } from "../../utilities/utilities";


class SnapshotRow extends React.Component {

  handleDeleteClick = snapshot => {
    this.props.toggleConfirmDeleteModal(snapshot);
  }

  handleRestoreClick = snapshot => {
    this.props.toggleRestoreModal(snapshot);
  }

  render() {
    const { snapshot, app } = this.props;
    const isExpired = dayjs(new Date()).isSameOrAfter(snapshot?.expiresAt);

    return (
      <div className={`flex flex-auto SnapshotRow--wrapper alignItems--center u-marginTop--10 ${snapshot?.status === "Deleting" && "is-deleting"} ${snapshot?.status === "InProgress" && "in-progress"} ${isExpired && "is-expired"}`}>
        <div className="flex-column flex1" style={{ maxWidth: "700px" }}>
          <p className={`u-fontSize--largest ${isExpired || snapshot?.status === "Deleting" ? "u-color--dustyGray" : "u-color--tuna"} u-lineHeight--normal u-fontWeight--bold u-marginRight--10`}>{snapshot?.name}</p>
          <div className="flex flex1 alignItems--center u-marginTop--10">
            <p className="u-fontSize--small u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginRight--20">{snapshot?.startedAt ? Utilities.dateFormat(snapshot?.startedAt, "MMM D YYYY @ hh:mm a z") : "n/a"}</p>
            {snapshot?.status === "Completed" ?
              <p className="u-fontSize--small u-color--dustyGray u-fontWeight--medium u-lineHeight--normal u-marginRight--20">
                <span className={`status-indicator u-marginRight--5 ${snapshot?.status.toLowerCase()}`}>{Utilities.snapshotStatusToDisplayName(snapshot?.status)}</span>
                on {snapshot?.finishedAt ? (snapshot?.finishedAt ? Utilities.dateFormat(snapshot?.finishedAt, "MMM D YYYY @ hh:mm a z") : "TBD") : "n/a"}
              </p> :
              <span className={`status-indicator u-marginRight--5 ${snapshot?.status.toLowerCase()}`}>{Utilities.snapshotStatusToDisplayName(snapshot?.status)}</span>
            }
          </div>
        </div>
        <div className="flex flex1">
          <div className="flex flex-auto alignItems--center u-marginTop--5">
            <div className="flex flex1 alignItems--center">
              {snapshot?.volumeSizeHuman &&
                <p className="u-fontSize--normal u-color--doveGray u-fontWeight--bold u-lineHeight--normal u-marginRight--30 justifyContent--center flex alignItems--center"><span className="icon snapshot-volume-size-icon" /> {snapshot?.volumeSizeHuman} </p>}
              <p className="u-fontSize--normal u-color--doveGray u-fontWeight--bold u-lineHeight--normal justifyContent--center flex alignItems--center"><span className="icon snapshot-volume-icon" /> {snapshot?.volumeSuccessCount}/{snapshot?.volumeCount}</p>
            </div>
          </div>
        </div>
        {!isExpired && snapshot?.status !== "Deleting" &&
          <div className="flex flex-auto">
            {snapshot?.status === "Completed" &&
              <div className="flex">
                <span className="icon snapshot-restore-icon u-cursor--pointer" onClick={() => this.handleRestoreClick(snapshot)} data-tip="Restore from this backup" />
                <ReactTooltip effect="solid" className="replicated-tooltip" />
              </div>}
            {snapshot?.status !== "InProgress" &&
              <span className="icon snapshot-trash-icon u-marginLeft--20 u-cursor--pointer" onClick={() => this.handleDeleteClick(snapshot)} />}
            {!isExpired && snapshot?.status !== "Deleting" &&
              <Link to={app ? `/snapshots/partial/${this.props.app.slug}/${snapshot?.name}` : `/snapshots/details/${snapshot?.name}`} className="icon snapshot-details-icon u-marginLeft--20 u-cursor--pointer" />
            }
          </div>
        }
      </div>
    )
  }
}

export default SnapshotRow;
