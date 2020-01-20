
import React from "react";
import { Link } from "react-router-dom";
import { Utilities } from "../../utilities/utilities";
import dayjs from "dayjs";
import isSameOrAfter from 'dayjs/plugin/isSameOrAfter'
dayjs.extend(isSameOrAfter)


class AppShanpshotRow extends React.Component {

  handleDeleteClick = snapshot => {
    this.props.toggleConfirmDeleteModal(snapshot);
  }

  handleRestoreClick = snapshot => {
    this.props.toggleRestoreModal(snapshot);
  }

  render() {
    const { appSlug, snapshot } = this.props;
    const isExpired = dayjs(new Date()).isSameOrAfter(snapshot.expires);

    
    return (
      <div className={`flex flex-auto ActiveDownstreamVersionRow--wrapper alignItems--center ${snapshot.status === "Deleting" && "is-deleting"} ${isExpired && "is-expired"}`}>
        <div className="flex-column flex1">
          <div className="flex flex-auto alignItems--center u-fontWeight--bold u-color--tuna">
            <p className={`u-fontSize--largest ${isExpired || snapshot.status === "Deleting" ? "u-color--dustyGray" : "u-color--tuna"} u-lineHeight--normal u-fontWeight--bold u-marginRight--10`}>{snapshot.name}</p>
            {!isExpired && snapshot.status !== "Deleting" && <Link className="replicated-link u-marginLeft--5 u-fontSize--small" to={`/app/${appSlug}/snapshots/${snapshot.name}`}>View</Link>}
          </div>
          <div className="flex flex-auto alignItems--center u-marginTop--5">
            <div className="flex flex1 alignItems--center">
              <p className="u-fontSize--normal u-color--doveGray u-fontWeight--bold u-lineHeight--normal u-marginRight--20"><span className="u-fontWeight--normal u-color--dustyGray">Started:</span> {Utilities.dateFormat(snapshot.started, "MMM D, YYYY h:mm A")}</p>
              <p className="u-fontSize--normal u-color--doveGray u-fontWeight--bold u-lineHeight--normal u-marginRight--20"><span className="u-fontWeight--normal u-color--dustyGray">Finished:</span> {snapshot.finished ? Utilities.dateFormat(snapshot.finished, "MMM D, YYYY h:mm A") : "TBD"}</p>
              <div>
                <span className={`status-indicator ${snapshot.status.toLowerCase()}`}>{snapshot.status}</span>
              </div>
            </div>
          </div>
          <div className="flex flex-auto alignItems--center u-marginTop--5">
            <div className="flex flex1 alignItems--center">
              <p className="u-fontSize--normal u-color--doveGray u-fontWeight--bold u-lineHeight--normal u-marginRight--20"><span className="u-fontWeight--normal u-color--dustyGray">Volumes included:</span> {snapshot.volumeSuccessCount}/{snapshot.volumeCount}</p>
              <p className="u-fontSize--normal u-color--doveGray u-fontWeight--bold u-lineHeight--normal u-marginRight--20"><span className="u-fontWeight--normal u-color--dustyGray">Backup size:</span> {snapshot.volumeSizeHuman}</p>
            </div>
          </div>
        </div>
        {!isExpired && snapshot.status !== "Deleting" &&
          <div className="flex-auto">
            <span className="u-fontSize--normal u-fontWeight--medium u-color--royalBlue u-textDecoration--underlineOnHover u-marginRight--20" onClick={() => this.handleRestoreClick(snapshot)}>Restore</span>
            <span className="u-fontSize--normal u-fontWeight--medium u-color--chestnut u-textDecoration--underlineOnHover" onClick={() => this.handleDeleteClick(snapshot)}>Delete</span>
          </div>
        }
      </div>
    )
  }
}

export default AppShanpshotRow;