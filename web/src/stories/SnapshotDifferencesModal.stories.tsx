import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import SnapshotDifferencesModal from "@src/components/modals/SnapshotDifferencesModal";

export default {
  title: "Example/SnapshotDifferencesModal",
  component: SnapshotDifferencesModal,
} as ComponentMeta<typeof SnapshotDifferencesModal>;

const Template: ComponentStory<typeof SnapshotDifferencesModal> = (args) => (
  <MemoryRouter>
    <SnapshotDifferencesModal {...args} />
  </MemoryRouter>
);

export const SnapshotDifferencesModalExample = Template.bind({});

SnapshotDifferencesModalExample.args = {
  snapshotDifferencesModal: true,
  toggleSnapshotDifferencesModal: () => alert("toggle modal"),
};
