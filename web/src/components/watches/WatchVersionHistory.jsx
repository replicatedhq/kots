import React from "react";
import Helmet from "react-helmet";
import dayjs from "dayjs";
import relativeTime from "dayjs/plugin/relativeTime";
import Loader from "../shared/Loader";
import ActiveDownstreamVersionRow from "./ActiveDownstreamVersionRow";

import "@src/scss/components/watches/WatchVersionHistory.scss";
dayjs.extend(relativeTime);

export default function WatchVersionHistory(props) {
  const { watch, match, checkingForUpdates, checkingUpdateText, errorCheckingUpdate, handleAddNewCluster } = props;
  
  if (!watch) {
    return null;
  }

  const {
    currentVersion,
    watchIcon,
    watches,
    watchName,
  } = watch;

  let updateText = <p className="u-marginTop--10 u-fontSize--small u-color--dustyGray u-fontWeight--medium">Last checked {dayjs(watch.lastUpdateCheck).fromNow()}</p>;
  if (errorCheckingUpdate) {
    updateText = <p className="u-marginTop--10 u-fontSize--small u-color--chestnut u-fontWeight--medium">Error checking for updates, please try again</p>
  } else if (checkingForUpdates) {
    updateText = <p className="u-marginTop--10 u-fontSize--small u-color--dustyGray u-fontWeight--medium">{checkingUpdateText}</p>
  }
              

  return (
    <div className="flex-column flex1 u-position--relative u-overflow--auto u-padding--20">
      <Helmet>
        <title>{`${watch.watchName} Version History`}</title>
      </Helmet>
      <div className="flex flex-auto alignItems--center justifyContent--center u-marginTop--10 u-marginBottom--30">
        <div className="upstream-version-box-wrapper flex">
          <div className="flex flex1">
            {watchIcon &&
              <div className="flex-auto u-marginRight--10">
                <div className="watch-icon" style={{ backgroundImage: `url(${watchIcon})` }}></div>
              </div>
            }
            <div className="flex1 flex-column">
              <p className="u-fontSize--34 u-fontWeight--bold u-color--tuna">
                {currentVersion ? currentVersion.title : "---"}
              </p>
              <p className="u-fontSize--large u-fontWeight--medium u-marginTop--5 u-color--nevada">{currentVersion ? "Current upstream version" : "No deployments have been made"}</p>
              {currentVersion?.deployedAt && <p className="u-fontSize--normal u-fontWeight--medium u-marginTop--5 u-color--dustyGray">Released on {dayjs(currentVersion.deployedAt).format("MMMM D, YYYY")}</p>}
            </div>
          </div>
          {!watch.cluster &&
            <div className="flex-auto flex-column alignItems--center justifyContent--center">
              {checkingForUpdates ?
                <Loader size="32" />
              :
                <button className="btn secondary green" onClick={props.onCheckForUpdates}>Check for updates</button>
              }
              {updateText}
            </div>
          }
        </div>
      </div>
      <div className="flex-column flex1 u-overflow--hidden">
        <p className="flex-auto u-fontSize--larger u-fontWeight--bold u-color--tuna u-paddingBottom--10">Active downstream versions</p>
        <div className="flex1 u-overflow--auto">
          {watches?.length ? watches.map((watch) => (
            <ActiveDownstreamVersionRow key={watch.cluster.slug} watch={watch} match={match} />
          ))
          :
          <div className="flex-column flex1">
            <div className="EmptyState--wrapper flex-column flex1">
              <div className="EmptyState flex-column flex1 alignItems--center justifyContent--center">
                <div className="flex alignItems--center justifyContent--center">
                  <span className="icon ship-complete-icon-gh"></span>
                  <span className="deployment-or-text">OR</span>
                  <span className="icon ship-medium-size"></span>
                </div>
                <div className="u-textAlign--center u-marginTop--10">
                  <p className="u-fontSize--largest u-color--tuna u-lineHeight--medium u-fontWeight--bold u-marginBottom--10">No active downstreams</p>
                  <p className="u-fontSize--normal u-color--dustyGray u-lineHeight--medium u-fontWeight--medium">{watchName} has no downstream deployment clusters yet. {watchName} must be deployed to a cluster to get version histories.</p>
                </div>
                <div className="u-marginTop--20">
                  <button className="btn secondary" onClick={handleAddNewCluster}>Add a deployment cluster</button>
                </div>
              </div>
            </div>
          </div>
          }
        </div>
      </div>
    </div>
  );
}
