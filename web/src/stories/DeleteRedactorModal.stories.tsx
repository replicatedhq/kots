import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import DeleteRedactorModal from "@src/components/modals/DeleteRedactorModal";

export default {
  title: "Example/DeleteRedactorModal",
  component: DeleteRedactorModal,
} as ComponentMeta<typeof DeleteRedactorModal>;

const Template: ComponentStory<typeof DeleteRedactorModal> = (args) => (
  <MemoryRouter>
    <DeleteRedactorModal {...args} />
  </MemoryRouter>
);

export const DeleteRedactorModalExample = Template.bind({});

DeleteRedactorModalExample.args = {
  deleteRedactorModal: () => alert("delete redactor modal"),
  toggleConfirmDeleteModal: () => alert("toggle confirm delete modal"),
  handleDeleteRedactor: () => alert("handle delete redactor"),
  redactorToDelete: {
    name: "Redact SSN",
    description: "A redactor that redacts SSN",
    slug: "appslug",
    enabled: false,
    updatedAt: "2021-01-26T13:48:47",
  },
  deletingRedactor: false,
  deleteErrMsg: "",
};
