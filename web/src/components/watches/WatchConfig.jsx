import React from "react";
import StateFileViewer from "../state/StateFileViewer";

export default function WatchConfig(props) {
  const { watch } = props;
  return (
    <div className="flex-column flex1">
      <StateFileViewer watch={watch} />
    </div>
  )
}