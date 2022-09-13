import React from "react";
import styled from "styled-components";

const LicenseFields = ({
  appLicense,
  toggleShowDetails,
  toggleHideDetails,
  entitlementsToShow,
}) => {
  return (
    <CustomerLicenseFields className="flex flexWrap--wrap">
      {appLicense.entitlements?.map((entitlement, i) => {
        const currEntitlement = entitlementsToShow?.find(
          (f) => f === entitlement.title
        );
        const isTextField = entitlement.valueType === "Text";
        const isBooleanField = entitlement.valueType === "Boolean";
        if (
          entitlement.value.length > 100 &&
          currEntitlement !== entitlement.title
        ) {
          return (
            <CustomerLicenseField
              key={entitlement.label}
              className={`u-fontSize--small u-lineHeight--normal u-textColor--secondary u-fontWeight--medium u-marginRight--10 u-marginLeft--5`}
            >
              {" "}
              {entitlement.title}:{" "}
              <span
                className={`u-fontWeight--bold ${
                  isTextField && "u-fontFamily--monospace"
                }`}
              >
                {" "}
                {entitlement.value.slice(0, 100) + "..."}{" "}
              </span>
              <span
                className="replicated-link"
                onClick={() => toggleShowDetails(entitlement.title)}
              >
                show
              </span>
            </CustomerLicenseField>
          );
        } else if (
          entitlement.value.length > 100 &&
          currEntitlement === entitlement.title
        ) {
          return (
            <CustomerLicenseField
              key={entitlement.label}
              className={`u-fontSize--small u-lineHeight--normal u-textColor--secondary u-fontWeight--medium u-marginRight--10 u-marginLeft--5`}
            >
              {" "}
              {entitlement.title}:{" "}
              <span
                className={`u-fontWeight--bold ${
                  isTextField && "u-fontFamily--monospace"
                }`}
              >
                {" "}
                {entitlement.value}{" "}
              </span>
              <span
                className="replicated-link"
                onClick={() => toggleHideDetails(entitlement.title)}
              >
                hide
              </span>
            </CustomerLicenseField>
          );
        } else {
          return (
            <CustomerLicenseField
              key={entitlement.label}
              className={`u-fontSize--small u-lineHeight--normal u-textColor--secondary u-fontWeight--medium u-marginRight--10 u-marginLeft--5`}
            >
              {" "}
              {entitlement.title}:{" "}
              <span
                className={`u-fontWeight--bold ${
                  isTextField && "u-fontFamily--monospace"
                }`}
              >
                {" "}
                {isBooleanField
                  ? entitlement.value.toString()
                  : entitlement.value}{" "}
              </span>
            </CustomerLicenseField>
          );
        }
      })}
    </CustomerLicenseFields>
  );
};

export default LicenseFields;

export const CustomerLicenseFields = styled.div`
  background: #f5f8f9;
  border-radius: 6px;
  border: 1px solid #bccacd;
  padding: 10px;
  line-height: 25px;
`;

export const CustomerLicenseField = styled.span`
  margin-right: 15px;
  display: block;
  overflow-wrap: anywhere;
  max-width: 100%;
`;

export const ExpandButton = styled.button`
  background: none;
  border: none;
  color: #007cbb;
  cursor: pointer;
  font-size: 12px;
  padding-left: 0;
`;
