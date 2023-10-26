import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import ConfigureRedactorsModal from "@src/components/troubleshoot/ConfigureRedactorsModal";

export default {
  title: "Example/ConfigureRedactorsModal",
  component: ConfigureRedactorsModal,
} as ComponentMeta<typeof ConfigureRedactorsModal>;

const Template: ComponentStory<typeof ConfigureRedactorsModal> = (args) => (
  <MemoryRouter>
    <ConfigureRedactorsModal {...args} />
  </MemoryRouter>
);

export const ConfigureRedactorsModalExample = Template.bind({});

ConfigureRedactorsModalExample.args = {
  onClose: () => alert("onClose"),
};
