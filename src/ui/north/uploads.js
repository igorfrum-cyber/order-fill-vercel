

export function createNorthUploads(context) {
  const {elements, state, actions} = context;
function resetNorthCalculationState() {
  state.currentNorthResult = null;
  state.northPlanEdits = new Map();
  actions.clearNorthDownloadLinks();
  elements.northResult.classList.add("hidden");
  elements.northStatus.textContent = "Готов к загрузке";
}

function renderNorthFileList(target, files, group) {
  target.innerHTML = files
    .map((file, index) => `
      <div class="north-file-item">
        <span>${actions.escapeHtml(file.name)}</span>
        <button type="button" data-remove-north-file="${index}" data-north-file-group="${actions.escapeHtml(group)}" aria-label="Удалить ${actions.escapeHtml(file.name)}">×</button>
      </div>
    `)
    .join("");
}

function sameNorthFile(left, right) {
  return left.name === right.name && left.size === right.size && left.lastModified === right.lastModified;
}

function pendingNorthFiles(input) {
  return Array.from(input.files || []);
}

function uniqueNorthFiles(files) {
  const unique = [];
  for (const file of files) {
    if (!unique.some((item) => sameNorthFile(item, file))) unique.push(file);
  }
  return unique;
}

function northFilesForMerge() {
  if (isNorthChristinaMode()) {
    return [
      ...uniqueNorthFiles(state.addedNorthHomeFiles).map((file) => ({ file, variant: "home", variantLabel: "HOME" })),
      ...uniqueNorthFiles(state.addedNorthProffFiles).map((file) => ({ file, variant: "proff", variantLabel: "PROFF" })),
    ];
  }
  return uniqueNorthFiles(state.addedNorthFiles).map((file) => ({ file }));
}

function addPendingNorthFiles({ input, names, placeholder, files, list, group }) {
  const pending = pendingNorthFiles(input);
  if (!pending.length) return;
  let addedCount = 0;
  for (const file of pending) {
    const alreadyAdded = files.some((item) => sameNorthFile(item, file));
    if (alreadyAdded) continue;
    files.push(file);
    addedCount += 1;
  }
  if (!addedCount) {
    alert("Все выбранные бланки уже добавлены.");
    input.value = "";
    names.textContent = placeholder;
    return;
  }
  input.value = "";
  names.textContent = placeholder;
  renderNorthFileList(list, files, group);
  resetNorthCalculationState();
}

function removeNorthFile(group, index) {
  if (group === "home") {
    state.addedNorthHomeFiles.splice(index, 1);
    renderNorthFileList(elements.northHomeFileList, state.addedNorthHomeFiles, "home");
  } else if (group === "proff") {
    state.addedNorthProffFiles.splice(index, 1);
    renderNorthFileList(elements.northProffFileList, state.addedNorthProffFiles, "proff");
  } else {
    state.addedNorthFiles.splice(index, 1);
    renderNorthFileList(elements.northFileList, state.addedNorthFiles, "default");
  }
  resetNorthCalculationState();
}

function handleNorthFileListClick(event) {
  const button = event.target.closest("[data-remove-north-file]");
  if (!button) return;
  removeNorthFile(button.dataset.northFileGroup || "default", Number(button.dataset.removeNorthFile));
}

function isNorthChristinaMode() {
  return actions.selectedNorthBrand() === "christina";
}

function resetNorthUploadedFiles() {
  state.addedNorthFiles = [];
  state.addedNorthHomeFiles = [];
  state.addedNorthProffFiles = [];
  elements.northFileInput.value = "";
  elements.northHomeInput.value = "";
  elements.northProffInput.value = "";
  elements.northNames.textContent = "Выберите один или несколько заполненных бланков";
  elements.northHomeNames.textContent = "Выберите HOME-бланки городов";
  elements.northProffNames.textContent = "Выберите PROFF-бланки городов";
  renderNorthFileList(elements.northFileList, state.addedNorthFiles, "default");
  renderNorthFileList(elements.northHomeFileList, state.addedNorthHomeFiles, "home");
  renderNorthFileList(elements.northProffFileList, state.addedNorthProffFiles, "proff");
}
  return { resetNorthCalculationState, renderNorthFileList, sameNorthFile, pendingNorthFiles, uniqueNorthFiles, northFilesForMerge, addPendingNorthFiles, removeNorthFile, handleNorthFileListClick, isNorthChristinaMode, resetNorthUploadedFiles };
}
