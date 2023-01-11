import React from "react";
import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import ShowLogsModal from "@src/components/modals/ShowLogsModal";

export default {
  title: "Example/ShowLogsModal",
  component: ShowLogsModal
} as ComponentMeta<typeof ShowLogsModal>;

const Template: ComponentStory<typeof ShowLogsModal> = (args) => (
  <MemoryRouter>
    <ShowLogsModal {...args} />
  </MemoryRouter>
);

export const ShowLogsModalExample = Template.bind({});

ShowLogsModalExample.args = {
  showLogsModal: true,
  hideLogsModal: () => {
    alert("hide modal");
  },
  viewLogsErrMsg: "Error message",
  //   logs: { renderError: "Error" },
  selectedTab: { name: "name", value: "value" },
  logsLoading: false,
  renderLogsTabs: () => {
    let selectedTab = "selectedTab";

    const tabs = Object.keys({ renderError: "Error" });
    return (
      <div className="flex action-tab-bar u-marginTop--10">
        {tabs
          .filter((tab) => tab !== "renderError")
          // .filter((tab) => filterNonHelmTabs(tab, this.props.isHelmManaged))
          .map((tab) => (
            <div
              className={`tab-item blue ${tab === selectedTab && "is-active"}`}
              key={tab}
              //  onClick={() => this.setState({ selectedTab: tab })}
            >
              {tab}
            </div>
          ))}
      </div>
    );
  },
  versionFailing: "version failing",
  troubleshootUrl: "troubleshoot url"
};
