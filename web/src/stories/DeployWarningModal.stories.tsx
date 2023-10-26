import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import DeployWarningModalModal from "@src/components/shared/modals/DeployWarningModal";

export default {
  title: "Example/DeployWarningModalModal",
  component: DeployWarningModalModal,
} as ComponentMeta<typeof DeployWarningModalModal>;

const Template: ComponentStory<typeof DeployWarningModalModal> = (args) => (
  <MemoryRouter>
    <DeployWarningModalModal {...args} />
  </MemoryRouter>
);

export const DeployWarningModalModalExample = Template.bind({});

DeployWarningModalModalExample.args = {
  showDeployWarningModal: true,
  hideDeployWarningModal: () => alert("hide deploy warning modal"),
  onForceDeployClick: () => alert("force deploy click"),
};
