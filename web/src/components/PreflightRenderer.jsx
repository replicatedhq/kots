import React from "react";
import classNames from "classnames";
import Markdown from "react-remarkable";
import size from "lodash/size";
import Icon from "./Icon";

export default function PreflightRenderer(props) {
  const { className, results, skipped } = props;

  let preflightJSON = {};
  if (results && !skipped) {
    preflightJSON = JSON.parse(results);
  }

  const noResults = size(preflightJSON?.results) === 0;

  return (
    <div className={className}>
      {skipped ? (
        <p className="u-fontWeight--medium u-lineHeight--more u-marginTop--5 u-marginBottom--10">
          Preflight checks for this version did not run.
        </p>
      ) : noResults ? (
        <p className="u-fontWeight--medium u-lineHeight--more u-marginTop--5 u-marginBottom--10">
          This application does not have any preflight checks.
        </p>
      ) : (
        preflightJSON?.results.map((row, idx) => {
          let icon;
          let rowClass;
          let iconClass;
          if (row.isWarn) {
            icon = "warning";
            iconClass = "warning-color";
            rowClass = "u-textColor--warning";
          } else if (row.isFail) {
            icon = "warning-circle-filled";
            iconClass = "error-color";
            rowClass = "u-textColor--error";
          } else {
            icon = "check-circle-filled";
            iconClass = "success-color";
          }
          return (
            <div
              key={idx}
              className={classNames(
                "flex justifyContent--space-between preflight-check-row",
                rowClass
              )}
            >
              <Icon
                icon={icon}
                size={18}
                className={`${iconClass} flex-auto u-marginRight--10`}
              />
              <div className="flex flex1">
                <div className="flex1">
                  <p className="u-textColor--primary u-fontSize--large u-fontWeight--bold">
                    {row.title}
                  </p>
                  <div className="PreflightMessageRow u-marginTop--10">
                    <Markdown source={row.message} />
                  </div>
                  {row.uri && (
                    <div className="u-marginTop--5">
                      <a
                        href={row.uri}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="replicated-link u-fontSize--small u-fontWeight--medium u-position--relative"
                      >
                        {" "}
                        Learn more{" "}
                        <Icon
                          icon="external-page"
                          size={13}
                          className="clickable"
                          style={{ top: "2px", marginLeft: "2px" }}
                        />
                      </a>
                    </div>
                  )}
                  {row.isFail && row.strict ? (
                    <p className="u-textColor--error u-fontSize--small u-fontWeight--medium u-marginTop--10">
                      To deploy the application, this check cannot fail.
                    </p>
                  ) : null}
                </div>
              </div>
            </div>
          );
        })
      )}
    </div>
  );
}
