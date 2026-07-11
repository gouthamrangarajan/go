let tag = document.createElement("script");
// tag.src = "https://www.youtube.com/player_api";
tag.src = "https://www.youtube.com/iframe_api";
let firstScriptTag = document.getElementsByTagName("script")[0];
firstScriptTag.parentNode.insertBefore(tag, firstScriptTag);

async function initYTPlayer(vId, retryIdx) {
  if (!window.YT || !window.YT.Player) {
    if (retryIdx > 10) {
      console.error("YT Player API failed to load.");
      return;
    }
    console.log(
      `YT Player API not ready yet, retrying in ${retryIdx * 100}ms...`
    );
    await new Promise((resolve) => {
      setTimeout(resolve, (retryIdx + 1) * 100);
    });
    return initYTPlayer(vId, retryIdx + 1);
  }
  const player = new YT.Player("player_" + vId, {
    height: "100%",
    width: "100%",
    videoId: vId,
    playerVars: {
      playsinline: 1,
    },
    events: {
      onReady: function (event) {
          // console.log(player);
      },
      onStateChange: function (event) {
        if (event.data == YT.PlayerState.PLAYING) {
          player.g.dispatchEvent(
            new CustomEvent("inline-video-playing", {
              bubbles: true,
              detail: { videoId: vId },
            })
          );
        }
        // else if(event.data == YT.PlayerState.ENDED || event.data == YT.PlayerState.PAUSED){
        //     player.g.dispatchEvent(new CustomEvent('inline-video-stopped',{bubbles:true,detail:{videoId:vId}}));
        // }
      },
    },
  });
  console.log(player);
  return player;
}
function initDockedYTPlayer(vId, dockedPlayerSignal, durationToPlaySignal) {
  if (dockedPlayerSignal && dockedPlayerSignal.loadVideoById) {
    dockedPlayerSignal.loadVideoById(vId);
    setTimeout(() => {
      if (durationToPlaySignal) {
        dockedPlayerSignal.seekTo(durationToPlaySignal);
      }
      dockedPlayerSignal.playVideo();
    }, 500);
    return dockedPlayerSignal;
  }
  const player = new YT.Player("docker_player", {
    height: "100%",
    width: "100%",
    videoId: vId,
    playerVars: {
      playsinline: 1,
    },
    events: {
      onReady: function (event) {
        setTimeout(() => {
          if (durationToPlaySignal) {
            player.seekTo(durationToPlaySignal);
          }
          player.playVideo();
        }, 500);
      },
      onStateChange: function (event) {
        if (event.data == YT.PlayerState.PLAYING) {
          player.g.dispatchEvent(
            new CustomEvent("docked-video-playing", {
              bubbles: true,
              detail: { videoId: vId },
            })
          );
        } else if (
          event.data == YT.PlayerState.ENDED ||
          event.data == YT.PlayerState.PAUSED
        ) {
          player.g.dispatchEvent(
            new CustomEvent("docked-video-stopped", {
              bubbles: true,
              detail: { videoId: vId },
            })
          );
        }
      },
    },
  });
  return player;
}
