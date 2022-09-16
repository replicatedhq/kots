// const loginWithSharedPassword = async (password: string) => {
//     fetch(`${process.env.API_ENDPOINT}/login`, {
//       headers: {
//         "Content-Type": "application/json",
//       },
//       method: "POST",
//       body: JSON.stringify({
//         password,
//       }),
//     })
//       .then(async (res) => {
//         if (res.status >= 400) {
//           let body = await res.json();
//           let msg = body.error;
//           if (!msg) {
//             msg =
//               res.status === 401
//                 ? "Invalid password. Please try again"
//                 : "There was an error logging in. Please try again.";
//           }
//           return;
//         }
//         this.completeLogin(await res.json());
//       })
//       .catch((err) => {
//         console.log("Login failed:", err);
//         this.setState({
//           authLoading: false,
//           loginErr: true,
//           loginErrMessage: "There was an error logging in. Please try again",
//         });
//       });
// };

// const loginWithIdentityProvider = async () => {
//   try {
//     this.setState({ loginErr: false, loginErrMessage: "" });

//     const res = await fetch(`${process.env.API_ENDPOINT}/oidc/login`, {
//       headers: {
//         "Content-Type": "application/json",
//       },
//       method: "GET",
//     });

//     if (res.status >= 400) {
//       const body = await res.json();
//       let msg = body.error;
//       if (!msg) {
//         msg = "There was an error logging in. Please try again.";
//       }
//       this.setState({
//         loginErr: true,
//         loginErrMessage: msg,
//       });
//       return;
//     }

//     const body = await res.json();
//     window.location = body.authCodeURL;
//   } catch (err) {
//     console.log("Login failed:", err);
//     this.setState({
//       loginErr: true,
//       loginErrMessage: "There was an error logging in. Please try again",
//     });
//   }
// };

// export { loginWithIdentityProvider, loginWithSharedPassword };