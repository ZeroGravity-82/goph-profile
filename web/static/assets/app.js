const api = {
  health: "/health",
  resolveUser: "/api/v1/users/resolve",
  avatarByEmail: "/api/v1/avatar",
  avatar: "/api/v1/avatar",
  avatars: "/api/v1/avatars",
  userAvatars: userID => `/api/v1/users/${userID}/avatars`,
  avatarByID: avatarID => `/api/v1/avatars/${avatarID}`,
  avatarMetadata: avatarID => `/api/v1/avatars/${avatarID}/metadata`,
};

const state = {
  email: "",
  userID: "",
  polling: new Map(),
};

const elements = {
  healthIndicator: document.querySelector("#healthIndicator"),
  healthText: document.querySelector("#healthText"),
  identityForm: document.querySelector("#identityForm"),
  emailInput: document.querySelector("#emailInput"),
  profileArea: document.querySelector("#profileArea"),
  currentEmail: document.querySelector("#currentEmail"),
  currentAvatarImage: document.querySelector("#currentAvatarImage"),
  deleteCurrentButton: document.querySelector("#deleteCurrentButton"),
  uploadForm: document.querySelector("#uploadForm"),
  avatarFileInput: document.querySelector("#avatarFileInput"),
  fileDropText: document.querySelector("#fileDropText"),
  avatarListPanel: document.querySelector("#avatarListPanel"),
  avatarList: document.querySelector("#avatarList"),
  avatarListSummary: document.querySelector("#avatarListSummary"),
  refreshButton: document.querySelector("#refreshButton"),
  toast: document.querySelector("#toast"),
  avatarItemTemplate: document.querySelector("#avatarItemTemplate"),
};

document.addEventListener("DOMContentLoaded", () => {
  checkHealth();
  elements.identityForm.addEventListener("submit", resolveUser);
  elements.uploadForm.addEventListener("submit", uploadAvatar);
  elements.refreshButton.addEventListener("click", refreshProfile);
  elements.deleteCurrentButton.addEventListener("click", deleteCurrentAvatar);
  elements.avatarFileInput.addEventListener("change", updateSelectedFileName);
});

async function checkHealth() {
  try {
    const health = await requestJSON(api.health);
    const ok = health.status === "ok";
    elements.healthIndicator.classList.toggle("ok", ok);
    elements.healthIndicator.classList.toggle("error", !ok);
    elements.healthText.textContent = ok ? "готов" : "деградация";
  } catch {
    elements.healthIndicator.classList.add("error");
    elements.healthText.textContent = "недоступен";
  }
}

async function resolveUser(event) {
  event.preventDefault();
  const email = elements.emailInput.value.trim();
  if (!email) {
    showToast("Введите email");
    return;
  }

  setFormDisabled(elements.identityForm, true);
  try {
    const user = await requestJSON(api.resolveUser, {
      method: "POST",
      headers: {"Content-Type": "application/json"},
      body: JSON.stringify({email}),
    });

    state.email = user.email;
    state.userID = user.user_id;
    elements.profileArea.hidden = false;
    elements.avatarListPanel.hidden = false;
    elements.currentEmail.textContent = state.email;
    showToast("Профиль открыт");
    await refreshProfile();
  } catch (error) {
    showToast(error.message);
  } finally {
    setFormDisabled(elements.identityForm, false);
  }
}

async function refreshProfile() {
  if (!state.userID || !state.email) {
    return;
  }
  await Promise.all([refreshCurrentAvatar(), refreshAvatarList()]);
}

async function refreshCurrentAvatar() {
  const url = `${api.avatarByEmail}?email=${encodeURIComponent(state.email)}&v=${Date.now()}`;
  elements.currentAvatarImage.src = url;
}

async function refreshAvatarList() {
  try {
    const response = await requestJSON(api.userAvatars(state.userID));
    const avatars = response.avatars ?? [];
    renderAvatarList(avatars);
    elements.avatarListSummary.textContent = avatarListSummary(avatars);
    startPollingProcessingAvatars(avatars);
  } catch (error) {
    showToast(error.message);
  }
}

function renderAvatarList(avatars) {
  elements.avatarList.replaceChildren();
  if (avatars.length === 0) {
    const empty = document.createElement("p");
    empty.className = "muted";
    empty.textContent = "Загруженных аватарок нет.";
    elements.avatarList.append(empty);
    return;
  }

  for (const avatar of avatars) {
    elements.avatarList.append(newAvatarItem(avatar));
  }
}

function newAvatarItem(avatar) {
  const item = elements.avatarItemTemplate.content.firstElementChild.cloneNode(true);
  const image = item.querySelector(".avatar-thumb");
  const title = item.querySelector("h3");
  const details = item.querySelector(".avatar-details");
  const badge = item.querySelector(".status-badge");
  const selectButton = item.querySelector(".select-avatar");
  const deleteButton = item.querySelector(".delete-avatar");

  image.src = avatar.status === "ready"
    ? `${avatar.url}?size=100x100&format=png&v=${Date.now()}`
    : "/default-avatar-100.png";
  image.alt = `Аватарка ${avatar.file_name}`;
  title.textContent = avatar.file_name;
  badge.textContent = avatarStatusText(avatar.status, avatar.is_current);
  badge.classList.add(avatar.status);
  details.textContent = avatarDetails(avatar);

  selectButton.disabled = avatar.status !== "ready" || avatar.is_current;
  selectButton.addEventListener("click", () => selectCurrentAvatar(avatar.id));
  deleteButton.addEventListener("click", () => deleteAvatar(avatar.id));

  return item;
}

function avatarListSummary(avatars) {
  const total = avatars.length;
  const ready = avatars.filter(avatar => avatar.status === "ready").length;
  return `Всего: ${total}, готовы: ${ready}`;
}

function avatarStatusText(status, isCurrent) {
  if (isCurrent) {
    return "текущая";
  }
  switch (status) {
  case "processing":
    return "обработка";
  case "ready":
    return "готова";
  case "failed":
    return "ошибка";
  default:
    return status;
  }
}

function avatarDetails(avatar) {
  const dimensions = avatar.width && avatar.height
    ? `${avatar.width}x${avatar.height}`
    : "размеры пока неизвестны";
  return `${avatar.mime_type}, ${formatBytes(avatar.size_bytes)}, ${dimensions}`;
}

function formatBytes(value) {
  if (!Number.isFinite(value)) {
    return "0 Б";
  }
  if (value < 1024) {
    return `${value} Б`;
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} КиБ`;
  }
  return `${(value / 1024 / 1024).toFixed(1)} МиБ`;
}

async function uploadAvatar(event) {
  event.preventDefault();
  const file = elements.avatarFileInput.files[0];
  if (!state.userID) {
    showToast("Сначала укажите email");
    return;
  }
  if (!file) {
    showToast("Выберите файл");
    return;
  }

  const form = new FormData();
  form.append("file", file);
  setFormDisabled(elements.uploadForm, true);
  try {
    const avatar = await requestJSON(api.avatars, {
      method: "POST",
      headers: {"X-User-ID": state.userID},
      body: form,
    });

    elements.avatarFileInput.value = "";
    updateSelectedFileName();
    showToast("Файл загружен, идет обработка");
    await refreshAvatarList();
    pollAvatarUntilDone(avatar.id);
  } catch (error) {
    showToast(error.message);
  } finally {
    setFormDisabled(elements.uploadForm, false);
  }
}

function startPollingProcessingAvatars(avatars) {
  for (const avatar of avatars) {
    if (avatar.status === "processing") {
      pollAvatarUntilDone(avatar.id);
    }
  }
}

function pollAvatarUntilDone(avatarID) {
  if (state.polling.has(avatarID)) {
    return;
  }

  let attempts = 0;
  const timer = window.setInterval(async () => {
    attempts += 1;
    try {
      const metadata = await requestJSON(api.avatarMetadata(avatarID));
      if (metadata.status === "ready" || metadata.status === "failed" || metadata.status === "deleted") {
        stopPolling(avatarID);
        await refreshProfile();
        showToast(metadata.status === "ready" ? "Аватарка готова" : "Обработка завершилась ошибкой");
      }
    } catch (error) {
      stopPolling(avatarID);
      showToast(error.message);
    }

    if (attempts >= 60) {
      stopPolling(avatarID);
    }
  }, 1000);

  state.polling.set(avatarID, timer);
}

function stopPolling(avatarID) {
  const timer = state.polling.get(avatarID);
  if (timer) {
    window.clearInterval(timer);
    state.polling.delete(avatarID);
  }
}

async function selectCurrentAvatar(avatarID) {
  try {
    await requestNoContent(api.avatar, {
      method: "PATCH",
      headers: {
        "Content-Type": "application/json",
        "X-User-ID": state.userID,
      },
      body: JSON.stringify({avatar_id: avatarID}),
    });
    showToast("Текущая аватарка обновлена");
    await refreshProfile();
  } catch (error) {
    showToast(error.message);
  }
}

async function deleteCurrentAvatar() {
  if (!state.userID) {
    return;
  }
  try {
    await requestNoContent(api.avatar, {
      method: "DELETE",
      headers: {"X-User-ID": state.userID},
    });
    showToast("Текущая аватарка удаляется");
    await refreshProfile();
  } catch (error) {
    showToast(error.message);
  }
}

async function deleteAvatar(avatarID) {
  try {
    await requestNoContent(api.avatarByID(avatarID), {
      method: "DELETE",
      headers: {"X-User-ID": state.userID},
    });
    showToast("Аватарка удаляется");
    await refreshProfile();
  } catch (error) {
    showToast(error.message);
  }
}

async function requestJSON(url, options = {}) {
  const response = await fetch(url, options);
  if (!response.ok) {
    throw new Error(await errorMessage(response));
  }
  return response.json();
}

async function requestNoContent(url, options = {}) {
  const response = await fetch(url, options);
  if (!response.ok) {
    throw new Error(await errorMessage(response));
  }
}

async function errorMessage(response) {
  try {
    const body = await response.json();
    return body.details ? `${body.error}: ${body.details}` : body.error;
  } catch {
    return `HTTP ${response.status}`;
  }
}

function setFormDisabled(form, disabled) {
  for (const element of form.elements) {
    element.disabled = disabled;
  }
}

function updateSelectedFileName() {
  const file = elements.avatarFileInput.files[0];
  elements.fileDropText.textContent = file ? file.name : "Выберите JPEG, PNG или WebP до 10 МиБ";
}

function showToast(message) {
  elements.toast.textContent = message;
  elements.toast.hidden = false;
  window.clearTimeout(showToast.timeoutID);
  showToast.timeoutID = window.setTimeout(() => {
    elements.toast.hidden = true;
  }, 3200);
}
