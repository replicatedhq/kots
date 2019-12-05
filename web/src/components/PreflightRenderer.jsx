import React from "react";
import classNames from "classnames";
import Markdown from "react-remarkable";

export default function PreflightRenderer(props) {
  const { className, results } = props;
  const preflightJSON = JSON.parse(results);

  return (
    <div className={className}>
      <p className="u-fontSize--jumbo u-color--tuna u-fontWeight--bold u-marginBottom--15">
        Results from your preflight checks
      </p>
      {preflightJSON?.results.map((row, idx) => {
        let icon;
        if (row.isWarn) {
          icon = "exclamationMark--icon";
        } else if (row.isFail) {
          icon = "error-small";
        } else {
          icon = "checkmark-icon";
        }
        return (
          <div key={idx} className="flex justifyContent--space-between preflight-check-row u-paddingTop--10 u-paddingBottom--10">
            <div className={classNames("flex-auto icon", icon, "u-marginRight--10")} />
            <div className="flex1">
              <p className="u-color--tuna u-fontSize--larger u-fontWeight--bold">{row.title}</p>
              <div className="PreflightMessageRow u-marginTop--5">
                <Markdown source={row.message}/>
              </div>
            </div>
            {row.uri &&
            <div className="flex-column flex justifyContent--flexEnd u-marginRight--20 alignItems--center">
              <a href={row.uri} target="_blank" rel="noopener noreferrer" className="btn secondary lightBlue"> Learn more </a>
            </div>}
          </div>
        );
      })}
    </div>
  )
}