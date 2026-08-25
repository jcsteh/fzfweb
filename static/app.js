"use strict";

const queryInput = document.getElementById("query");
const statusRegion = document.getElementById("status");
const matchesList = document.getElementById("matches");
const selectedList = document.getElementById("selected-list");
const filterForm = document.getElementById("filter-form");
const submitForm = document.getElementById("submit-form");
const btnSelectAll = document.getElementById("btn-select-all");
const btnDeselectAll = document.getElementById("btn-deselect-all");
const btnToggleAll = document.getElementById("btn-toggle-all");
const btnPush = document.getElementById("btn-push");
const btnCancel = document.getElementById("btn-cancel");

let seq = 0;
let matches = [];
let total = 0;
const selected = new Set();
let debounceTimer = null;
let lastFilteredQuery = null;

function announce(text) {
  statusRegion.textContent = text;
}

function scheduleFilter() {
  clearTimeout(debounceTimer);
  debounceTimer = setTimeout(runFilter, 250);
}

async function runFilter() {
  clearTimeout(debounceTimer);
  const mySeq = ++seq;
  const query = queryInput.value;

  const res = await fetch("/filter", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query, seq: mySeq }),
  });
  const data = await res.json();

  if (data.seq !== seq) {
    return; // A newer request landed first; this one is stale.
  }

  matches = data.matches;
  total = data.total;
  lastFilteredQuery = query;
  renderMatches();
  announce(`${matches.length} of ${total} items match.`);
}

async function flushFilter() {
  clearTimeout(debounceTimer);
  if (queryInput.value === lastFilteredQuery) {
    return;
  }
  await runFilter();
}

function renderMatches() {
  matchesList.textContent = "";
  const frag = document.createDocumentFragment();

  matches.forEach((item, i) => {
    const li = document.createElement("li");
    const id = "item-" + i;

    const checkbox = document.createElement("input");
    checkbox.type = "checkbox";
    checkbox.id = id;
    checkbox.value = item;
    checkbox.checked = selected.has(item);
    checkbox.addEventListener("change", () => {
      if (checkbox.checked) {
        selected.add(item);
      } else {
        selected.delete(item);
      }
      updateSelectedUI();
    });

    const label = document.createElement("label");
    label.htmlFor = id;
    label.textContent = item;

    li.appendChild(checkbox);
    li.appendChild(label);
    frag.appendChild(li);
  });

  matchesList.appendChild(frag);
  updateSelectedUI();
}

function syncCheckboxes() {
  matches.forEach((item, i) => {
    const checkbox = document.getElementById("item-" + i);
    if (checkbox) {
      checkbox.checked = selected.has(item);
    }
  });
}

function updateSelectedUI() {
  selectedList.textContent = "";
  const frag = document.createDocumentFragment();

  for (const item of selected) {
    const li = document.createElement("li");
    li.className = "selection-item";

    const span = document.createElement("span");
    span.className = "selection-label";
    span.textContent = item;

    const remove = document.createElement("button");
    remove.type = "button";
    remove.className = "selection-remove";
    remove.setAttribute("aria-label", "Remove " + item);
    remove.textContent = "×";
    remove.addEventListener("click", () => {
      selected.delete(item);
      updateSelectedUI();
      syncCheckboxes();
    });

    li.appendChild(span);
    li.appendChild(remove);
    frag.appendChild(li);
  }

  selectedList.appendChild(frag);
  btnPush.textContent = `Push ${selected.size} item${
    selected.size === 1 ? "" : "s"
  }`;
  btnDeselectAll.textContent = `Deselect all (${selected.size} selected)`;
}

queryInput.addEventListener("input", scheduleFilter);
queryInput.addEventListener("keydown", e => {
  if (e.key === "Escape" && queryInput.value !== "") {
    queryInput.value = "";
    runFilter();
  }
});

filterForm.addEventListener("submit", e => {
  e.preventDefault();
  flushFilter();
});

btnSelectAll.addEventListener("click", async () => {
  await flushFilter();
  matches.forEach(item => selected.add(item));
  syncCheckboxes();
  updateSelectedUI();
  btnSelectAll.ariaNotify(`Selected ${matches.length} matching items.`);
});

btnDeselectAll.addEventListener("click", async () => {
  await flushFilter();
  selected.clear();
  syncCheckboxes();
  updateSelectedUI();
});

btnToggleAll.addEventListener("click", async () => {
  await flushFilter();
  matches.forEach(item => {
    if (selected.has(item)) {
      selected.delete(item);
    } else {
      selected.add(item);
    }
  });
  syncCheckboxes();
  updateSelectedUI();
  btnToggleAll.ariaNotify(`Toggled ${matches.length} matching items.`);
});

async function doSubmit(action) {
  await flushFilter();
  await fetch("/submit", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      action,
      query: queryInput.value,
      selected: Array.from(selected),
    }),
  });
  document.body.innerHTML = "<p>You may now close this page.</p>";
}

submitForm.addEventListener("submit", e => {
  e.preventDefault();
  doSubmit("push");
});

btnCancel.addEventListener("click", () => {
  doSubmit("cancel");
});

queryInput.focus();
runFilter();
