import React from "react";
import { shallow } from "enzyme";

import SideBar from "./SideBar";

describe("<SideBar> tests", () => {
  it("Renders without crashing", () => {
    const wrapper = shallow(<SideBar />);

    expect(wrapper).toBeTruthy();
  });
});