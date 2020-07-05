"igor.subs 'garage-door-trigger','test','force'";

var init = {
  door1Triggered: false,
  door2Triggered: false,
};

function reducer(state, event) {
  if (Object.keys(state).length === 0) {
    state = init;
  }

  state.triggered = event.payload;

  return state;
}

reducer(input.state, input.event);
