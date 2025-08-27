import styled from "styled-components";
import * as colors from "./colors";

interface FlexProps {
  align?: string;
  direction?: string;
  flex?: string;
  gap?: string;
  justifyContent?: string;
  justifyItems?: string;
  m?: string;
  mb?: string;
  ml?: string;
  mr?: string;
  mt?: string;
  mx?: string;
  my?: string;
  p?: string;
  pb?: string;
  pl?: string;
  pr?: string;
  pt?: string;
  px?: string;
  py?: string;
  width?: string;
  wrap?: string;
}
export const Flex = styled.div<FlexProps>`
  display: flex;
  flex-direction: ${(props) => props.direction || "row"};
  flex-wrap: ${(props) => props.wrap || "nowrap"};
  justify-content: ${(props) => props.justifyContent || "flex-start"};
  justify-items: ${(props) => props.justifyItems || "flex-start"};
  align-items: ${(props) => props.align || "flex-start"};
  gap: ${(props) => props.gap};
  padding: ${(props) => props.p && `${props.p}px`};
  padding-top: ${(props) =>
    (props.pt && `${props.pt}px`) || (props.px && `${props.py}px`)};
  padding-bottom: ${(props) =>
    (props.pb && `${props.pb}px`) || (props.py && `${props.py}px`)};
  padding-right: ${(props) =>
    (props.pr && `${props.pr}px`) || (props.px && `${props.px}px`)};
  padding-left: ${(props) =>
    (props.pl && `${props.pl}px`) || (props.px && `${props.px}px`)};
  margin: ${(props) => props.m && `${props.m}px`};
  margin-top: ${(props) =>
    (props.mt && `${props.mt}px`) || (props.my && `${props.my}px`)};
  margin-bottom: ${(props) =>
    (props.mb && `${props.mb}px`) || (props.my && `${props.my}px`)};
  margin-right: ${(props) =>
    (props.mr && `${props.mr}px`) || (props.mx && `${props.mx}px`)};
  margin-left: ${(props) =>
    (props.ml && `${props.ml}px`) || (props.mx && `${props.mx}px`)};
  width: ${(props) => props.width};
  flex: ${(props) => props.flex};
  -webkit-flex: ${(props) => props.flex};
  -moz-flex: ${(props) => props.flex};
  -ms-flex: ${(props) => props.flex};
`;

interface ParagraphProps {
  color?: string;
  m?: string;
  mb?: string;
  ml?: string;
  mr?: string;
  mt?: string;
  mx?: string;
  my?: string;
  p?: string;
  pb?: string;
  pl?: string;
  pr?: string;
  pt?: string;
  px?: string;
  py?: string;
  size?: string;
  weight?: string;
}

export const Paragraph = styled.p<ParagraphProps>`
  font-size: ${(props) => (props.size ? `${props.size}px` : "14px")};
  font-weight: ${(props) => props.weight};
  color: ${(props) => (props.color || colors.primary)};
  padding: ${(props) => props.p && `${props.p}px`};

  padding-top: ${(props) =>
    (props.pt && `${props.pt}px`) || (props.px && `${props.py}px`)};
  padding-bottom: ${(props) =>
    (props.pb && `${props.pb}px`) || (props.py && `${props.py}px`)};
  padding-right: ${(props) =>
    (props.pr && `${props.pr}px`) || (props.px && `${props.px}px`)};
  padding-left: ${(props) =>
    (props.pl && `${props.pl}px`) || (props.px && `${props.px}px`)};
  margin: ${(props) => props.m && `${props.m}px`};
  margin-top: ${(props) =>
    (props.mt && `${props.mt}px`) || (props.my && `${props.my}px`)};
  margin-bottom: ${(props) =>
    (props.mb && `${props.mb}px`) || (props.my && `${props.my}px`)};
  margin-right: ${(props) =>
    (props.mr && `${props.mr}px`) || (props.mx && `${props.mx}px`)};
  margin-left: ${(props) =>
    (props.ml && `${props.ml}px`) || (props.mx && `${props.mx}px`)};
`;

interface SpanProps {
  color?: string;
  m?: string;
  mb?: string;
  ml?: string;
  mr?: string;
  mt?: string;
  mx?: string;
  my?: string;
  p?: string;
  pb?: string;
  pl?: string;
  pr?: string;
  pt?: string;
  px?: string;
  py?: string;
  size?: string;
  weight?: string;
}

export const Span = styled.span<SpanProps>`
  font-size: ${(props) => (props.size ? `${props.size}px` : "14px")};
  font-weight: ${(props) => props.weight};
  color: ${(props) => (props.color || colors.primary)};
  padding: ${(props) => props.p && `${props.p}px`};
  padding-top: ${(props) =>
    (props.pt && `${props.pt}px`) || (props.px && `${props.py}px`)};
  padding-bottom: ${(props) =>
    (props.pb && `${props.pb}px`) || (props.py && `${props.py}px`)};
  padding-right: ${(props) =>
    (props.pr && `${props.pr}px`) || (props.px && `${props.px}px`)};
  padding-left: ${(props) =>
    (props.pl && `${props.pl}px`) || (props.px && `${props.px}px`)};
  margin: ${(props) => props.m && `${props.m}px`};
  margin-top: ${(props) =>
    (props.mt && `${props.mt}px`) || (props.my && `${props.my}px`)};
  margin-bottom: ${(props) =>
    (props.mb && `${props.mb}px`) || (props.my && `${props.my}px`)};
  margin-right: ${(props) =>
    (props.mr && `${props.mr}px`) || (props.mx && `${props.mx}px`)};
  margin-left: ${(props) =>
    (props.ml && `${props.ml}px`) || (props.mx && `${props.mx}px`)};
`;
