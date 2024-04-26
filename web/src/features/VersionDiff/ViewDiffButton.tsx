import { Version } from "@types";
import { useSelectedApp } from "@features/App/hooks/useSelectedApp";

interface Props {
  onWhyNoGeneratedDiffClicked: (version: Version) => void;
  onWhyUnableToGeneratedDiffClicked: (version: Version) => void;
  onViewDiffClicked: (firstSequence: number, secondSequence: number) => void;
  version: Version;
  versionHistory: Version[];
}

interface DiffSummary {
  filesChanged?: number;
}

// TODO: unmarshal this in the fetch handler
function unmarshallDiffSummary(diffSummary: string): DiffSummary {
  try {
    return JSON.parse(diffSummary) || {};
  } catch (err) {
    throw err;
  }
}

function getPreviousSequence(versionHistory: Version[], version: Version) {
  let previousSequence = 0;
  for (const v of versionHistory) {
    if (v.status === "pending_download") {
      continue;
    }
    if (v.parentSequence < version.parentSequence) {
      previousSequence = v.parentSequence;
      break;
    }
  }
  return previousSequence;
}

function ViewDiffButton(props: Props) {
  const selectedApp = useSelectedApp();

  // TODO: flatten in selector
  const showViewDiffButton = !selectedApp?.downstream.gitops?.isConnected;
  const showDiffSummaryError =
    props.version?.diffSummaryError?.length > 0 ? true : false;
  const numberOfFilesChanged = props.version?.diffSummary
    ? unmarshallDiffSummary(props.version.diffSummary)?.filesChanged || 0
    : 0;

  return (
    <>
      {showDiffSummaryError && (
        <div className="flex flex1 alignItems--center">
          <span className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal u-textColor--bodyCopy">
            Unable to generate diff{" "}
            <span
              className="link"
              onClick={() =>
                props.onWhyUnableToGeneratedDiffClicked(props.version)
              }
            >
              Why?
            </span>
          </span>
        </div>
      )}

      {numberOfFilesChanged > 0 && (
        <div className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
          <div className="DiffSummary u-marginRight--10">
            <span className="files">{numberOfFilesChanged} files changed </span>
            {showViewDiffButton && (
              <span
                className="u-fontSize--small link u-marginLeft--5"
                onClick={() =>
                  props.onViewDiffClicked(
                    getPreviousSequence(props.versionHistory, props.version),
                    props.version.parentSequence
                  )
                }
              >
                View diff
              </span>
            )}
          </div>
        </div>
      )}
      {numberOfFilesChanged === 0 && (
        <div className="u-fontSize--small u-fontWeight--medium u-lineHeight--normal">
          <div className="DiffSummary">
            <span className="files">
              No changes to show.{" "}
              <span
                className="link"
                onClick={() => props.onWhyNoGeneratedDiffClicked(props.version)}
              >
                Why?
              </span>
            </span>
          </div>
        </div>
      )}
    </>
  );
}

export { ViewDiffButton };
