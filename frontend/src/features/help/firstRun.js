const DISMISSED = "dismissed";

const SCREEN_SCENES = new Set(["users", "company", "companies", "account"]);

export function quickStartKey(user) {
  return `order-fill:quick-start:${user.id}`;
}

export function shouldAutoStartTour(reason) {
  return reason === "invite";
}

export function tourSceneForView(view = {}) {
  if (view.screen === "order") {
    if (view.stage === "fill") return "order-fill";
    if (view.stage === "preview") return "order-preview";
    return "order-upload";
  }
  if (view.screen === "north") return "north";
  if (SCREEN_SCENES.has(view.screen)) return view.screen;
  if (view.screen === "overview" && (view.seenHome || view.replay)) return "overview";
  return "home";
}

export function shouldShowQuickStart(user, storage = localStorage) {
  if (!user?.id) return false;
  try {
    return storage.getItem(quickStartKey(user)) !== DISMISSED;
  } catch {
    return true;
  }
}

export function dismissQuickStart(user, storage = localStorage) {
  if (!user?.id) return;
  try {
    storage.setItem(quickStartKey(user), DISMISSED);
  } catch {
    // Preference storage is optional; the session can still hide the dialog.
  }
}

export function consumeQuickStart(user, storage = localStorage) {
  if (!shouldShowQuickStart(user, storage)) return false;
  dismissQuickStart(user, storage);
  return true;
}
