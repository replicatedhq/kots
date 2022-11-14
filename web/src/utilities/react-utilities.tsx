import React, { useEffect } from "react"
import { useHistory, useRouteMatch } from "react-router";
import { KotsParams } from "@types";
// eslint-disable-next-line
let historyRef: {
  location: {
    pathname: string
  }
};


// eslint-disable-next-line
let matchRef: any;

const RouterWrapper = () => {
    const history = useHistory();
    const match = useRouteMatch<KotsParams>();
    useEffect(() => {
      historyRef = history;
      matchRef = match;
    }, [match?.path, history?.location?.pathname]);
    return <></>;
}

// const getHistory = () => matchRef?.path;
const getHistory = () => historyRef?.location?.pathname;

export { getHistory, RouterWrapper };