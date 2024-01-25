import { useEffect, useReducer } from "react";
import fetch from "../../utilities/fetchWithTimeout";
import { Utilities } from "../../utilities/utilities";
import Loader from "@components/shared/Loader";

interface Props {
  setTerminatedState: (terminated: boolean) => void;
}

interface State {
  seconds: number;
  reconnectAttempts: number;
}

const EmbeddedClusterUpgrading = (props: Props) => {
  const [state, setState] = useReducer(
    (currentState: State, newState: Partial<State>) => ({
      ...currentState,
      ...newState,
    }),
    {
      seconds: 1,
      reconnectAttempts: 1,
    }
  );

  let countdown: (seconds: number) => void;
  let ping: () => Promise<void>;

  countdown = (seconds: number) => {
    setState({ seconds });
    if (seconds === 0) {
      setState({
        reconnectAttempts: state.reconnectAttempts + 1,
      });
      ping();
    } else {
      const nextCount = seconds - 1;
      setTimeout(() => {
        countdown(nextCount);
      }, 1000);
    }
  };

  ping = async () => {
    const { reconnectAttempts } = state;
    await fetch(
      `${process.env.API_ENDPOINT}/ping`,
      {
        headers: {
          "Content-Type": "application/json",
        },
        credentials: "include",
      },
      10000
    )
      .then(async (res) => {
        if (res.status === 401) {
          Utilities.logoutUser();
          return;
        }
        props.setTerminatedState(false);
      })
      .catch(() => {
        props.setTerminatedState(true);
        const seconds = reconnectAttempts > 10 ? 10 : reconnectAttempts;
        countdown(seconds);
      });
  };

  useEffect(() => {
    ping();
  }, []);

  return (
    <div className="Modal-body u-textAlign--center">
      <div className="flex u-marginTop--30 u-marginBottom--10 justifyContent--center">
        <Loader size="60" />
      </div>
      <h2 className="u-fontSize--largest u-textColor--primary u-fontWeight--bold u-lineHeight--normal u-userSelect--none">
        Cluster update in progress
      </h2>
      <p className="u-fontSize--normal u-fontWeight--medium u-textColor--bodyCopy u-lineHeight--more u-marginTop--10 u-marginBottom--10 u-userSelect--none">
        The API cannot be reached because the cluster is updating. Stay on this
        page to automatically reconnect when the update is complete.
      </p>
    </div>
  );
};

export default EmbeddedClusterUpgrading;
