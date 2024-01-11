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
  <div className="tw-grid-rows-5 tw-gap-2 tw-max-w-xl tw-flex-wrap">
    {flattenedIconNames.map((icon, idx) => {
      return (
        <div key={idx} className={"tw-flex tw-flex-row tw-pb-2"}>
          <p className={"tw-pb-2"}>{icon}</p>
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
