import { ComponentStory, ComponentMeta } from "@storybook/react";
import Toast from "@src/components/shared/Toast";
import Icon from "@src/components/Icon";

export default {
  title: "Example/Toast",
  component: Toast,
} as ComponentMeta<typeof Toast>;

const Template: ComponentStory<typeof Toast> = () => (
  <div className="tw-flex">
    <div
      className="tw-relative tw-w-auto"
      style={{ width: "500px", height: "200px" }}
    >
      <Toast isToastVisible={true} type="success">
        <div className="tw-flex tw-items-center">
          <p className="tw-ml-2 tw-mr-4">Success!</p>
          <Icon
            icon="close"
            size={10}
            className="tw-mx-4 tw-cursor-pointer"
            onClick={() => alert("close toast")}
          />
        </div>
      </Toast>
    </div>
    <div className="tw-relative" style={{ width: "500px", height: "200px" }}>
      <Toast isToastVisible={true} type="warning">
        <div className="tw-flex tw-items-center">
          <p className="tw-ml-2 tw-mr-4">Deleting item</p>
          <span
            onClick={() => alert("undo")}
            className="tw-underline tw-cursor-pointer"
          >
            undo
          </span>
          <Icon
            icon="close"
            size={10}
            className="tw-mx-4 tw-cursor-pointer"
            onClick={() => alert("close toast")}
          />
        </div>
      </Toast>
    </div>
    <div className="tw-relative" style={{ width: "500px", height: "200px" }}>
      <Toast isToastVisible={true} type="error">
        <div className="tw-flex tw-items-center">
          <p className="tw-ml-2 tw-mr-4">Error! Please do something!</p>
          <Icon
            icon="close"
            size={10}
            className="tw-mx-4 tw-cursor-pointer"
            onClick={() => alert("close toast")}
          />
        </div>
      </Toast>
    </div>
  </div>
);

export const ToastExample = Template.bind({});
