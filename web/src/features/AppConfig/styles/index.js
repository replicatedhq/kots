import styled from "styled-components";
import * as colors from "../../../styles/colors";

export const GroupTitle = styled.a`
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: ${(props) => (props.fontSize && `${props.fontSize}px`) || "14px"};
  &:hover {
    color: ${colors.primary};
  }
`;

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
  & ${GroupTitle} {
    margin-bottom: 0px;
    color: ${colors.secondaryText};
  }
`;

export const SideNavItems = styled(SideNavGroup)`
  display: none;
  padding-left: 15px;
`;

export const SideNavWrapper = styled.div`
  background: ${colors.subNav};
  max-width: 250px;
  width: 100%;
  padding: 10px;
  border-radius: 4px;
  overflow: auto;
  & ${SideNavGroup} {
    .icon.u-darkDropdownArrow {
      margin-left: 6px;
      top: 6px;
      transition: transform 0.1s ease-in-out;
    }

    &.group-open {
      & ${GroupTitle} {
        font-weight: 700;
      }
      .icon.u-darkDropdownArrow {
        transform: rotate(180deg);
      }
      & ${SideNavItems} {
        display: block;
      }
    }
  }
`;
