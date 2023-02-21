export const setOrder = (index, affix) => {
  if (affix === "left") {
    if (index === 1) {
      return index;
    }
    return index - 1;
  }
  if (affix === "right") {
    if (index === 2) {
      return index;
    }
    return index - 1;
  }
  return "";
};
