import useLicense from "@features/App/api/getLicense";
import { useLicenseWithIntercept } from "@features/App/api/useLicense";
import { useEffect } from "react";
import { usePrevious } from "@src/hooks/usePrevious";

const LicenseTester = ({ appSlug, setLoader }) => {
  const { data, isSlowLoading } = useLicenseWithIntercept(appSlug);
  const { license } = data || {};
  const { entitlements } = license || [];

  useEffect(() => {
    setLoader(isSlowLoading);
  }, [isSlowLoading, data]);

  return (
    <div>
      {isSlowLoading ? (
        <h1>HOLD ON... IT'S LOADING!!</h1>
      ) : data && entitlements.length === 0 ? (
        <h1>YOU DON'T HAVE A entitlements FOR THIS APP</h1>
      ) : (
        entitlements?.map((item) => {
          return <div key={item.title}>{item.title}</div>;
        })
      )}
    </div>
  );
};

export default LicenseTester;
