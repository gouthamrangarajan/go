let mediaRecorder;
let audioChunks = [];

async function setupRecorderAndStart() {
  document.dispatchEvent(
    new CustomEvent("recording-started", { bubbles: true })
  );
  let stream;
  try{
   stream = await navigator.mediaDevices.getUserMedia({ audio: true });
  } catch (err) {
    console.error("Error accessing microphone:", err);
    document.dispatchEvent(
      new CustomEvent("recording-access-error", {
        bubbles: true,
        detail: { error: err },
      })
    );
    return;
  }
  const mimeType = getSupportedMimeType();
  const options = mimeType ? { mimeType } : undefined;
  mediaRecorder = new MediaRecorder(stream, options);
  audioChunks = [];
  mediaRecorder.ondataavailable = (e) => {
    audioChunks.push(e.data);
  };
  mediaRecorder.addEventListener("stop", () => {
    let type = mediaRecorder.mimeType || "audio/webm";
    type = type.replaceAll("codecs=opus", "").trim(); // Remove codecs for Safari compatibility
    const audioBlob = new Blob(audioChunks, { type });
    const file = new File([audioBlob], "recording." + type.split("/")[1], {
      type,
    });
    // const previewURL = URL.createObjectURL(audioBlob);
    // window.open(previewURL, "_blank");
    /*
     * A file input's files property expects a FileList.
     * DataTransfer gives us a FileList containing our
     * programmatically created recording.
     */
    const transfer = new DataTransfer();
    transfer.items.add(file);
    document.dispatchEvent(
      new CustomEvent("recording-stopped", {
        bubbles: true,
        detail: {
          fileList: transfer.files,
        },
      })
    );
  });
  mediaRecorder.start();
  
}

async function stopRecording() {
  mediaRecorder.stop();
}

function getSupportedMimeType() {
  const types = [
    "audio/webm;codecs=opus", // Best for Chrome/Edge/Firefox (your target)
    "audio/webm",
    "audio/mp4", // Fallback for Safari
    "audio/wav", // Last resort, though rarely needed
  ];
  for (const type of types) {
    if (MediaRecorder.isTypeSupported(type)) {
      return type;
    }
  }
  return ""; // Let browser choose
}
