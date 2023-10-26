import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import ChangePasswordModal from "@src/components/modals/ChangePasswordModal/ChangePasswordModal";

export default {
  title: "Example/ChangePasswordModal",
  component: ChangePasswordModal,
} as ComponentMeta<typeof ChangePasswordModal>;

const Template: ComponentStory<typeof ChangePasswordModal> = (args) => (
  <MemoryRouter>
    <ChangePasswordModal {...args} />
  </MemoryRouter>
);

export const ChangePasswordModalExample = Template.bind({});

ChangePasswordModalExample.args = {
  isOpen: true,
  closeModal: () => {
    alert("close");
  },
};
