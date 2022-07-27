export const setOrder = (index, affix) => {
  if (affix === "left") {
    if (index % 2 !== 0) {
      return index;
    }
    return index - 2;
  }
  if (affix === "right") {
    if (index % 2 === 0) {
      return index;
    }
    return index + 1;
  }
  return "oops something went wrong";
};
