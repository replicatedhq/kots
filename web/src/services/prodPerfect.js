!function(name,path,ctx){
  var latest,prev=name!=='Keen'&&window.Keen?window.Keen:false;ctx[name]=ctx[name]||{ready:function(fn){var h=document.getElementsByTagName('head')[0],s=document.createElement('script'),w=window,loaded;s.onload=s.onreadystatechange=function(){if((s.readyState&&!(/^c|loade/.test(s.readyState)))||loaded){return}s.onload=s.onreadystatechange=null;loaded=1;latest=w.Keen;if(prev){w.Keen=prev}else{try{delete w.Keen}catch(e){w.Keen=void 0}}ctx[name]=latest;ctx[name].ready(fn)};s.async=1;s.src=path;h.parentNode.insertBefore(s,h)}}
}('ProdPerfectKeen','https://replicated-ship-clusters.trackinglibrary.prodperfect.com/keen-tracking.min.js',this);

ProdPerfectKeen.ready(function(){
  var client = new ProdPerfectKeen({
    projectId: "P2rNLLysRcFQ7gBgBroraoim",
    writeKey: "@@PROD_PERFECT_WRITE_KEY",
    requestType: "beacon",
    host: "replicated-ship-clusters.datapipe.prodperfect.com/v1"
  });

  client.extendEvents({
    visitor: {
      user_id: null
    }
  });

  var options = {
    ignoreDisabledFormFields: false,
    recordClicks: true,
    recordFormSubmits: true,
    recordInputChanges: true,
    recordPageViews: true,
    recordPageUnloads: true,
    recordScrollState: true
  };
  client.initAutoTracking(options);
});
