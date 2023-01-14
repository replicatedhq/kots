import React, { createContext, ReactNode, useState } from "react";
import Toast from "@src/components/shared/Toast";
import Icon from "@components/Icon";

interface ToastContextProps {
  isToastVisible: boolean;
  setIsToastVisible: (val: boolean) => void;
  isCancelled: boolean;
  setIsCancelled: (val: boolean) => void;
  deleteBundleId: string;
  setDeleteBundleId: (val: string) => void;
  toastMessage: string;
  setToastMessage: (val: string) => void;
}

const ToastContext = createContext({} as ToastContextProps);

const ToastProvider = ({ children }: { children: ReactNode }) => {
  const [isToastVisible, setIsToastVisible] = useState(false);
  const [isCancelled, setIsCancelled] = useState(false);
  const [deleteBundleId, setDeleteBundleId] = useState("");
  const [toastMessage, setToastMessage] = useState("");
  return (
    <ToastContext.Provider
      value={{
        isToastVisible,
        setIsToastVisible,
        isCancelled,
        setIsCancelled,
        deleteBundleId,
        setDeleteBundleId,
        toastMessage,
        setToastMessage,
      }}
    >
      <Toast isToastVisible={isToastVisible} type="warning">
        <div className="tw-flex tw-items-center">
          <p className="tw-ml-2 tw-mr-4">{toastMessage}</p>
          <span
            onClick={() => setIsCancelled(true)}
            className="tw-underline tw-cursor-pointer"
          >
            undo
          </span>
          <Icon
            icon="close"
            size={10}
            className="tw-mx-4 tw-cursor-pointer"
            onClick={() => setIsToastVisible(false)}
          />
        </div>
      </Toast>
      {children}
    </ToastContext.Provider>
  );
};

export { ToastContext, ToastProvider };
