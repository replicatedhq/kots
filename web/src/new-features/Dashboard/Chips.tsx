import React from "react";
import CheckCircleIcon from "@mui/icons-material/CheckCircle";
import ErrorIcon from "@mui/icons-material/Error";
import WarningIcon from "@mui/icons-material/Warning";
import { Chip } from "@mui/material";

const titleCase = (string: string) => {
  if (string) {
    return string[0].toUpperCase() + string.slice(1).toLowerCase();
  }
};

export const PreflightStatusChip = ({
  status,
  onClick,
}: {
  status: string | undefined;
  onClick: () => void;
}) => {
  let color = "default";
  let icon;
  if (status === "Passed") {
    icon = <CheckCircleIcon />;
    color = "success";
  } else if (status === "Failed") {
    icon = <ErrorIcon />;
    color = "error";
  } else if (status === "warning") {
    icon = <WarningIcon />;
    color = "warning";
  }
  return (
    <Chip
      icon={icon}
      label={status}
      variant="filled"
      color={color}
      sx={{ marginTop: "4px" }}
      onClick={() => onClick()}
    />
  );
};

export const StatusChip = ({
  label,
  onClick,
}: {
  label: string | undefined;
  onClick: () => void;
}) => {
  let color = "default";
  let icon;
  if (label === "ready") {
    icon = <CheckCircleIcon />;
    color = "success";
  } else if (label === "failed") {
    icon = <ErrorIcon />;
    color = "error";
  } else if (label === "updating") {
    icon = <WarningIcon />;
    color = "warning";
  }

  return (
    <Chip
      icon={icon}
      label={titleCase(label)}
      variant="filled"
      color={color}
      sx={{ marginTop: "4px" }}
      onClick={() => onClick()}
    />
  );
};
