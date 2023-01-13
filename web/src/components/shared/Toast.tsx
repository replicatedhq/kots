import Icon from "@components/Icon";
import React from "react";

interface ToastProps {
  isVisible: boolean;
  toggleVisible: () => void;
}

const Toast = ({ isVisible }: ToastProps) => {
  return (
    <div
      className="tw-absolute tw-bottom-10 tw-left-6 tw-z-40 
      tw-bg-white tw-border tw-border-gray-300 tw-rounded 
      tw-shadow-md tw-p-2 tw-text-gray-700 tw-text-sm 
    "
      style={{
        width: "auto",
        height: "auto",
        display: isVisible ? "none" : "block",
        animation: "fadein 0.5s, fadeout 0.5s 2.5s"
      }}
    >
      <div className="tw-flex tw-items-center">
        <div className="tw-bg-[#FFA500] tw-w-1 tw-h-10 tw-border tw-rounded"></div>
        <div className="tw-flex tw-items-center">
          <p className="tw-ml-2 tw-mr-4">
            Support bundle collected on 1/11 @ 10:56am has been deleted.
          </p>
          <a href="/" className="tw-underline">
            undo
          </a>
          <Icon icon="close" size={10} className="tw-mx-4" />
        </div>
      </div>
    </div>
  );
};

export default Toast;
