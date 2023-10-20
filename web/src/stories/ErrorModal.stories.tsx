import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import ErrorModal from "@src/components/modals/ErrorModal";

export default {
  title: "Example/ErrorModal",
  component: ErrorModal,
} as ComponentMeta<typeof ErrorModal>;

const Template: ComponentStory<typeof ErrorModal> = (args) => (
  <MemoryRouter>
    <ErrorModal {...args} />
  </MemoryRouter>
);

export const ErrorModalExample = Template.bind({});

ErrorModalExample.args = {
  errorModal: true,
  toggleErrorModal: () => {
    console.log("toggled");
  },
  err: "404",
  errMsg: "Error Message",
  appSlug: "appSlug",
};
