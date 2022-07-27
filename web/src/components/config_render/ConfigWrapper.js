import styled from "styled-components";

export const ConfigWrapper = styled.div`
  margin-top: ${(props) => (props.marginTop ? props.marginTop : "0")};
  display: ${(props) => (props.hidden ? "none" : "block")};
  order: ${(props) => (props.order ? props.order : "")};
  &:first-child {
    margin-top: 0;
  }
`;
