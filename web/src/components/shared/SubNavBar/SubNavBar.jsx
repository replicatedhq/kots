import React from "react";
import classNames from "classnames";
import PropTypes from "prop-types";
import { Link } from "react-router-dom";

import { isHelmChart } from "@src/utilities/utilities";
import subNavConfig from "@src/config-ui/subNavConfig";

export default function SubNavBar(props) {
  const { className, activeTab, watch } = props;
  let { slug } = watch;

  if(isHelmChart(watch)) {
    slug = `helm/${watch.id}`;
  }

  return (
    <div className={classNames("details-subnav", className)}>
      <ul>
        {subNavConfig.map( (link, idx) => {
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
            return link.displayRule(watch || {}) && generatedMenuItem;
          }
          return generatedMenuItem;
        }).filter(Boolean)}
      </ul>
    </div>
  );
}

SubNavBar.defaultProps = {
  watch: {}
};

SubNavBar.propTypes = {
  className: PropTypes.string,
  activeTab: PropTypes.string,
  slug: PropTypes.string,
  watch: PropTypes.object
};
