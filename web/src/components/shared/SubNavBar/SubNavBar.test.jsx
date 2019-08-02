import React from "react";
import { shallow } from "enzyme";

import SubNavBar from "./SubNavBar";

describe("<SubNavBar> tests", () => {
  const topLevelWatch = {
    cluster: null,
    metadata: "{\"applicationType\": \"helm\" }"
  };

  const childWatch = {
    cluster: {},
    metadata: "{}"
  };

  it("Renders without crashing", () => {
    const wrapper = shallow(<SubNavBar watch={topLevelWatch} />);

    expect(wrapper).toBeTruthy();
  });

  it("Renders appropriate tabs for a top level watch", () => {
    const wrapper = shallow(<SubNavBar watch={topLevelWatch} />);

    expect(wrapper.find("li")).toHaveLength(4);
  });

  it("... for a child watch", () => {
    const wrapper = shallow(<SubNavBar watch={childWatch} />);

    expect(wrapper.find("li")).toHaveLength(1);
  });

});