import React from "react";
import classNames from "classnames";
import PropTypes from "prop-types";
import { Link } from "react-router-dom";

import "@src/scss/components/shared/SideBar.scss";

export default function SideBar(props) {
  const { className, watches, currentWatch } = props;

  return (
    <div className={classNames("sidebar u-minHeight--full", className)}>
      <div className="flex-column u-width--full">
        {watches?.map( (watch, idx) => {
          const { watchIcon, watchName } = watch;
          const [ owner, slug ] = watch.slug.split("/");
          return (
            <div key={idx} className={classNames("sidebar-link", {
              selected: currentWatch === watchName
            })}>
              <Link
                className="flex"
                to={`/watch/${owner}/${slug}`}>
                  <img className="sidebar-link-icon" src={watchIcon} />
                  <div className="flex-column u-lineHeight--medium">
                    <p className="u-color--tuna u-fontWeight--bold">{watch.watchName}</p>
                    <div className="flex">
                      <img src="http://placekitten.com/10/10" />
                      <span className="u-marginLeft--10 u-color--dustyGray u-fontSize--small">Up to date</span>
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
