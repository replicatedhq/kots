import styled from "styled-components";
import * as colors from "../../../styles/colors";

export const SideNavWrapper = styled.div`
  background: ${colors.subNav};
  max-width: 200px;
  width: 100%;
  padding: 10px;
  border-radius: 4px;
  overflow: auto;

   & a
   {
    display: block;
    margin-top: 10px;
    margin-bottom: 10px;
    color: ${colors.subNavText};
    cursor: pointer;
    font-weight: 400;

    &.active-item {
      font-weight: 500;
      color: ${colors.secondaryText};
    }

    &:hover {
      color: ${colors.secondaryText};
    }

    &:last-child {
      margin-bottom: 0;
    }
`;
