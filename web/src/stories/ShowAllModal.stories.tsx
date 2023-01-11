import React from "react";
import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import ShowAllModal from "@src/components/modals/ShowAllModal";

export default {
  title: "Example/ShowAllModal",
  component: ShowAllModal
} as ComponentMeta<typeof ShowAllModal>;

const Template: ComponentStory<typeof ShowAllModal> = (args) => (
  <MemoryRouter>
    <ShowAllModal {...args} />
  </MemoryRouter>
);

export const ShowAllModalExample = Template.bind({});

ShowAllModalExample.args = {
  displayShowAllModal: true,
  toggleShowAllModal: () => {
    alert("toggle modal");
  },
  dataToShow: "hi",
  name: "name"
};
