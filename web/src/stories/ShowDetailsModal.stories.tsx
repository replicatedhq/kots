import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import ShowDetailsModal from "@src/components/modals/ShowDetailsModal";

export default {
  title: "Example/ShowDetailsModal",
  component: ShowDetailsModal,
} as ComponentMeta<typeof ShowDetailsModal>;

const Template: ComponentStory<typeof ShowDetailsModal> = (args) => (
  <MemoryRouter>
    <ShowDetailsModal {...args} />
  </MemoryRouter>
);

export const ShowDetailsModalExample = Template.bind({});

ShowDetailsModalExample.args = {
  displayShowDetailsModal: true,
  toggleShowDetailsModal: () => alert("toggle modal"),
  yamlErrorDetails: [{ path: "path", error: "error" }],
  deployView: true,
  showDeployWarningModal: true,
  showSkipModal: false,
  forceDeploy: () => alert("force deploy"),
  slug: "slug",
  sequence: "sequence",
};
