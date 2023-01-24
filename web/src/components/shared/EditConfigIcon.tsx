import { useSelectedApp } from "@features/App";
import { Version } from "@types";
import React from "react";
import { Link } from "react-router-dom";
import { useIsHelmManaged } from "@src/components//hooks";
import Icon from "@components/Icon";
import ReactTooltip from "react-tooltip";

const EditConfigIcon = ({
  version,
  isPending = false,
}: {
  version: Version | null;
  isPending: boolean;
}) => {
  const { data: isHelmManagedResponse } = useIsHelmManaged();
  const { isHelmManaged = false } = isHelmManagedResponse || {};
  const { selectedApp } = useSelectedApp();

  if (!version) {
    return null;
  }
  if (!version?.isConfigurable) {
    return null;
  }
  if (version.status === "pending_download") {
    return null;
  }
  if (version.status === "pending_config") {
    // action button will already be set to "Configure", no need to show edit config icon as well
    return null;
  }

  let url = `/app/${selectedApp?.slug}/config/${version.sequence}`;
  if (isHelmManaged) {
    url = `${url}?isPending=${isPending}&semver=${version.versionLabel}`;
  }

  return (
    <div className="u-marginLeft--10">
      <Link to={url} data-tip="Edit config">
        <Icon icon="edit-config" size={22} />
      </Link>
      <ReactTooltip effect="solid" className="replicated-tooltip" />
    </div>
  );
};

export default EditConfigIcon;
