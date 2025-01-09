import { Component } from "react";
import ReactTooltip from "react-tooltip";
import dayjs from "dayjs";
import isSameOrAfter from "dayjs/plugin/isSameOrAfter";
dayjs.extend(isSameOrAfter);

import { Utilities } from "../../utilities/utilities";
import Icon from "../Icon";
import { withRouter } from "@src/utilities/react-router-utilities";

class SnapshotRow extends Component {
  handleDeleteClick = (e, snapshot) => {
    e.stopPropagation();
    this.props.toggleConfirmDeleteModal(snapshot);
  };

  handleRestoreClick = (e, snapshot) => {
    e.stopPropagation();
    this.props.toggleRestoreModal(snapshot);
  };

  handleSnapshotClick = () => {
    const { app, snapshot } = this.props;
    const isExpired = dayjs(new Date()).isSameOrAfter(snapshot?.expiresAt);
    if (!isExpired && snapshot?.status !== "Deleting") {
      if (app) {
        this.props.navigate(
          `/snapshots/partial/${this.props.app.slug}/${snapshot?.name}`
        );
      } else {
        this.props.navigate(`/snapshots/details/${snapshot?.name}`);
      }
    }
  };

  render() {
    const { snapshot, app, hideRestore } = this.props;
    const isExpired = dayjs(new Date()).isSameOrAfter(snapshot?.expiresAt);

    return (
      <div
        className={`flex flex-auto SnapshotRow--wrapper card-item alignItems--center u-padding--15 u-marginTop--10 clickable ${
          snapshot?.status === "Deleting" && "is-deleting"
        } ${snapshot?.status === "InProgress" && "in-progress"} ${
          isExpired && "is-expired"
        }`}
        onClick={() => this.handleSnapshotClick()}
      >
        <div className="flex-column flex1" style={{ maxWidth: "700px" }}>
          <p
            className={`u-fontSize--largest ${
              isExpired || snapshot?.status === "Deleting"
                ? "u-textColor--bodyCopy"
                : "card-item-title"
            } u-lineHeight--normal u-fontWeight--bold u-marginRight--10`}
          >
            {snapshot?.name}
          </p>
          <div className="flex flex1 alignItems--center u-marginTop--10">
            <p className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium u-lineHeight--normal u-marginRight--20">
              {snapshot?.startedAt
                ? Utilities.dateFormat(
                    snapshot?.startedAt,
                    "MMM D YYYY @ hh:mm a z"
                  )
                : "n/a"}
            </p>
          </div>
        </div>
        <div className="flex flex1">
          <div className="flex flex-auto alignItems--center u-marginTop--5">
            <div className="flex flex1 flex-column">
              <div
                className="flex justifyContent--flexStart"
                style={{ gap: "60px" }}
              >
                {snapshot?.volumeSizeHuman && (
                  <p className="u-fontSize--normal u-textColor--accent u-fontWeight--bold u-lineHeight--normal justifyContent--center flex alignItems--center">
                    <span className="icon snapshot-volume-size-icon" />{" "}
                    {snapshot?.volumeSizeHuman}{" "}
                  </p>
                )}
              </div>
              {snapshot?.status === "Completed" ? (
                <p className="u-fontSize--small u-textColor--bodyCopy u-fontWeight--medium u-lineHeight--normal u-marginTop--10 u-marginRight--20">
                  <span
                    className={`status-indicator u-marginRight--5 ${snapshot?.status.toLowerCase()}`}
                  >
                    {Utilities.snapshotStatusToDisplayName(snapshot?.status)}
                  </span>
                  on{" "}
                  {snapshot?.finishedAt
                    ? snapshot?.finishedAt
                      ? Utilities.dateFormat(
                          snapshot?.finishedAt,
                          "MMM D YYYY @ hh:mm a z"
                        )
                      : "TBD"
                    : "n/a"}
                </p>
              ) : (
                <span
                  className={`status-indicator u-marginTop--10 u-marginRight--5 ${snapshot?.status.toLowerCase()}`}
                >
                  {Utilities.snapshotStatusToDisplayName(snapshot?.status)}
                </span>
              )}
            </div>
          </div>
        </div>
        {!isExpired && snapshot?.status !== "Deleting" && (
          <div className="flex flex-auto">
            {snapshot?.status === "Completed" && !hideRestore && (
              <div className="flex">
                <Icon
                  icon="sync"
                  size={20}
                  className="clickable"
                  onClick={(e) => this.handleRestoreClick(e, snapshot)}
                  data-tip="Restore from this backup"
                />
                <ReactTooltip effect="solid" className="replicated-tooltip" />
              </div>
            )}
            {snapshot?.status !== "InProgress" && (
              <Icon
                icon="trash"
                size={20}
                className="clickable u-marginLeft--20 error-color"
                onClick={(e) => this.handleDeleteClick(e, snapshot)}
              />
            )}
          </div>
        )}
      </div>
    );
  }
}

export default withRouter(SnapshotRow);
