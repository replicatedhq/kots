import React from "react";
import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import ChangePasswordForm from "@src/components/modals/ChangePasswordModal/ChangePasswordForm";

export default {
  title: "Example/ChangePasswordForm",
  component: ChangePasswordForm,
} as ComponentMeta<typeof ChangePasswordForm>;

const Template: ComponentStory<typeof ChangePasswordForm> = (args) => (
  <MemoryRouter>
    <ChangePasswordForm {...args} />
  </MemoryRouter>
);

export const ChangePasswordFormExample = Template.bind({});

ChangePasswordFormExample.args = {
  handleClose: () => {
    alert("close");
  },
  handleSetPasswordChangeSuccessful: () => {
    alert("set password change successful");
  },
};
