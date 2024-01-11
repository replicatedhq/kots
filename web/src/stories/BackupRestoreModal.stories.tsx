import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import BackupRestoreModal from "@src/components/modals/BackupRestoreModal";

export default {
  title: "Example/BackupRestoreModal",
  component: BackupRestoreModal,
} as ComponentMeta<typeof BackupRestoreModal>;

const Template: ComponentStory<typeof BackupRestoreModal> = (args) => (
  <MemoryRouter>
    <BackupRestoreModal {...args} />
  </MemoryRouter>
);

export const BackupRestoreModalExample = Template.bind({});

BackupRestoreModalExample.args = {
  veleroNamespace: "velero",
  isMinimalRBACEnabled: false,
  restoreSnapshotModal: true,
  toggleRestoreModal: () => {
    alert("toggle modal");
  },
  snapshotToRestore: { name: "snapshot-1" },
  includedApps: ["app-1"],
  // change this to empty string and turn on appSlugMismatch to see error
  selectedRestore: "full",
  onChangeRestoreOption: () => {
    alert("change restore option");
  },
  selectedRestoreApp: { name: "name" },
  onChangeRestoreApp: () => {
    alert("change restore app");
  },
  getLabel: () => {
    alert("get label");
  },
  handleApplicationSlugChange: () => {
    alert("handle application slug change");
  },
  appSlugToRestore: "app-2",
  appSlugMismatch: false,
  handlePartialRestoreSnapshot: () => {
    alert("handle partial restore snapshot");
  },
};
