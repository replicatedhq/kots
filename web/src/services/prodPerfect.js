// TODO: fix an enable eslint on this file
/* eslint-disable */
!(function (name, path, ctx) {
  let latest;
  const prev = name !== "Keen" && window.Keen ? window.Keen : false;
  ctx[name] = ctx[name] || {
    ready(fn) {
      const h = document.getElementsByTagName("head")[0];
      const s = document.createElement("script");
      const w = window;
      let loaded;
      s.onload = s.onreadystatechange = function () {
        if ((s.readyState && !/^c|loade/.test(s.readyState)) || loaded) {
          return;
        }
        s.onload = s.onreadystatechange = null;
        loaded = 1;
        latest = w.Keen;
        if (prev) {
          w.Keen = prev;
        } else {
          try {
            delete w.Keen;
          } catch (e) {
            w.Keen = void 0;
          }
        }
        ctx[name] = latest;
        ctx[name].ready(fn);
      };
      s.async = 1;
      s.src = path;
      h.parentNode.insertBefore(s, h);
    },
  };
})(
  "ProdPerfectKeen",
  "https://replicated-ship-clusters.trackinglibrary.prodperfect.com/keen-tracking.min.js",
  this
);

ProdPerfectKeen.ready(() => {
  const client = new ProdPerfectKeen({
    projectId: "P2rNLLysRcFQ7gBgBroraoim",
    writeKey: "@@PROD_PERFECT_WRITE_KEY",
    requestType: "beacon",
    host: "replicated-ship-clusters.datapipe.prodperfect.com/v1",
  });

  client.extendEvents({
    visitor: {
      user_id: null,
    },
  });

  const options = {
    ignoreDisabledFormFields: false,
    recordClicks: true,
    recordFormSubmits: true,
    recordInputChanges: true,
    recordPageViews: true,
    recordPageUnloads: true,
    recordScrollState: true,
  };
  client.initAutoTracking(options);
});
