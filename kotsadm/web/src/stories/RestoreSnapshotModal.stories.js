import React from "react";

import RestoreSnapshotModal from "../components/modals/RestoreSnapshotModal";
import "../scss/components/snapshots/AppSnapshots.scss";
import "../scss/index.scss";

export default {
  title: "KOTSADM/RestoreSnapshotModal",
  component: RestoreSnapshotModal,
};

const Template = (args) => <RestoreSnapshotModal {...args} />;

export const Modal = Template.bind({});

Modal.args = {
  restoreSnapshotModal: true,
  app: {
    slug: `testslug`,
  },
};
