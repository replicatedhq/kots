import React from "react";
import { ComponentStory, ComponentMeta } from "@storybook/react";
import Icon from "@src/components/Icon";
import data from "../components/selection.json";

const iconNames = data.icons.map((item) => item.icon.tags);
const flattenedIconNames = iconNames.flat();

export default {
  title: "Example/Icons",
  component: Icon,
} as ComponentMeta<typeof Icon>;

const Template: ComponentStory<typeof Icon> = () => (
  <div className="tw-flex tw-gap-2 tw-max-w-xl tw-flex-wrap">
    {flattenedIconNames.map((icon, idx) => {
      return (
        <div key={idx}>
          <Icon
            icon={icon}
            size={26}
            className="tw-mx-4 tw-cursor-pointer"
            onClick={() => alert("close toast")}
          />
        </div>
      );
    })}
  </div>
);

export const IconsExample = Template.bind({});
