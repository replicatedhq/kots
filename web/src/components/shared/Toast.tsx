import { ReactNode } from "react";

export interface ToastProps {
  isToastVisible: boolean;
  type: "success" | "error" | "warning";
  children: ReactNode;
}

const toastType = (type: "success" | "error" | "warning") => {
  switch (type) {
    case "success":
      return "tw-bg-[#38cc97]";
    case "error":
      return "tw-bg-[#f65c5c]";
    case "warning":
      return "tw-bg-[#FFA500]";
    default:
      return "tw-bg-[#38cc97]";
  }
};

const Toast = ({ isToastVisible, type, children }: ToastProps) => {
  return (
    <div
      className={`tw-absolute tw-w-auto tw-h-auto tw-left-6 tw-z-40 
      tw-bg-white tw-border tw-border-gray-300 tw-rounded 
      tw-shadow-md tw-p-2 tw-text-gray-700 tw-text-sm toast ${
        isToastVisible ? "visible" : ""
      }
    `}
    >
      <div className="tw-flex tw-items-center">
        <div
          className={`${toastType(type)} tw-w-1 tw-h-10 tw-border tw-rounded`}
        ></div>
        {children}
      </div>
    </div>
  );
};

export default Toast;
