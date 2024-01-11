import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import ShowLogsModal from "@src/components/modals/ShowLogsModal";
import { useState } from "react";

export default {
  title: "Example/ShowLogsModal",
  component: ShowLogsModal,
} as ComponentMeta<typeof ShowLogsModal>;

const Template: ComponentStory<typeof ShowLogsModal> = (args) => {
  const [selectedTab, setSelectedTab] = useState("dryrunStdout");
  const renderLogsTab = () => {
    const isHelmManaged = false;
    const filterNonHelmTabs = (tab: string) => {
      if (isHelmManaged) {
        return tab.startsWith("helm");
      }
      return true;
    };

    const tabs = Object.keys(args.logs);

    return (
      <div className="flex action-tab-bar u-marginTop--10">
        {tabs
          .filter((tab) => tab !== "renderError")
          .filter((tab) => filterNonHelmTabs(tab))
          .map((tab) => (
            <div
              className={`tab-item blue ${tab === selectedTab && "is-active"}`}
              key={tab}
              onClick={() => setSelectedTab(tab)}
            >
              {tab}
            </div>
          ))}
      </div>
    );
  };

  return (
    <MemoryRouter>
      <ShowLogsModal
        {...args}
        renderLogsTabs={renderLogsTab()}
        selectedTab={selectedTab}
      />
    </MemoryRouter>
  );
};
export const ShowLogsModalExample = Template.bind({});

ShowLogsModalExample.args = {
  showLogsModal: true,
  hideLogsModal: () => {
    alert("hide modal");
  },
  // if this is true, the modal will show the error msg and not the logs
  //viewLogsErrMsg: "Error message",
  logs: {
    dryrunStdout:
      "configmap/nginx-content created (dry run)\nsecret/kotsadm-replicated-registry created (dry run)\nsecret/snapshot3-registry created (dry run)\nservice/nginx created (dry run)\ndeployment.apps/nginx created (dry run)\n",
    dryrunStderr: "",
    applyStdout:
      "configmap/nginx-content created\nsecret/kotsadm-replicated-registry created\nsecret/snapshot3-registry created\nservice/nginx created\ndeployment.apps/nginx created\n",
    applyStderr: "",
    helmStdout: "",
    helmStderr: "",
    renderError: "",
  },
  logsLoading: false,
  versionFailing: "version failing",
  troubleshootUrl: "troubleshoot.url",
};
