import React from "react";
import classNames from "classnames";
import PropTypes from "prop-types";
import { Link } from "react-router-dom";

import "@src/scss/components/shared/SideBar.scss";

function SideBar(props) {
  const { className, watches, currentWatch } = props;

  return (
    <div className={classNames("sidebar u-minHeight--full", className)}>
      <div className="flex-column u-width--full">
        {watches?.map( (watch, idx) => {
          const { watchIcon, watchName, pendingVersions } = watch;
          const [ owner, slug ] = watch.slug.split("/");

          const isBehind = pendingVersions.length && pendingVersions.length > 2
              ? "2+"
              : pendingVersions.length;

          return (
            <div key={idx} className={classNames("sidebar-link", {
              selected: currentWatch === watchName
            })}>
              <Link
                className="flex"
                to={`/watch/${owner}/${slug}`}>
                  <img className="sidebar-link-icon" src={watchIcon} />
                  <div className="flex-column">
                    <p className="u-color--tuna u-fontWeight--bold u-marginBottom--5">{watch.watchName}</p>
                    <div className="flex alignItems--center">
                      <div className={classNames("icon", {
                        "checkmark-icon": !isBehind,
                        "exclamationMark-icon": isBehind
                      })}
                      />
                    <span className={classNames("u-marginLeft--5 u-fontSize--small u-fontWeight--bold", {
                      "u-color--dustyGray": !isBehind,
                      "u-color--orange": isBehind
                    })}>
                        { isBehind
                            ? `${isBehind} ${isBehind >= 2 || typeof isBehind === 'string' ? "versions" : "version"} behind`
                            : "Up to date"
                        }
                      </span>
                    </div>
                  </div>
              </Link>
            </div>
          );
        })}
      </div>
    </div>
  );
}

SideBar.displayName = "SideBar";

SideBar.propTypes = {
  className: PropTypes.string,
  currentWatch: PropTypes.string
};
export default SideBar;
