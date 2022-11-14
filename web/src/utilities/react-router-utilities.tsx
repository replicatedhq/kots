import React, { useEffect } from "react";
import { match, RouterProps, useHistory, useRouteMatch } from "react-router";
import { KotsParams } from "@types";
// eslint-disable-next-line
// let historyRef: {
//   location: {
//     pathname: string
//   }
// };

// This is a hack
// TODO: remove once refactored all class compnoents
let historyRef: RouterProps["history"];

let matchRef: match<KotsParams>;

/**
 * @deprecated The method should not be used
 */
const RouterWrapper = () => {
  const history = useHistory();
  const routeMatch = useRouteMatch<KotsParams>();
  useEffect(() => {
    historyRef = history;
    matchRef = routeMatch;
  }, [routeMatch, history]);
  return <></>;
};

/**
 * @deprecated The method should not be used
 */
const getHistory = () => historyRef;

/**
 * @deprecated The method should not be used
 */
const getMatch = () => matchRef;

export { getHistory, getMatch, RouterWrapper };
