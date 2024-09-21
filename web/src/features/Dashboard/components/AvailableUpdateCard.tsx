import { useNavigate } from "react-router-dom";

import { AvailableUpdate } from "@types";
import { AvailableUpdateRow } from "@components/apps/AvailableUpdatesComponent";

const AvailableUpdateCard = ({
  updates,
  showReleaseNotes,
  appSlug,
}: {
  updates: AvailableUpdate[];
  showReleaseNotes: (releaseNotes: string) => void;
  appSlug: string;
}) => {
  const navigate = useNavigate();
  const update = updates[0];
  return (
    <div className="tw-mt-4">
      <div className="flex alignItems--center u-marginBottom--15">
        <p className="u-fontSize--normal u-fontWeight--medium card-title">
          Latest Available Update
        </p>
        <p className="u-fontSize--normal">
          <span className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy tw-ml-2">
            ({updates.length} available)
          </span>
        </p>
      </div>
      <div className="tw-flex tw-flex-col tw-gap-2 tw-max-h-[275px] tw-overflow-auto">
        <AvailableUpdateRow
          update={update}
          showReleaseNotes={showReleaseNotes}
          index={1}
        >
          <button
            className={"btn tw-ml-2 primary blue"}
            onClick={() => navigate(`/app/${appSlug}/version-history`)}
          >
            <span
              key={update.nonDeployableCause}
              data-tip-disable={update.isDeployable}
              data-tip={update.nonDeployableCause}
              data-for="disable-deployment-tooltip"
            >
              Go to Version history
            </span>
          </button>
        </AvailableUpdateRow>
      </div>
    </div>
  );
};

export default AvailableUpdateCard;
