// use the getApps hook and react router
import {
  useParams
} from "react-router-dom";
import {
  useState,
  useEffect,
} from "react"
import {
  useApps
} from "../api/getApps";

function useCurrentApp() {
  let { slug } = useParams();
  let { data = {} , isFetched } = useApps();

  const { apps } = data;

  const [currentApp, setCurrentApp] = useState(null);

  useEffect(() => {
    if (apps && isFetched) {
      setCurrentApp(apps.find(app => app.slug === slug));
    }
  }, [apps, slug]);

  return { currentApp } ;
}



export { useCurrentApp };