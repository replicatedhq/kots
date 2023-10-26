import { createContext, ReactNode, useState } from "react";

interface ToastContextProps {
  isToastVisible: boolean;
  setIsToastVisible: (val: boolean) => void;
  isCancelled: boolean;
  setIsCancelled: (val: boolean) => void;
  deleteBundleId: string;
  setDeleteBundleId: (val: string) => void;
  toastMessage: string;
  setToastMessage: (val: string) => void;
  toastChild: ReactNode;
  setToastChild: (val: ReactNode) => void;
  toastType: "success" | "error" | "warning";
  setToastType: (val: "success" | "error" | "warning") => void;
}

const ToastContext = createContext({} as ToastContextProps);

const ToastProvider = ({ children }: { children: ReactNode }) => {
  const [isToastVisible, setIsToastVisible] = useState(false);
  const [isCancelled, setIsCancelled] = useState(false);
  const [deleteBundleId, setDeleteBundleId] = useState("");
  const [toastMessage, setToastMessage] = useState("");
  const [toastType, setToastType] = useState<"success" | "error" | "warning">(
    "success"
  );
  const [toastChild, setToastChild] = useState<ReactNode>(null);
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
        toastType,
        setToastType,
        toastChild,
        setToastChild,
      }}
    >
      {children}
    </ToastContext.Provider>
  );
};

export { ToastContext, ToastProvider };
