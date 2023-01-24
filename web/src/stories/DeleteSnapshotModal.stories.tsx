import React from "react";
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
  deleteSnapshotModal: true,
  toggleConfirmDeleteModal: () => alert("toggle modal"),
  snapshotToDelete: { name: "name", date: "date", status: "status" },
  deletingSnapshot: false,
  handleDeleteSnapshot: () => alert("delete snapshot"),
  deleteErr: "delete error",
  deleteErrorMsg: "delete error message",
};
