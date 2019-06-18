import React from "react";
import { shallow } from "enzyme";

import WatchSidebarItem from "./WatchSidebarItem";

describe("<WatchSidebarItem> tests", () => {
  const dummyWatch = {
    "id": "q9gkniw0piq2j1fb2v12g4asntzpwr2o",
    "stateJSON": "{\"hello\": \"world\"}",
    "watchName": "New Relic",
    "slug": "mrbenj/newrelic-infrastructure",
    "watchIcon": "https://newrelic.com/assets/newrelic/source/NewRelic-logo-square.svg",
    "createdOn": "Tue Jun 04 2019 17:25:44 GMT+0000 (UTC)",
    "lastUpdated": "Wed Jun 12 2019 18:25:33 GMT+0000 (UTC)",
    "metadata": "{\"applicationType\":\"helm\",\"sequence\":0,\"icon\":\"https://newrelic.com/assets/newrelic/source/NewRelic-logo-square.svg\",\"name\":\"newrelic-infrastructure\",\"releaseNotes\":\"[newrelic-infra] Update limits and version (#14454)\\n\\nSigned-off-by: JF Joly \\u003cjoly.pro@gmail.com\\u003e\",\"version\":\"0.12.0\",\"license\":{\"id\":\"\",\"assignee\":\"\",\"createdAt\":\"0001-01-01T00:00:00Z\",\"expiresAt\":\"0001-01-01T00:00:00Z\",\"type\":\"\"}}",
    "contributors": [
      {
        "id": "pp5ez7iqd9xsc2zebdnbijqw62snauf0",
        "createdAt": "Wed May 22 2019 22:11:15 GMT+0000 (UTC)",
        "githubId": 7918387,
        "login": "mrbenj",
        "avatar_url": "https://avatars0.githubusercontent.com/u/7918387?v=4"
      }
    ],
    "currentVersion": {
      "title": "0.12.0",
      "status": "deployed",
      "createdOn": "Tue Jun 04 2019 17:25:44 GMT+0000 (UTC)",
      "sequence": 0,
      "pullrequestNumber": null
    },
    "pendingVersions": [],
    "pastVersions": [],
    "notifications": [
      {
        "id": "n4e0elb51ns644eqtuvv5egqqk5fnkcf",
        "createdOn": "Tue Jun 04 2019 17:25:44 GMT+0000 (UTC)",
        "updatedOn": null,
        "triggeredOn": null,
        "enabled": 1,
        "webhook": {
          "uri": "placeholder"
        },
        "email": null
      }
    ],
    "watches": [
      {
        "id": "00qd4q9ewckx64q7xre8sj6hzeejkvzb",
        "stateJSON": "{\"hello\": \"world\"}",
        "contributors": [
          {
            "id": "pp5ez7iqd9xsc2zebdnbijqw62snauf0",
            "createdAt": "Wed May 22 2019 22:11:15 GMT+0000 (UTC)",
            "githubId": 7918387,
            "login": "mrbenj",
            "avatar_url": "https://avatars0.githubusercontent.com/u/7918387?v=4"
          }
        ],
        "currentVersion": {
          "title": "0.12.0",
          "status": "pending",
          "createdOn": "Tue Jun 04 2019 17:28:34 GMT+0000 (UTC)",
          "sequence": 0,
          "pullrequestNumber": null
        },
        "pendingVersions": [],
        "pastVersions": [],
        "notifications": [
          {
            "id": "7s53l7lruhoqekxmw1expl588n4tkm4g",
            "createdOn": "Tue Jun 04 2019 17:28:34 GMT+0000 (UTC)",
            "updatedOn": null,
            "triggeredOn": null,
            "enabled": 1,
            "webhook": {
              "uri": "placeholder"
            },
            "email": null
          }
        ],
        "cluster": {
          "id": "lfiirfxu7ng9nkl6nj3f0h2k0njcppm2",
          "title": "ShipTime",
          "slug": "shiptime-1",
          "createdOn": "Fri May 24 2019 17:34:22 GMT+0000 (UTC)",
          "lastUpdated": "Fri May 24 2019 18:18:53 GMT+0000 (UTC)",
          "gitOpsRef": null,
          "shipOpsRef": {
            "token": "pzeom3j9p457yja71j4s0pji9cfmfyhk"
          }
        }
      }
    ]
  };

  it("renders without crashing", () => {
    const wrapper = shallow(<WatchSidebarItem watch={dummyWatch} />);

    expect(wrapper).toBeTruthy();
  });

  it("displays the correct version number if up to date", () => {
    const wrapper = shallow(<WatchSidebarItem watch={dummyWatch} />);


    expect(wrapper.find(".checkmark-icon")).toHaveLength(1);
    expect(wrapper.find(".u-color--dustyGray").text()).toBe("Up to date");
  });
});
