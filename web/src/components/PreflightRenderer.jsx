import React from "react";
import classNames from "classnames";
import Markdown from "react-remarkable";
import size from "lodash/size";

export default function PreflightRenderer(props) {
  const { className, results, skipped } = props;

  let preflightJSON = {}
  if (results && !skipped) {
    preflightJSON = JSON.parse(results);
  }

  const noResults = size(preflightJSON?.results) === 0;

  return (
    <div className={className}>
      <p className="u-fontSize--jumbo u-textColor--primary u-fontWeight--bold u-marginBottom--15">
        Results from your preflight checks
      </p>
      {skipped
        ?
        <p className="u-fontWeight--medium u-lineHeight--more u-marginTop--5 u-marginBottom--10">
          Preflight checks for this version did not run.
        </p>
        :
          noResults
          ?
          <p className="u-fontWeight--medium u-lineHeight--more u-marginTop--5 u-marginBottom--10">
            This application does not have any preflight checks.
          </p>
          :
          preflightJSON?.results.map((row, idx) => {
            let icon;
            if (row.isWarn) {
              icon = "exclamationMark--icon";
            } else if (row.isFail) {
              icon = "error-small";
            } else {
              icon = "checkmark-icon";
            }
            return (
              <div key={idx} className="flex justifyContent--space-between preflight-check-row">
                <div className={classNames("flex-auto icon", icon, "u-marginRight--10")} />
                <div className="flex1">
                  <p className="u-textColor--primary u-fontSize--larger u-fontWeight--bold">{row.title}</p>
                  <div className="PreflightMessageRow u-marginTop--10">
                    <Markdown source={row.message}/>
                  </div>
                </div>
                {row.uri &&
                <div className="flex-column flex justifyContent--center">
                  <a href={row.uri} target="_blank" rel="noopener noreferrer" className="btn secondary lightBlue"> Learn more </a>
                </div>}
              </div>
            );
        })}
    </div>
  )
}