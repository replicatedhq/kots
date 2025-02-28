import classNames from "classnames";
// TODO: find replacement for react-remarkable
// @ts-ignore
import Markdown from "react-remarkable";
import Icon from "./Icon";

import { PreflightResult } from "@src/features/PreflightChecks/types";

interface Props {
  className?: string;
  results: PreflightResult[];
  skipped: boolean;
}
export default function PreflightRenderer(props: Props) {
  const { className, results, skipped } = props;

  const noResults = results.length === 0;

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
        results.map((row, idx) => {
          let icon;
          let rowClass;
          let iconClass;
          if (row.showWarn) {
            icon = "warning";
            iconClass = "warning-color";
            rowClass = "u-textColor--warning";
          } else if (row.showFail) {
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
                  <p className="u-textColor--primary u-fontSize--large u-fontWeight--bold" data-testid="preflight-message-title">
                    {row.title}
                  </p>
                  <div className="PreflightMessageRow u-marginTop--10" data-testid="preflight-message-row">
                    <Markdown source={row.message} />
                  </div>
                  {row.learnMoreUri && (
                    <div className="u-marginTop--5">
                      <a
                        href={row.learnMoreUri}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="link u-fontSize--small u-fontWeight--medium u-position--relative"
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
                  {row.showCannotFail && (
                    <p className="u-textColor--error u-fontSize--small u-fontWeight--medium u-marginTop--10">
                      To deploy the application, this check cannot fail.
                    </p>
                  )}
                </div>
              </div>
            </div>
          );
        })
      )}
    </div>
  );
}
