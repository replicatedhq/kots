import React from "react";
import classNames from "classnames";
import Markdown from "react-remarkable";
import size from "lodash/size";

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
          if (row.isWarn) {
            icon = "preflightCheckWarning--icon";
            rowClass = "warn";
          } else if (row.isFail) {
            icon = "preflightCheckError--icon";
            rowClass = "fail";
          } else {
            icon = "preflightCheckPass--icon";
          }
          return (
            <div
              key={idx}
              className={classNames(
                "flex justifyContent--space-between preflight-check-row",
                rowClass,
              )}
            >
              <div
                className={classNames(
                  "flex-auto icon",
                  icon,
                  "u-marginRight--10",
                )}
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
                        <span
                          style={{ top: "2px", marginLeft: "2px" }}
                          className="icon external-link-icon u-cursor--pointer"
                        />
                      </a>
                    </div>
                  )}
                </div>
                {row.isFail && row.strict ? (
                  <div className="flex flex-auto alignItems--center">
                    <p className="u-textColor--error u-fontSize--small u-fontWeight--medium">
                      To deploy the application, this check cannot fail.
                    </p>
                  </div>
                ) : null}
              </div>
            </div>
          );
        })
      )}
    </div>
  );
}
