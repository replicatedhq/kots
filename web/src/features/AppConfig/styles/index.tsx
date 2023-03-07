import styled from "styled-components";
import * as colors from "../../../styles/colors";

export const SideNavGroup = styled.div`
  a {
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
  }
`;

export const SideNavItems = styled(SideNavGroup)`
  display: none;
  padding-left: 15px;
`;

export const SideNavWrapper = styled.div`
  background: ${colors.subNav};
  width: 250px;
  padding: 10px;
  border-radius: 4px;
  overflow: auto;
  & ${SideNavGroup} {
    .icon.u-darkDropdownArrow,
    .arrow-down {
      margin-left: 6px;
      top: 6px;
      transition: transform 0.1s ease-in-out;
    }

    &.group-open {
      .icon.u-darkDropdownArrow,
      .arrow-down {
        transform: rotate(180deg);
      }
      & ${SideNavItems} {
        display: block;
      }
    }
  }
  @media screen and (max-width: 1200px) {
    width: 100%;
    max-width: 200px;
  }
`;
