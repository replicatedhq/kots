import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import AutomaticUpdatesModal from "@src/components/modals/AutomaticUpdatesModal";

export default {
  title: "Example/AutomaticUpdatesModal",
  component: AutomaticUpdatesModal,
} as ComponentMeta<typeof AutomaticUpdatesModal>;

const Template: ComponentStory<typeof AutomaticUpdatesModal> = (args) => {
  return (
    <MemoryRouter>
      <AutomaticUpdatesModal {...args} />
    </MemoryRouter>
  );
};
export const AutomaticUpdatesModalExample = Template.bind({});

AutomaticUpdatesModalExample.args = {
  isOpen: true,
  onRequestClose: () => {
    alert("close modal");
  },
  updateCheckerSpec: "string",
  autoDeploy: "autoDeploy",
  appSlug: "appSlug",
  isSemverRequired: false,
  gitopsIsConnected: false,
  onAutomaticUpdatesConfigured: () => {},
  isHelmManaged: false,
};
