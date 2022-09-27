import React from "react";
import { useParams } from "react-router-dom";

import { KotsParams } from "@types";

function makePageTitle({
  appSlug,
  pageName,
}: {
  appSlug?: string;
  pageName: string;
}): string {
  if (appSlug) {
    return `${pageName} | ${appSlug} | Admin Console`;
  }

  return `${pageName} | Admin Console`;
}


function KotsPageTitle(pageName: string, showAppSlug: boolean) {
  const { slug } = useParams<KotsParams>();

  if (app)
return (<title>{makePageTitle({ appSlug: showAppSlug ? slug: undefined, pageName })}</title>);
}

export { KotsPageTitle }