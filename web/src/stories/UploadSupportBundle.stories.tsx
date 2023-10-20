import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import UploadSupportBundleModal from "@src/components/troubleshoot/UploadSupportBundleModal";

export default {
  title: "Example/UploadSupportBundleModal",
  component: UploadSupportBundleModal,
} as ComponentMeta<typeof UploadSupportBundleModal>;

const Template: ComponentStory<typeof UploadSupportBundleModal> = () => (
  <MemoryRouter>
    <UploadSupportBundleModal />
  </MemoryRouter>
);

export const UploadSupportBundleModalExample = Template.bind({});
