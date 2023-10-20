import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import ViewSnapshotLogsModal from "@src/components/modals/ViewSnapshotLogsModal";

export default {
  title: "Example/ViewSnapshotLogsModal",
  component: ViewSnapshotLogsModal,
} as ComponentMeta<typeof ViewSnapshotLogsModal>;

const Template: ComponentStory<typeof ViewSnapshotLogsModal> = (args) => (
  <MemoryRouter>
    <ViewSnapshotLogsModal {...args} />
  </MemoryRouter>
);

export const ViewSnapshotLogsModalExample = Template.bind({});

ViewSnapshotLogsModalExample.args = {
  displayShowSnapshotLogsModal: true,
  toggleViewLogsModal: () => alert("toggle view logs modal"),
  logs: "Test log line one\nTest log line two",
  snapshotDetails: {
    name: "Test snapshot",
  },
  loadingSnapshotLogs: false,
  snapshotLogsErr: false,
  snapshotLogsErrMsg: "",
};
