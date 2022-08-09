import styled from "styled-components";
import * as colors from "../../../styles/colors";

export const SideNavWrapper = styled.div`
  background: ${colors.subNav};
  max-width: 200px;
  width: 100%;
  padding: 10px;
  border-radius: 4px;
  overflow: auto;
`;

// .AppConfigSidenav--group a,
// .AppConfigSidenav--items a {
//   display: block;
//   margin-top: 10px;
//   margin-bottom: 10px;
//   color: ${colors.subNavText};
//   cursor: pointer;
//   font-weight: 400;

//   &.active-item {
//     font-weight: 500;
//     color: ${colors.secondaryText};
//   }

//   &:hover {
//     color: ${colors.secondaryText};
//   }

//   &:last-child {
//     margin-bottom: 0;
//   }
// }
export const SideNavGroup = styled.div`
  a {
    display: block;
  margin-top: 10px;
  margin-bottom: 10px;
  color: ${colors.subNavText};
  cursor: pointer;
  font-weight: 400;
  }

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
  a.group-title {
      margin-bottom: 0;
      color: {color.secondaryText};
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
      &:hover {
        color: ${colors.primary};
      }
    }
    .icon.u-darkDropdownArrow {
      margin-left: 6px;
      top: 6px;
      transition: transform 0.1s ease-in-out;
    }
    .AppConfigSidenav--items {
      display: none;
    }
    &.group-open {
      a.group-title {
        font-weight: 700;
      }
      .icon.u-darkDropdownArrow {
        transform: rotate(180deg);
      }
      .AppConfigSidenav--items {
        display: block;
      }
    }
  }
`;
