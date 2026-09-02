const DISMISSED = "dismissed";

export function quickStartKey(user) {
  return `order-fill:quick-start:${user.id}`;
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
