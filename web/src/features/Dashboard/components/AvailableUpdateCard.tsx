import { useNavigate } from "react-router-dom";
import ReactTooltip from "react-tooltip";

import Icon from "@components/Icon";
import { Utilities } from "@src/utilities/utilities";
import { AvailableUpdate } from "@types";

const AvailableUpdateCard = ({
  updates,
  showReleaseNotes,
  upgradeService,
  appSlug,
}: {
  updates: AvailableUpdate[];
  showReleaseNotes: (releaseNotes: string) => void;
  upgradeService: {
    versionLabel?: string;
    isLoading?: boolean;
    error?: string;
  } | null;
  appSlug: string;
}) => {
  const navigate = useNavigate();
  const update = updates[0];
  const isCurrentVersionLoading =
    upgradeService?.versionLabel === update.versionLabel &&
    upgradeService.isLoading;
  return (
    <div className="tw-mt-4">
      <div className="flex justifyContent--spaceBetween alignItems--center u-marginBottom--15">
        <p className="u-fontSize--normal u-fontWeight--medium card-title">
          Latest Available Update
        </p>
        <p className="u-fontSize--normal">
          {updates && updates.length > 0 && (
            <span className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy">
              {updates.length} available
            </span>
          )}
        </p>
      </div>
      {updates && updates.length > 0 && (
        <div className="tw-flex tw-flex-col tw-gap-2 tw-max-h-[275px] tw-overflow-auto">
          <div className="available-update-row">
            <div className="tw-h-10 tw-bg-white tw-p-4 tw-flex tw-justify-between tw-items-center tw-rounded">
              <div className="flex-column">
                <div className="flex alignItems--center">
                  <p className="u-fontSize--header2 u-fontWeight--bold u-lineHeight--medium card-item-title ">
                    {update.versionLabel}
                  </p>
                  {update.isRequired && (
                    <span className="status-tag required u-marginLeft--10">
                      {" "}
                      Required{" "}
                    </span>
                  )}
                </div>
                {update.upstreamReleasedAt && (
                  <p className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginTop--5">
                    {" "}
                    Released{" "}
                    <span className="u-fontWeight--bold">
                      {Utilities.dateFormat(
                        update.upstreamReleasedAt,
                        "MM/DD/YY @ hh:mm a z"
                      )}
                    </span>
                  </p>
                )}
              </div>
              <div className="flex alignItems--center">
                {update?.releaseNotes && (
                  <>
                    <Icon
                      icon="release-notes"
                      size={24}
                      onClick={() => showReleaseNotes(update?.releaseNotes)}
                      data-tip="View release notes"
                      className="u-marginRight--5 clickable"
                    />
                    <ReactTooltip
                      effect="solid"
                      className="replicated-tooltip"
                    />
                  </>
                )}

                <ReactTooltip effect="solid" id="disable-deployment-tooltip" />

                <button
                  className={"btn tw-ml-2 primary blue"}
                  onClick={() => navigate(`/app/${appSlug}/version-history`)}
                  disabled={!update.isDeployable || isCurrentVersionLoading}
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
              </div>
            </div>
            {upgradeService?.error &&
              upgradeService?.versionLabel === update.versionLabel && (
                <div className="tw-my-4">
                  <span className="u-fontSize--small u-textColor--error u-fontWeight--bold">
                    {upgradeService.error}
                  </span>
                </div>
              )}
          </div>
        </div>
      )}
    </div>
  );
};

export default AvailableUpdateCard;
