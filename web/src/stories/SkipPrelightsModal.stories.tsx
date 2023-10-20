import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import SkipPreflightsModalModal from "@src/components/shared/modals/SkipPreflightsModal";

export default {
  title: "Example/SkipPreflightsModalModal",
  component: SkipPreflightsModalModal,
} as ComponentMeta<typeof SkipPreflightsModalModal>;

const Template: ComponentStory<typeof SkipPreflightsModalModal> = (args) => (
  <MemoryRouter>
    <SkipPreflightsModalModal {...args} />
  </MemoryRouter>
);

export const SkipPreflightsModalModalExample = Template.bind({});

SkipPreflightsModalModalExample.args = {
  showSkipModal: true,
  hideSkipModal: () => alert("hide skip modal"),
  onIgnorePreflightsAndDeployClick: () => alert("ignore kots downstream"),
  onForceDeployClick: () => alert("force deploy click"),
};
