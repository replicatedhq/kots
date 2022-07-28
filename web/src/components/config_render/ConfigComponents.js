import styled from "styled-components";

export const ConfigWrapper = styled.div`
  margin-top: ${(props) => (props.marginTop ? props.marginTop : "0")};
  display: ${(props) => (props.hidden ? "none" : "block")};
  order: ${(props) => (props.order ? props.order : "")};
  &:first-child {
    margin-top: 0;
  }
`;

export const ConfigItems = styled.div`
  padding: 10px;
  //TODO: update to use theme color
  background-color: #f5f8f9;
  border-radius: 4px;
  margin-top: 15px;
  display: ${(props) => (props.display ? props.display : "block")};
  grid-template-columns: 1fr 1fr;
  grid-gap: 15px 0;
`;
