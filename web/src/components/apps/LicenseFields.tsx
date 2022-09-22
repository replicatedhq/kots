import React from "react";
// TODO: add type checking support for styled components or add a global ignore
// @ts-ignore
import styled from "styled-components";

export const CustomerLicenseFields = styled.div`
  background: ${(props: { count: number }) =>
    props.count < 5 ? "none" : "#f5f8f9"};
  border-radius: ${(props: { count: number }) => (props.count < 5 ? 0 : "6px")};
  border: ${(props: { count: number }) =>
    props.count < 5 ? "none" : "1px solid #bccacd"};
  padding: ${(props: { count: number }) => (props.count < 5 ? 0 : "10px")};
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

type Entitlement = {
  label: string;
  title: string;
  value: string;
  valueType: "Text" | "Boolean" | "Integer" | "String";
};

const LicenseFields = ({
  entitlements,
  toggleShowDetails,
  toggleHideDetails,
  entitlementsToShow,
}: {
  entitlements: Entitlement[];
  toggleShowDetails: (title: Entitlement) => void;
  toggleHideDetails: (title: Entitlement) => void;
  entitlementsToShow: Entitlement[];
}) => {
  return (
    <CustomerLicenseFields
      className="flex flexWrap--wrap"
      count={entitlements.length}
    >
      {entitlements?.map((entitlement) => {
        const displayedEntitlement = entitlementsToShow?.find(
          (f) => f === entitlement.title
        );
        const isTextField = entitlement.valueType === "Text";
        if (
          entitlement.value.length > 100 &&
          displayedEntitlement !== entitlement.title
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
          displayedEntitlement === entitlement.title
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
                {entitlement.value.toString() + " "}
              </span>
            </CustomerLicenseField>
          );
        }
      })}
    </CustomerLicenseFields>
  );
};

export default LicenseFields;
