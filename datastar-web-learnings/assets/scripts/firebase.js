import { initializeApp } from "https://www.gstatic.com/firebasejs/12.5.0/firebase-app.js";
import {
  getAuth,
  signInWithEmailAndPassword,
  onAuthStateChanged,
} from "https://www.gstatic.com/firebasejs/12.5.0/firebase-auth.js";

const config = {
  FIREBASE_API_KEY: "AIzaSyD7kPSuaLnsaYqYqZWULRSQlcOckwP8AJE",
  FIREBASE_AUTH_DOMAIN: "weblearnings-e679a.firebaseapp.com",
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
