import { initializeApp } from "https://www.gstatic.com/firebasejs/12.5.0/firebase-app.js";
import {
  getAuth,
  signInWithEmailAndPassword,
  onAuthStateChanged,
} from "https://www.gstatic.com/firebasejs/12.5.0/firebase-auth.js";

fetch("/config")
  .then((resp) => resp.json())
  .then((response) => {
    const config = {
      FIREBASE_API_KEY: response.firebaseApiKey,
      FIREBASE_AUTH_DOMAIN: response.firebaseAuthDomain,
    };

    const firebaseApp = initializeApp({
      apiKey: config.FIREBASE_API_KEY,
      authDomain: config.FIREBASE_AUTH_DOMAIN,
    });
    window.AUTH = getAuth(firebaseApp);
    window.LOGIN = signInWithEmailAndPassword;
    onAuthStateChanged(window.AUTH, (user) => {
      window.dispatchEvent(new Event("firebase-loaded"));
    });
  });
