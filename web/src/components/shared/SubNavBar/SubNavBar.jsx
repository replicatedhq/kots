import React from "react";
import classNames from "classnames";
import PropTypes from "prop-types";
import { Link } from "react-router-dom";

import navConfig from "@src/config-ui/subNavConfig";

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
  slug: PropTypes.string,
  watch: PropTypes.object
};
