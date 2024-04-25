import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import DeleteSnapshotModal from "@src/components/modals/DeleteSnapshotModal";

export default {
  title: "Example/DeleteSnapshotModal",
  component: DeleteSnapshotModal,
} as ComponentMeta<typeof DeleteSnapshotModal>;

const Template: ComponentStory<typeof DeleteSnapshotModal> = (args) => (
  <MemoryRouter>
    <DeleteSnapshotModal {...args} />
  </MemoryRouter>
);

export const DeleteSnapshotModalExample = Template.bind({});

DeleteSnapshotModalExample.args = {
  featureName: "snapshot",
  deleteSnapshotModal: true,
  toggleConfirmDeleteModal: () => alert("toggle modal"),
  snapshotToDelete: {
    name: "snapshot name",
    status: "Deleting",
    trigger: "manual",
    sequence: 1,
    startedAt: "2023-01-26T13:48:47",
    finishedAt: "2022-01-26T13:50:47",
    expiresAt: "2023-02-010T13:48:47",
    volumeCount: 1,
    volumeSuccessCount: 1,
    volumeBytes: 0,
    volumeSizeHuman: "0B",
  },
  deletingSnapshot: false,
  handleDeleteSnapshot: () => alert("delete snapshot"),
  deleteErr: true,
  deleteErrorMsg: "delete error message",
};
