import { useParams } from "react-router-dom";
import { Helmet } from "react-helmet";

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

/*
  example output:
  Troubleshoot | http-echo | Admin Console
  GitOps | Admin Console
*/
function KotsPageTitle({
  pageName,
  showAppSlug = false,
}: {
  pageName: string;
  showAppSlug?: boolean;
}) {
  const { slug } = useParams<KotsParams>();

  if (slug && showAppSlug) {
    return (
      <Helmet>
        <title>{makePageTitle({ appSlug: slug, pageName })}</title>
      </Helmet>
    );
  }

  return (
    <Helmet>
      <title>{makePageTitle({ pageName })}</title>
    </Helmet>
  );
}

export { KotsPageTitle };
