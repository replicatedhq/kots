import React from "react";

import { getClusterType } from "@src/utilities/utilities";
import "@src/scss/components/watches/WatchVersionHistory.scss";
import "@src/scss/components/watches/VersionCard.scss";

export default function WatchVersionHistory(props) {
  const { watch } = props;
  const { currentVersion, watches, pastVersions } = watch;

  return (
    <div className="flex-column u-position--relative verison-history-wrapper u-overflow--auto">
      <div className="flex alignItems--center u-borderBottom--gray u-paddingBottom--10">
        <p className="u-fontSize--header u-fontWeight--bold u-color--tuna">
          {currentVersion.title}
        </p>
        <div className="icon checkmark-icon flex-auto u-marginLeft--10 u-marginRight--5"></div>
        <p className="u-fontSize--large">Most recent version</p>
        <div className="flex flex1 justifyContent--flexEnd">
          {watches.length > 0 && (
            <div className="watch-cell">
              {watches.map(({ cluster }) => {
                return (
                  <div key={cluster.slug} className="flex justifyContent--center u-fontWeight--bold u-color--tuna">
                    <span>{getClusterType(cluster.gitOpsRef)[0] + "|"}</span>
                    <p className="u-fontSize--normal">
                      {cluster.slug}
                    </p>
                  </div>
                );
              })}
            </div>
          )}
        </div>
      </div>
      <div className="flex-column">
        {pastVersions.length > 0 && pastVersions.map( version => {
          return (
            <div 
              key={version.title}
              className="flex u-paddingTop--20 u-paddingBottom--20 u-borderBottom--gray">
              <div className="u-fontSize--larger u-color--tuna u-fontWeight--bold u-marginLeft--10">
                {version.title}
              </div>
              <div className="flex flex1 justifyContent--flexEnd">
                <div className="watch-cell">
                  <div className="flex justifyContent--center">
                    <div className="icon checkmark-icon"></div>
                  </div>
                </div>
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
