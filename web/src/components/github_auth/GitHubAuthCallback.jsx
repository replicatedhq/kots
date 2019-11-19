import * as React from "react";
import { withRouter } from "react-router-dom";
import { graphql, compose, withApollo } from "react-apollo";
import { Link } from "react-router-dom";
import { createGithubAuthToken } from "../../mutations/GitHubMutations";
import { userInfo } from "../../queries/UserQueries";
import { Utilities } from "../../utilities/utilities";

class GitHubAuthCallback extends React.Component {
  constructor(props) {
    super(props);
    this.state = {
      isLoading: true,
    };
  }

  getUser = async () => {
    await this.props.client.query({ query: userInfo })
      .then(() => {
        if (Utilities.localStorageEnabled()) {
          const next = localStorage.getItem("next");
          if (next) {
            localStorage.removeItem("next");
            return this.props.history.push(next);
          }
        } else {
          return this.props.history.push("/unsupported");
        }
        this.props.history.push("/apps");
      })
      .catch();
  }

  async componentDidMount() {
    const { search } = this.props.location;
    const queryParams = new URLSearchParams(search);
    const code = queryParams.get("code");
    const state = queryParams.get("state");
    if (code && state) {
      await this.props.createGithubAuthToken(code, state)
        .then((res) => {
          if (Utilities.localStorageEnabled()) {
            window.localStorage.setItem("token", res.data.createGithubAuthToken.access_token);
            this.props.refetchListApps().then(this.getUser);
            // this.getUser();

          } else {
            this.props.history.push("/unsupported");
          }
        })
        .catch((err) => {
          err.graphQLErrors.map(({ code }) => {
            if (code === "Ship Cloud Access Denied") {
              this.props.history.push("/login");
            } else {
              this.setState({ isLoading: false });
            }
          });
          return;
        });
    }
  }

  render() {
    return (
      <div className="container flex-column flex1">
        {this.state.isLoading ?
          <div className="flex-column flex1 u-overflow--auto justifyContent--center alignItems--center">
            <div className="u-marginBottom--20">
              <svg width="84px" height="84px" viewBox="0 0 84 84" version="1.1" xmlns="http://www.w3.org/2000/svg">
                <g stroke="none" strokeWidth="1" fill="none" fillRule="evenodd">
                  <g transform="translate(2.000000, 2.000000)">
                    <path d="M69.8571282,22.4176864 C66.7735842,17.2185494 62.5909538,13.1023854 57.308136,10.0682659 C52.0245319,7.03399163 46.2564113,5.51724138 39.9996855,5.51724138 C33.743746,5.51724138 27.9737383,7.03445589 22.691235,10.0682659 C17.4079454,13.1022307 13.2254722,17.2185494 10.1419283,22.4176864 C7.05885609,27.6166686 5.51724138,33.2939974 5.51724138,39.449518 C5.51724138,46.8436948 7.70943857,53.492728 12.0949338,59.39832 C16.4799572,65.3043761 22.1447583,69.3911367 29.0888653,71.659066 C29.897179,71.8067019 30.4955514,71.7028616 30.8846113,71.3501757 C31.2738285,70.9970256 31.4682013,70.5547369 31.4682013,70.0251665 C31.4682013,69.9368016 31.4604956,69.141827 31.4455559,67.6393142 C31.4301445,66.1368013 31.4229105,64.8260295 31.4229105,63.7076177 L30.3901875,63.883419 C29.7317421,64.0021158 28.9010975,64.0524111 27.8982537,64.0381736 C26.8958818,64.0244005 25.8552958,63.9210244 24.7779111,63.7288191 C23.7000546,63.5383161 22.6975254,63.0964916 21.7695372,62.4042741 C20.8420207,61.7120566 20.1835753,60.8059682 19.7943581,59.6874016 L19.3453822,58.6706636 C19.0461174,57.9937668 18.574968,57.241814 17.931305,56.4175908 C17.2876419,55.5925938 16.6367449,55.0333105 15.9782994,54.7388124 L15.6639377,54.5173585 C15.4544681,54.3701869 15.2600954,54.1926833 15.0803478,53.9867049 C14.9007575,53.7807264 14.7663006,53.5745933 14.6765054,53.3681506 C14.586553,53.1615531 14.661094,52.9920968 14.9009147,52.8591625 C15.1407354,52.7262283 15.5741426,52.6616956 16.2030233,52.6616956 L17.1006604,52.7937013 C17.6993472,52.9117791 18.4398821,53.264465 19.3232087,53.8539254 C20.2060634,54.4429216 20.9318159,55.2086475 21.5006235,56.1507938 C22.18942,57.3588085 23.0192783,58.2792891 23.9925572,58.9126999 C24.9650499,59.5461106 25.9455627,59.8622744 26.9331523,59.8622744 C27.9207418,59.8622744 28.7737173,59.7886112 29.4923931,59.6420585 C30.2102827,59.4947321 30.883825,59.2732782 31.5127057,58.9789349 C31.7820912,57.0045751 32.5155494,55.4878249 33.7124513,54.4276008 C32.0065005,54.2511805 30.4727487,53.9854668 29.1104098,53.632162 C27.7488572,53.2783929 26.3418566,52.7042531 24.8903516,51.9081953 C23.4380603,51.1132207 22.2332954,50.1260408 21.2757424,48.948358 C20.3180321,47.7700561 19.5320492,46.2231287 18.9188945,44.3089686 C18.3054253,42.3940346 17.998612,40.1850668 17.998612,37.6814462 C17.998612,34.1166731 19.1812033,31.0831726 21.5459142,28.5792424 C20.4381784,25.8992015 20.5427559,22.8947949 21.8599614,19.5663321 C22.7280337,19.3009279 24.0153598,19.5000971 25.7213107,20.1626017 C27.4275761,20.8254158 28.6768454,21.3932106 29.4703768,21.8639742 C30.2639081,22.3345831 30.8997082,22.7333858 31.3787206,23.056823 C34.1629998,22.2912518 37.0362878,21.9083888 39.999371,21.9083888 C42.9624541,21.9083888 45.8363712,22.2912518 48.6208076,23.056823 L50.3269157,21.9969085 C51.4936238,21.2896798 52.8713742,20.6415673 54.4568643,20.0524164 C56.0432979,19.463575 57.2563976,19.3013921 58.0949052,19.5667963 C59.4413608,22.8954139 59.5613498,25.8996658 58.4532995,28.5797067 C60.8178531,31.0836368 62.0009162,34.1179111 62.0009162,37.6819105 C62.0009162,40.1855311 61.6930021,42.4014628 61.0804764,44.3310985 C60.4671645,46.2610436 59.6744194,47.8064235 58.7019268,48.9704879 C57.7283333,50.1343976 56.5158627,51.1138397 55.0643577,51.9086595 C53.6125381,52.7040984 52.2050657,53.2782381 50.8435131,53.6320072 C49.4813315,53.9857763 47.9475798,54.2516448 46.2416289,54.4283746 C47.7975542,55.7533839 48.5756741,57.8448929 48.5756741,60.7019731 L48.5756741,70.0239285 C48.5756741,70.5534988 48.7628129,70.9956328 49.137405,71.3489377 C49.5115253,71.7016235 50.1023492,71.8054639 50.9106629,71.6576732 C57.8557135,69.3900534 63.5205146,65.3031381 67.9053807,59.3970819 C72.2897751,53.49149 74.4827586,46.8424568 74.4827586,39.44828 C74.481186,33.2935331 72.938785,27.6166686 69.8571282,22.4176864 Z" fill="#24292E" fillRule="nonzero"></path>
                    <g stroke="#24292E" strokeWidth="3">
                      <circle strokeOpacity="0.2" cx="40" cy="40" r="40"></circle>
                      <path d="M80,40 C80,17.9111111 62.0888889,0 40,0">
                        <animateTransform
                          attributeName="transform"
                          type="rotate"
                          from="0 40 40"
                          to="360 40 40"
                          dur=".6s"
                          repeatCount="indefinite"/></path>
                    </g>
                  </g>
                </g>
              </svg>
            </div>
            <p className="u-fontSize--large u-fontWeight--bold u-marginTop--normal u-color--tuna">Authorizing GitHub</p>
          </div>
          :
          <div className="flex-column flex1 u-overflow--auto justifyContent--center alignItems--center">
            <p className="u-fontSize--medium u-color--chestnut u-fontWeight--medium u-marginLeft--normal flex alignItems--center u-lineHeight--normal">
                Unable to connect to your GitHub, please try logging in again
            </p>
            <Link data-prevent-navigate className="u-marginTop--30 btn primary" to={`/login`}>Back to log in</Link>
          </div>
        }
      </div>
    );
  }
}

export default compose(
  withApollo,
  withRouter,
  graphql(createGithubAuthToken, {
    props: ({ mutate }) => ({
      createGithubAuthToken: (code, state) => mutate({ variables: { code, state }})
    })
  })
)(GitHubAuthCallback);
