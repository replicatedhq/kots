import React from "react";
import classNames from "classnames";
import PropTypes from "prop-types";
import { Link } from "react-router-dom";

import { isHelmChart } from "@src/utilities/utilities";
import subNavConfig from "@src/config-ui/subNavConfig";


export default function SubNavBar(props) {
  const { className, activeTab, app, isVeleroInstalled, isAccess, isSnapshots } = props;
  let { slug } = app;

  if (isHelmChart(app)) {
    slug = `helm/${app.id}`;
  }

  let configSequence = app?.downstreams[0]?.currentVersion?.parentSequence;
  let kotsSequence = app?.downstreams[0]?.currentVersion?.parentSequence;

  // file view always shows top version on the list
  // config view always shows the deployed version, falling back to the top version if nothing is deployed
  if (app?.downstreams[0]?.pendingVersions?.length) {
    kotsSequence = app?.downstreams[0]?.pendingVersions[0]?.parentSequence;
    if (!app?.downstreams[0]?.currentVersion) {
      configSequence = app?.downstreams[0]?.pendingVersions[0]?.parentSequence;
    }
  }

  const accessConfig = [
    {
      tabName: "configure-ingress",
      displayName: "Configure Ingress",
      to: () => `/access/configure-ingress`,
    },
    {
      tabName: "identity-providers",
      displayName: "Identity Providers",
      to: () => `/access/identity-providers`,
    },
  ];

  const snapshotsConfig = [
    {
      tabName: activeTab === slug ? slug : "snapshots",
      displayName: "Full Snapshots (Instance)",
      to: () => `/snapshots`,
    },
    {
      tabName: "partial",
      displayName: "Partial Snapshots (Application)",
      to: (slug) => `/snapshots/partial/${slug}`,
    },
    {
      tabName: "settings",
      displayName: "Settings & Schedule",
      to: () => `/snapshots/settings`,
    },
  ]

  return (
    <div className={classNames("details-subnav", className)}>
      <ul>
        {isAccess ?
          accessConfig.map((link, idx) => {
            const generatedMenuItem = (
              <li
                key={idx}
                className={classNames({
                  "is-active": activeTab === link.tabName
                })}>
                <Link to={link.to()}>
                  {link.displayName}
                </Link>
              </li>
            );
            return generatedMenuItem;
          }).filter(Boolean)
          :
          isSnapshots ?
            snapshotsConfig.map((link, idx) => {
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
              return generatedMenuItem;
            }).filter(Boolean)
            :
            subNavConfig.map((link, idx) => {
              let hasBadge = false;
              if (link.hasBadge) {
                hasBadge = link.hasBadge(app || {});
              }
              const generatedMenuItem = (
                <li
                  key={idx}
                  className={classNames({
                    "is-active": activeTab === link.tabName
                  })}>
                  <Link to={link.to(slug, kotsSequence, configSequence)}>
                    {link.displayName} {hasBadge && <span className="subnav-badge" />}
                  </Link>
                </li>
              );
              if (link.displayRule) {
                return link.displayRule(app || {}, isVeleroInstalled, app.isAppIdentityServiceSupported) && generatedMenuItem;
              }
              return generatedMenuItem;
            }).filter(Boolean)}
      </ul>
    </div>
  );
}

SubNavBar.defaultProps = {
  app: {}
};

SubNavBar.propTypes = {
  className: PropTypes.string,
  activeTab: PropTypes.string,
  slug: PropTypes.string,
  app: PropTypes.object
};
