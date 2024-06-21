import { isEqual, some, zipWith } from "lodash";
import { createContext, useContext, useEffect, useState } from "react";

export const UpgradeServiceContext = createContext(null);

export const UpgradeServiceProvider = ({ children }: any) => {
  const [existingConfig, setExistingConfig] = useState(null);
  const [config, setConfig] = useState(null);
  const [numberOfConfigChanges, setNumberOfConfigChanges] = useState(0);

  useEffect(() => {
    const nameMismatch = some(
      existingConfig,
      (obj, idx) => obj.name !== config[idx].name
    );
    if (nameMismatch) {
      console.error("Object names don't match at some index.");
      return;
    }
    const existingConfigValues =
      (existingConfig && existingConfig[0]?.items) || null;
    const configValues = (config && config[0]?.items) || null;

    console.log(existingConfigValues, configValues);

    // Count value changes using lodash _.zipWith and _.isEqual
    const numberOfChanges = zipWith(
      existingConfigValues,
      configValues,
      (obj1, obj2) => isEqual(obj1.value, obj2.value)
    ).length;
    setNumberOfConfigChanges(numberOfChanges);
    console.log("what", numberOfChanges, existingConfig, config);
  }, [config]);
  return (
    <UpgradeServiceContext.Provider
      value={{
        config,
        setConfig,
        existingConfig,
        setExistingConfig,
        numberOfConfigChanges,
      }}
    >
      {children}
    </UpgradeServiceContext.Provider>
  );
};

export const useUpgradeServiceContext = () => {
  const context = useContext(UpgradeServiceContext);
  if (!context) {
    throw new Error(
      "useUpgradeServiceContext must be used within a UpgradeServiceProvider"
    );
  }
  return context;
};
