import Icon from "@components/Icon";
import MountAware from "@components/shared/MountAware";
import { AirgapUploader } from "@src/utilities/airgapUploader";
import { Utilities } from "@src/utilities/utilities";
import { AvailableUpdate } from "@types";
import ReactTooltip from "react-tooltip";

export const AvailableUpdateRow = ({
  update,
  index,
  showReleaseNotes,
  children,
  upgradeService,
}: {
  update: AvailableUpdate;
  index: number;
  showReleaseNotes: (releaseNotes: string) => void;
  children: React.ReactNode;
  upgradeService?: {
    versionLabel?: string;
    isLoading?: boolean;
    error?: string;
  } | null;
}) => {
  return (
    <div key={index} className="available-update-row">
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
              <ReactTooltip effect="solid" className="replicated-tooltip" />
            </>
          )}
          {children}
        </div>
      </div>
      {upgradeService?.error &&
        upgradeService?.versionLabel === update.versionLabel && (
          <div className="tw-my-4">
            <span className="u-fontSize--small u-textColor--error u-fontWeight--bold">
              {upgradeService.error}
              error
            </span>
          </div>
        )}
    </div>
  );
};

const AvailableUpdatesComponent = ({
  updates,
  showReleaseNotes,
  upgradeService,
  startUpgradeService,
  airgapUploader,
  isAirgap,
  fetchAvailableUpdates,
}: {
  updates: AvailableUpdate[];
  showReleaseNotes: (releaseNotes: string) => void;
  upgradeService: {
    versionLabel?: string;
    isLoading?: boolean;
    error?: string;
  } | null;
  startUpgradeService: (version: AvailableUpdate) => void;
  airgapUploader: AirgapUploader | null;
  isAirgap: boolean;
  fetchAvailableUpdates: () => void;
}) => {
  return (
    <div className="TableDiff--Wrapper card-bg u-marginBottom--30">
      <div className="flex justifyContent--spaceBetween alignItems--center u-marginBottom--15">
        <p className="u-fontSize--normal u-fontWeight--medium card-title">
          Available Updates
        </p>
        {isAirgap && airgapUploader && (
          <div className="tw-flex tw-items-center">
            <MountAware
              onMount={(el: Element) => airgapUploader?.assignElement(el)}
            >
              <div className="flex alignItems--center">
                <span className="icon clickable dashboard-card-upload-version-icon u-marginRight--5" />
                <span className="link u-fontSize--small u-lineHeight--default">
                  Upload new version
                </span>
              </div>
            </MountAware>
          </div>
        )}
        <div className="flex alignItems--center u-marginRight--20">
          <span
            className="flex-auto flex alignItems--center link u-fontSize--small"
            onClick={fetchAvailableUpdates}
          >
            <Icon
              icon="check-update"
              size={16}
              className="clickable u-marginRight--5"
              color={""}
              style={{}}
              disableFill={false}
              removeInlineStyle={false}
            />
            Check for update
          </span>
        </div>
      </div>
      {updates && updates.length > 0 ? (
        <div className="tw-flex tw-flex-col tw-gap-2 tw-max-h-[275px] tw-overflow-auto">
          {updates.map((update, index) => {
            const isCurrentVersionLoading =
              upgradeService?.versionLabel === update.versionLabel &&
              upgradeService.isLoading;
            return (
              <AvailableUpdateRow
                update={update}
                index={index}
                showReleaseNotes={showReleaseNotes}
                upgradeService={upgradeService}
              >
                <>
                  <button
                    className={"btn tw-ml-2 primary blue"}
                    onClick={() => startUpgradeService(update)}
                    disabled={!update.isDeployable || isCurrentVersionLoading}
                  >
                    <span
                      key={update.nonDeployableCause}
                      data-tip-disable={update.isDeployable}
                      data-tip={update.nonDeployableCause}
                      data-for="disable-deployment-tooltip"
                    >
                      {isCurrentVersionLoading ? "Preparing..." : "Deploy"}
                    </span>
                  </button>
                  <ReactTooltip
                    effect="solid"
                    id="disable-deployment-tooltip"
                  />
                </>
              </AvailableUpdateRow>
            );
          })}
        </div>
      ) : (
        <div className="card-item flex-column flex1 u-marginTop--20 u-marginBottom--10 alignItems--center justifyContent--center">
          <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-padding--10">
            Application up to date.
          </p>
        </div>
      )}
    </div>
  );
};

export default AvailableUpdatesComponent;
