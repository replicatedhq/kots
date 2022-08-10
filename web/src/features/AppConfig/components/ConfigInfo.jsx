import React from "react";
import size from "lodash/size";
import findIndex from "lodash/findIndex";
import { Link } from "react-router-dom";

const ConfigInfo = ({ match, fromLicenseFlow, app }) => {
  if (fromLicenseFlow || app?.downstream?.gitops?.enabled) {
    return null;
  }

  let sequence;
  if (!match.params.sequence) {
    sequence = app?.currentSequence;
  } else {
    sequence = parseInt(match.params.sequence);
  }

  const currentSequence = app?.downstream?.currentVersion?.parentSequence;
  const pendingSequenceInxex = findIndex(
    app?.downstream?.pendingVersions,
    function (v) {
      return v.parentSequence == sequence;
    }
  );
  const pastSequenceIndex = findIndex(
    app?.downstream?.pastVersions,
    function (v) {
      return v.parentSequence == sequence;
    }
  );
  const pendingVersions = app?.downstream?.pendingVersions;

  if (size(pendingVersions) > 0 && currentSequence === sequence) {
    return (
      <div className="ConfigInfo current justifyContent--center">
        <p className="flex alignItems--center u-marginRight--5">
          {" "}
          <span className="icon info-icon-green flex u-marginRight--5" /> This
          is the currently deployed config. There{" "}
          {size(pendingVersions) === 1 ? "is" : "are"} {size(pendingVersions)}{" "}
          newer version{size(pendingVersions) === 1 ? "" : "s"} since this one.{" "}
        </p>
        <Link
          to={`/app/${app?.slug}/config/${pendingVersions[0].parentSequence}`}
          className="replicated-link"
        >
          {" "}
          Edit the latest config{" "}
        </Link>
      </div>
    );
  } else if (pastSequenceIndex > -1) {
    return (
      <div className="ConfigInfo older justifyContent--center">
        <p className="flex alignItems--center u-marginRight--5">
          {" "}
          <span className="icon info-warning-icon flex u-marginRight--5" /> This
          config is {pastSequenceIndex + 1} version
          {pastSequenceIndex === 0 ? "" : "s"} older than the currently deployed
          config.{" "}
        </p>
        <Link
          to={`/app/${app?.slug}/config/${currentSequence}`}
          className="replicated-link"
        >
          {" "}
          Edit the currently deployed config{" "}
        </Link>
      </div>
    );
  } else if (pendingSequenceInxex > -1) {
    const numVersionsNewer =
      app?.downstream?.pendingVersions?.length - pendingSequenceInxex;
    return (
      <div className="ConfigInfo newer justifyContent--center">
        <p className="flex alignItems--center u-marginRight--5">
          {" "}
          <span className="icon info-icon flex u-marginRight--5" /> This config
          is {numVersionsNewer} version
          {numVersionsNewer === 1 ? "" : "s"} newer than the currently deployed
          config.{" "}
        </p>
        <Link
          to={`/app/${app?.slug}/config/${currentSequence}`}
          className="replicated-link"
        >
          {" "}
          Edit the currently deployed config{" "}
        </Link>
      </div>
    );
  } else {
    return null;
  }
};

export default ConfigInfo;
