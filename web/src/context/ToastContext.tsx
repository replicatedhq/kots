import React, { createContext, ReactNode, useState } from "react";
import Toast from "@src/components/shared/Toast";

interface ToastContext {
  isVisible: boolean;
  toggleVisible: () => void;
}

const ToastContext = createContext({} as ToastContext);

const ToastProvider = ({ children }: { children: ReactNode }) => {
  const [isVisible, setIsVisible] = useState(false);
  const toggleVisible = () => {
    setIsVisible(!isVisible);
  };
  return (
    <ToastContext.Provider value={{ isVisible, toggleVisible }}>
      <Toast isVisible={isVisible} toggleVisible={toggleVisible} />
      {children}
    </ToastContext.Provider>
  );
};

export { ToastContext, ToastProvider };
