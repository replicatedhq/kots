import { ComponentStory, ComponentMeta } from "@storybook/react";
import { MemoryRouter } from "react-router-dom";
import RestoreSnapshotModal from "@src/components/modals/RestoreSnapshotModal";
import { ChangeEvent, useState } from "react";

export default {
  title: "Example/RestoreSnapshotModal",
  component: RestoreSnapshotModal,
} as ComponentMeta<typeof RestoreSnapshotModal>;

const Template: ComponentStory<typeof RestoreSnapshotModal> = (args) => {
  const [appSlugMismatch, setAppSlugMismatch] = useState(false);
  const [appSlugToRestore, setAppSlugToRestore] = useState("");
  const handleApplicationSlugChange = (e: ChangeEvent<HTMLInputElement>) => {
    if (appSlugMismatch) {
      setAppSlugMismatch(false);
    }
    setAppSlugToRestore(e.target.value);
  };
  return (
    <MemoryRouter>
      <RestoreSnapshotModal
        {...args}
        appSlugMismatch={appSlugMismatch}
        appSlugToRestore={appSlugToRestore}
        handleApplicationSlugChange={handleApplicationSlugChange}
      />
    </MemoryRouter>
  );
};

export const RestoreSnapshotModalExample = Template.bind({});

RestoreSnapshotModalExample.args = {
  restoreSnapshotModal: true,
  toggleRestoreModal: () => alert("toggleRestoreModal"),
  handleRestoreSnapshot: () => alert("handleRestoreSnapshot"),
  snapshotToRestore: "snapshot-string",
  restoringSnapshot: false,
  restoreErr: false,
  restoreErrorMsg: "restore-error-message",
  app: { id: "id", name: "name", slug: "slug" },
  apps: [
    { id: "id-1", name: "name1", slug: "slug1" },
    { id: "id-2", name: "name2", slug: "slug2" },
  ],
};
