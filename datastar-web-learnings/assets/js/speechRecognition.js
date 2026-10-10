function toggleSpeechRecognition(instance) {
  let finalTranscript = "";
  if (instance) {
    try {
      instance.stop();
    } catch (error) {
      console.error("Error stopping speech recognition:", error);
    }
    return instance;
  }

  const SpeechRecognition =
    window.SpeechRecognition || window.webkitSpeechRecognition;
  if (!SpeechRecognition) {
    console.log("Speech recognition is not supported in this browser.");
    return "";
  }
  const recognition = new SpeechRecognition();
  recognition.continuous = true;
  recognition.interimResults = true;
  recognition.lang = "en-US";
  recognition.onstart = () => {
    window.dispatchEvent(new CustomEvent("speech-start"));
  };

  recognition.onerror = (event) => {
    console.error("Speech recognition error:", event.error);
    window.dispatchEvent(
      new CustomEvent("speech-error", { detail: { error: event.error } })
    );
  };
  recognition.onend = () => {
    // finalTranscript = "";
    window.dispatchEvent(new CustomEvent("speech-end"));
  };
  recognition.onresult = (event) => {
    let interimTranscript = "";
    for (let idx = event.resultIndex; idx < event.results.length; idx++) {
      const result = event.results[idx];
      if (result.isFinal) {
        finalTranscript += result[0].transcript.trim() + " ";
      } else {
        interimTranscript += result[0].transcript.trim() + " ";
      }
      window.dispatchEvent(
        new CustomEvent("speech-result", {
          detail: { value: (finalTranscript + " " + interimTranscript).trim() },
        })
      );
    }
  };
  recognition.start();
  return recognition;
}
