import { ComponentMeta } from "@storybook/react";

import ShowAllModal from "@src/components/modals/ShowAllModal";
import dayjs from "dayjs";

export default {
  title: "Example/ShowAllModal",
  component: ShowAllModal,
} as ComponentMeta<typeof ShowAllModal>;

export const Volumes = () => {
  const renderShowAllVolumes = () => {
    const volumes = [
      {
        completionPercent: 100,
        doneBytesHuman: "582B",
        finishedAt: "2022-12-03T00:00:21Z",
        name: "snapshot1",
        phase: "Completed",
        sizeBytesHuman: "582B",
        startedAt: "2022-12-03T00:00:20Z",
        timeRemainingSeconds: 0,
      },
      {
        completionPercent: 100,
        doneBytesHuman: "582B",
        finishedAt: "2022-12-03T00:00:21Z",
        name: "snapshot2",
        phase: "Completed",
        sizeBytesHuman: "582B",
        startedAt: "2022-12-03T00:00:20Z",
        timeRemainingSeconds: 0,
      },
      {
        completionPercent: 100,
        doneBytesHuman: "582B",
        finishedAt: "2022-12-03T00:00:21Z",
        name: "snapshot3",
        phase: "Completed",
        sizeBytesHuman: "582B",
        startedAt: "2022-12-03T00:00:20Z",
        timeRemainingSeconds: 0,
      },
    ];
    return volumes.map((volume) => {
      const diffMinutes = dayjs(volume?.finishedAt).diff(
        dayjs(volume?.startedAt),
        "minutes"
      );
      return (
        <div
          className="flex flex1 u-borderBottom--gray alignItems--center"
          key={volume.name}
        >
          <div className="flex1 u-paddingBottom--15 u-paddingTop--15 u-paddingLeft--10">
            <p className="flex1 u-fontSize--large u-textColor--primary u-fontWeight--bold u-lineHeight--bold u-marginBottom--8">
              {volume.name}
            </p>
            <p className="u-fontSize--normal u-textColor--accent u-fontWeight--bold u-lineHeight--normal u-marginRight--20">
              Size:
              <span className="u-fontWeight--normal u-textColor--bodyCopy">
                {" "}
                {volume.doneBytesHuman}/{volume.sizeBytesHuman}{" "}
              </span>
            </p>
          </div>
          <div className="flex flex-column justifyContent--flexEnd">
            <p className="u-fontSize--small u-fontWeight--normal alignSelf--flexEnd u-marginBottom--8">
              <span
                className={`status-indicator ${volume?.phase?.toLowerCase()} u-marginLeft--5`}
              >
                {volume.phase}
              </span>
            </p>
            <p className="u-fontSize--small u-fontWeight--normal">
              {" "}
              Finished in{" "}
              {diffMinutes === 0
                ? "less than a minute"
                : `${diffMinutes} minutes`}{" "}
            </p>
          </div>
        </div>
      );
    });
  };
  return (
    <div>
      <p>will show if 3 or more volumes</p>
      <ShowAllModal
        displayShowAllModal={true}
        toggleShowAllModal={() => alert("toggle modal")}
        dataToShow={renderShowAllVolumes()}
        name="Volumes"
      />
    </div>
  );
};

const renderShowAllScripts = () => {
  const hooks = [
    {
      finishedAt: "2022-12-03T00:00:21Z",
      name: "snapshot1",
      phase: "Completed",
      startedAt: "2022-12-03T00:00:20Z",
      errors: "errors",
      stderr: "stderr",
      stdout: "stdout",
      command: "command",
      podName: "podName",
    },
    {
      finishedAt: "2022-12-03T00:00:21Z",
      name: "snapshot2",
      phase: "Completed",
      startedAt: "2022-12-03T00:00:20Z",
      errors: "errors",
      stderr: "stderr",
      stdout: "stdout",
      command: "command",
      podName: "podName",
    },
    {
      finishedAt: "2022-12-03T00:00:21Z",
      name: "snapshot3",
      phase: "Completed",
      startedAt: "2022-12-03T00:00:20Z",
      errors: "errors",
      stderr: "stderr",
      stdout: "stdout",
      command: "command",
      podName: "podName",
    },
  ];
  return hooks.map((hook, i) => {
    const diffMinutes = dayjs(hook?.finishedAt).diff(
      dayjs(hook?.startedAt),
      "minutes"
    );
    return (
      <div
        className="flex flex1 u-borderBottom--gray alignItems--center"
        key={`${hook.name}-${hook.phase}-${i}`}
      >
        <div className="flex flex1 u-paddingBottom--15 u-paddingTop--15 u-paddingLeft--10">
          <div className="flex flex-column">
            <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold u-lineHeight--bold u-marginBottom--8">
              {hook.name}{" "}
              <span className="u-fontSize--small u-fontWeight--medium u-textColor--bodyCopy u-marginLeft--5">
                Pod: {hook.podName}{" "}
              </span>{" "}
            </p>
            <span className="u-fontSize--small u-fontWeight--normal u-textColor--bodyCopy u-marginRight--10">
              {" "}
              {hook.command}{" "}
            </span>
          </div>
        </div>
        <div className="flex flex-column justifyContent--flexEnd">
          <p className="u-fontSize--small u-fontWeight--normal alignSelf--flexEnd u-marginBottom--8">
            <span
              className={`status-indicator ${
                hook.errors ? "failed" : "completed"
              } u-marginLeft--5`}
            >
              {hook.errors ? "Failed" : "Completed"}
            </span>
          </p>
          {!hook.errors && (
            <p className="u-fontSize--small u-fontWeight--normal u-marginBottom--8">
              {" "}
              Finished in{" "}
              {diffMinutes === 0
                ? "less than a minute"
                : `${diffMinutes} minutes`}{" "}
            </p>
          )}
          {hook.stderr !== "" ||
            (hook.stdout !== "" && (
              <span
                className="link u-fontSize--small alignSelf--flexEnd"
                onClick={() => console.log("toggleScriptsOutput")}
              >
                {" "}
                View output{" "}
              </span>
            ))}
        </div>
      </div>
    );
  });
};
export const PreSnapshotScripts = () => {
  return (
    <div>
      <p> will show if 3 or more pre snapshot scripts </p>
      <ShowAllModal
        displayShowAllModal={true}
        toggleShowAllModal={() => alert("toggle modal")}
        dataToShow={renderShowAllScripts()}
        name="Pre-snapshot scripts"
      />
    </div>
  );
};
export const PostSnapshotScripts = () => {
  return (
    <div>
      <p> will show if 3 or more post snapshot scripts </p>
      <ShowAllModal
        displayShowAllModal={true}
        toggleShowAllModal={() => alert("toggle modal")}
        dataToShow={renderShowAllScripts()}
        name="Post-snapshot scripts"
      />
    </div>
  );
};

export const Warning = () => {
  const renderShowAllWarnings = (warnings: { title: string }[]) => {
    return warnings.map((warning, i) => (
      <div
        className="flex flex1 u-borderBottom--gray"
        key={`${warning.title}-${i}`}
      >
        <div className="flex1">
          <p className="u-fontSize--large u-textColor--primary u-fontWeight--bold u-lineHeight--bold u-paddingBottom--10 u-paddingTop--10 u-paddingLeft--10">
            {warning.title}
          </p>
        </div>
      </div>
    ));
  };

  return (
    <div>
      <p> will show if 3 or more Warnings </p>
      <ShowAllModal
        displayShowAllModal={true}
        toggleShowAllModal={() => alert("toggle modal")}
        dataToShow={renderShowAllWarnings([
          { title: "warning1" },
          { title: "warning2" },
          { title: "warning3" },
        ])}
        name="Warnings"
      />
    </div>
  );
};

export const Errors = () => {
  const renderShowAllErrors = (
    errors: { title: string; message: string }[]
  ) => {
    return errors.map((error, i) => (
      <div
        className="flex flex1 u-borderBottom--gray"
        key={`${error.title}-${i}`}
      >
        <div className="flex1 u-paddingBottom--10 u-paddingTop--10 u-paddingLeft--10">
          <p className="u-fontSize--large u-textColor--error u-fontWeight--bold u-lineHeight--bold u-marginBottom--8">
            {error.title}
          </p>
          <p className="u-fontSize--normal u-fontWeight--normal u-textColor--bodyCopy">
            {" "}
            {error.message}{" "}
          </p>
        </div>
      </div>
    ));
  };
  return (
    <div>
      <p> will show if 3 or more Errors </p>
      <ShowAllModal
        displayShowAllModal={true}
        toggleShowAllModal={() => alert("toggle modal")}
        dataToShow={renderShowAllErrors([
          { title: "error1", message: "message1" },
          { title: "error2", message: "message2" },
          { title: "error3", message: "message3" },
        ])}
        name="Errors"
      />
    </div>
  );
};
