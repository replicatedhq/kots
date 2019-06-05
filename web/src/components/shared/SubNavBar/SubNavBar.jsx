import React from "react";
import classNames from "classnames";
import PropTypes from "prop-types";
import { Link } from "react-router-dom";

import navConfig from "@src/config-ui/navConfig";

export default function SubNavBar(props) {
  const { className, activeTab, slug, watch } = props;
  
  return (
    <div className={classNames("details-subnav", className)}>
      <ul>
        {navConfig.map( (link, idx) => {
          const generatedMenuItem = (
            <li
              key={idx}
              className={classNames({
                "is-active": activeTab === link.tabName
              })}>
              <Link to={link.to(slug)}>
                {link.displayName}
              </Link>
            </li>
          );
          if (link.displayRule) {
            return link.displayRule(watch) && generatedMenuItem;
          }
          return generatedMenuItem;
        }).filter(Boolean)}
      </ul>
    </div>
  );
}

SubNavBar.propTypes = {
  className: PropTypes.string,
  activeTab: PropTypes.string,
  slug: PropTypes.string
};

/*
Reference JSX
<div className="details-subnav flex flex u-marginBottom--30">
  <ul>
    {!watch.cluster && (
      <li className={classNames({ "is-active": !match.params.tab})}>
        <Link to={`/watch/${slug}`}>
          Application
        </Link>
      </li>
    )}
    {!watch.cluster && (
      <li className={classNames({ "is-active": match.params.tab === "deployment-clusters" })}>
        <Link to={`/watch/${slug}/deployment-clusters`}>
          Deployment clusters
        </Link>
      </li>
    )}
    <li className={`${match.params.tab === "integrations" ? "is-active" : ""}`}>
      <Link to={`/watch/${slug}/integrations`}>
        Integrations
      </Link>
    </li>
    <li className={`${match.params.tab === "state" ? "is-active" : ""}`}>
      <Link to={`/watch/${slug}/state`}>
        State JSON
      </Link>
    </li>
  </ul>
</div>
*/
