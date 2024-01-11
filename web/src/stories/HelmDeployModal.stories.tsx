import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import { HelmDeployModal } from "@src/components/shared/modals/HelmDeployModal";

export default {
  title: "Example/HelmDeployModal",
  component: HelmDeployModal,
} as ComponentMeta<typeof HelmDeployModal>;

const Template: ComponentStory<typeof HelmDeployModal> = (args) => (
  <MemoryRouter>
    <HelmDeployModal {...args} />
  </MemoryRouter>
);

export const HelmDeployModalExample = Template.bind({});

HelmDeployModalExample.args = {
  appSlug: "appslug",
  chartPath: "chart.sentry.io/",
  downloadClicked: () => alert("download clicked"),
  downloadError: false,
  hideHelmDeployModal: () => alert("hide helm deploy modal"),
  saveError: false,
  showHelmDeployModal: true,
  subtitle: "Follow the steps below to upgrade the release.",
  registryUsername: "myUsername",
  registryPassword: "myPassword",
  revision: null,
  title: "Deploy Sentry",
  upgradeTitle: "",
  showDownloadValues: true,
  version: "1.0",
  namespace: "default",
};
