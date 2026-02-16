import { initializeApp } from "https://www.gstatic.com/firebasejs/12.5.0/firebase-app.js";
import {
  getAuth,
  signInWithEmailAndPassword,
  onAuthStateChanged,
} from "https://www.gstatic.com/firebasejs/12.5.0/firebase-auth.js";

window.initializeFirebaseApp = function (config) {
  const firebaseApp = initializeApp({
    apiKey: config.apiKey,
    authDomain: config.authDomain,
  });
  window.AUTH = getAuth(firebaseApp);
  window.LOGIN = signInWithEmailAndPassword;
  onAuthStateChanged(window.AUTH, (user) => {
    window.dispatchEvent(new Event("firebase-loaded"));
  });
};
