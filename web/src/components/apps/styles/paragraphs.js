import styled from "styled-components";

export const ParagraphLarge = styled.p`
  font-size: 16px;
  font-weight: ${props => props.fontWeight}

  color: ${props => props.theme.primaryColor};
`;
