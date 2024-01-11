import { ChangeEvent } from "react";

const ConfigureGraphs = ({
  toggleConfigureGraphs,
  updatePromValue,
  promValue,
  savingPromValue,
  savingPromError,
  onPromValueChange,
  placeholder,
}: {
  toggleConfigureGraphs?: () => void;
  updatePromValue?: () => void;
  promValue?: string;
  savingPromValue?: boolean;
  savingPromError?: string;
  onPromValueChange?: (e: ChangeEvent<HTMLInputElement>) => void;
  placeholder?: string;
}) => {
  return (
    <div
      className={`${
        toggleConfigureGraphs ? "Modal-body" : "ConfigureGraphs--wrapper"
      } flex-column flex1`}
    >
      {toggleConfigureGraphs && (
        <h2 className="u-fontSize--largest u-fontWeight--bold u-textColor--primary u-marginBottom--10">
          Configure graphs
        </h2>
      )}
      <p className="u-fontSize--normal u-textColor--bodyCopy u-lineHeight--normal u-marginBottom--20">
        To see graphs and metrics, provide the address of your Prometheus
        installation.
        <br />
        This must be resolvable from the Admin Console installation.
      </p>
      <h3 className="u-fontSize--normal u-fontWeight--bold u-textColor--primary u-marginBottom--10">
        Prometheus endpoint
      </h3>
      <div className="EditWatchForm flex-column">
        <div className="flex alignItems--center">
          <input
            type="text"
            className="Input u-marginRight--10"
            placeholder={
              placeholder ||
              "https://prometheus-k8s.default.svc.cluster.local:9090"
            }
            value={promValue}
            onChange={onPromValueChange}
          />
          <button
            disabled={savingPromValue}
            onClick={updatePromValue}
            className="btn secondary blue"
          >
            {savingPromValue ? "Saving" : "Save"}
          </button>
        </div>
        <div className="flex u-marginTop--10">
          {savingPromError && (
            <span className="u-textColor--error u-fontSize--normal u-marginRight--10 u-fontWeight--bold">
              {savingPromError}
            </span>
          )}
          {toggleConfigureGraphs && (
            <button
              type="button"
              onClick={toggleConfigureGraphs}
              className="btn secondary force-gray u-marginRight--20"
            >
              Cancel
            </button>
          )}
        </div>
      </div>
    </div>
  );
};

export default ConfigureGraphs;
